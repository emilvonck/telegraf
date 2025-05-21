package netbox

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/processors"
)

type NetboxProcessor struct {
	GraphQLQuery string `toml:"graphql_query"`
	Url          string `toml:"url"`
	Token        string `toml:"token"`
}

type GraphQLRequest struct {
	Query     string            `json:"query"`
	Variables map[string]string `json:"variables"`
}

type MetricTags struct {
	Values []string
}

func (nb *NetboxProcessor) SampleConfig() string {
	return ""
}

func (nb *NetboxProcessor) Description() string {
	return ""
}

// Extract variables used in GraphQL query
func (nb *NetboxProcessor) ParseGraphQLVars() MetricTags {
	var tags MetricTags
	re := regexp.MustCompile(`\$(\w+)`)
	matches := re.FindAllStringSubmatch(nb.GraphQLQuery, -1)

	seen := make(map[string]bool)
	for _, match := range matches {
		varName := match[1]
		if !seen[varName] {
			tags.Values = append(tags.Values, varName)
			seen[varName] = true
		}
	}
	return tags
}

// Extract alias-to-field mapping like "__tag_status": "status"
func (nb *NetboxProcessor) ParseGraphQLTags() map[string]string {
	aliasMap := make(map[string]string)
	re := regexp.MustCompile(`\b(__tag_[a-zA-Z0-9_]+)\s*:\s*[a-zA-Z0-9_]+`)
	matches := re.FindAllStringSubmatch(nb.GraphQLQuery, -1)

	for _, match := range matches {
		if len(match) == 2 {
			alias := match[1]
			key := strings.TrimPrefix(alias, "__tag_")
			aliasMap[key] = alias
		}
	}
	return aliasMap
}

// Recursive function to extract aliased fields from nested data
func extractAliasedFields(data interface{}, aliasMap map[string]string, result map[string]interface{}) {
	switch v := data.(type) {
	case map[string]interface{}:
		for k, val := range v {
			for cleanKey, alias := range aliasMap {
				if k == alias {
					result[cleanKey] = val
				}
			}
			extractAliasedFields(val, aliasMap, result)
		}
	case []interface{}:
		for _, item := range v {
			extractAliasedFields(item, aliasMap, result)
		}
	}
}

func (nb *NetboxProcessor) Apply(in ...telegraf.Metric) []telegraf.Metric {
	gqlVars := nb.ParseGraphQLVars()
	aliasMap := nb.ParseGraphQLTags()

	for _, gqlVar := range gqlVars.Values {
		for _, metric := range in {
			if value, ok := metric.GetTag(gqlVar); ok {
				// Build GraphQL request
				variables := map[string]string{gqlVar: value}
				requestBody := GraphQLRequest{
					Query:     nb.GraphQLQuery,
					Variables: variables,
				}

				jsonBody, err := json.Marshal(requestBody)
				if err != nil {
					fmt.Println("Error marshalling request:", err)
					continue
				}

				// Send HTTP request
				req, err := http.NewRequest("POST", nb.Url, bytes.NewBuffer(jsonBody))
				if err != nil {
					fmt.Println("Error creating request:", err)
					continue
				}
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Token "+nb.Token)

				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					fmt.Println("HTTP request error:", err)
					continue
				}
				defer resp.Body.Close()

				body, err := io.ReadAll(resp.Body)
				if err != nil {
					fmt.Println("Error reading response:", err)
					continue
				}

				var raw map[string]interface{}
				if err := json.Unmarshal(body, &raw); err != nil {
					fmt.Println("Error unmarshalling response:", err)
					continue
				}

				// Extract tags from nested structure
				result := make(map[string]interface{})
				data, ok := raw["data"].(map[string]interface{})
				if !ok {
					continue
				}

				deviceList, ok := data["device_list"].([]interface{})
				if !ok || len(deviceList) == 0 {
					continue
				}

				firstDevice := deviceList[0]
				extractAliasedFields(firstDevice, aliasMap, result)

				// Add extracted tags to the metric
				for key, val := range result {
					metric.AddTag(key, fmt.Sprintf("%v", val))
				}
			}
		}
	}
	return in
}

func init() {
	processors.Add("netbox", func() telegraf.Processor {
		return &NetboxProcessor{}
	})
}
