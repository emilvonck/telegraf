package graphql

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/common/parallel"
	"github.com/influxdata/telegraf/plugins/processors"
)

var sampleConfig = `
  ## Maximum number of metrics to process in parallel
  max_parallel = 10

  ## Only process metrics matching these names (optional)
  namepass = ["ping"]

  ## The GraphQL query to execute (supports variables with $varname)
  graphql_query = """
  query ($url: String) {
    __root_element: device_list(filters: {name: {exact: $url}}) {
      __tag_status: status
      site {
        __tag_site: name
        __tag_latitude: latitude
        __tag_longitude: longitude
        __tag_address: physical_address
      }
    }
  }
  """

  ## URL of the GraphQL API endpoint
  url = "http://192.0.2.1:1337/graphql/"

  ## Authentication token to include in the Authorization header
  token = "0123456789abcdef0123456789abcdef01234567"
`

type GraphQLProcessor struct {
	GraphQLQuery string          `toml:"graphql_query"`
	Url          string          `toml:"url"`
	Token        string          `toml:"token"`
	MaxParallel  int             `toml:"max_parallel"`
	Log          telegraf.Logger `toml:"-"`
	acc          telegraf.Accumulator
	parallel     parallel.Parallel
	parsedVars   MetricTags
	aliasMap     map[string]string
}

type GraphQLErrorResponse struct {
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type GraphQLRequest struct {
	Query     string            `json:"query"`
	Variables map[string]string `json:"variables"`
}

type MetricTags struct {
	Values []string
}

func (*GraphQLProcessor) SampleConfig() string {
	return sampleConfig
}

func (g *GraphQLProcessor) Init() error {
	if g.MaxParallel <= 0 {
		g.MaxParallel = 10
	}
	return nil
}

func (g *GraphQLProcessor) Start(acc telegraf.Accumulator) error {
	g.acc = acc
	g.parallel = parallel.NewUnordered(acc, g.processMetric, g.MaxParallel)
	g.Log.Infof("GraphQL processor started with max_parallel=%d targeting URL=%s", g.MaxParallel, g.Url)

	g.parsedVars = g.ParseGraphQLVars()
	g.aliasMap = g.ParseGraphQLTags()
	return nil
}

func (g *GraphQLProcessor) Add(metric telegraf.Metric, _ telegraf.Accumulator) error {
	g.parallel.Enqueue(metric)
	return nil
}

func (g *GraphQLProcessor) Stop() {
	g.Log.Infof("Stopping GraphQL processor")
	g.parallel.Stop()
}

func (gp *GraphQLProcessor) ParseGraphQLVars() MetricTags {
	var tags MetricTags
	re := regexp.MustCompile(`\$(\w+)`)
	matches := re.FindAllStringSubmatch(gp.GraphQLQuery, -1)

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

func (g *GraphQLProcessor) ParseGraphQLTags() map[string]string {
	aliasMap := make(map[string]string)
	re := regexp.MustCompile(`\b(__tag_[a-zA-Z0-9_]+)\s*:\s*[a-zA-Z0-9_]+`)
	matches := re.FindAllStringSubmatch(g.GraphQLQuery, -1)

	for _, match := range matches {
		if len(match) == 2 {
			alias := match[1]
			key := strings.TrimPrefix(alias, "__tag_")
			aliasMap[key] = alias
		}
	}
	return aliasMap
}

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

func (g *GraphQLProcessor) processMetric(metric telegraf.Metric) []telegraf.Metric {
	gqlVars := g.parsedVars
	aliasMap := g.aliasMap

	requestBody := GraphQLRequest{
		Query: g.GraphQLQuery,
	}
	for _, gqlVar := range gqlVars.Values {
		if value, ok := metric.GetTag(gqlVar); ok {
			variables := map[string]string{gqlVar: value}
			requestBody.Variables = variables
		} else {
			g.Log.Debugf("GraphQL variable %q not found as tag in metric %q", gqlVar, metric.Name())
		}
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		g.Log.Errorf("Error marshalling GraphQL request for metric %q: %v", metric.Name(), err)
		return []telegraf.Metric{metric}
	}

	req, err := http.NewRequest("POST", g.Url, bytes.NewBuffer(jsonBody))
	if err != nil {
		g.Log.Errorf("Error creating HTTP request for metric %q: %v", metric.Name(), err)
		return []telegraf.Metric{metric}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Token "+g.Token)

	g.Log.Debugf("Sending GraphQL request for metric %q to %s with variables: %+v", metric.Name(), g.Url, requestBody.Variables)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		g.Log.Errorf("HTTP request error for metric %q: %v", metric.Name(), err)
		return []telegraf.Metric{metric}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		g.Log.Errorf("Error reading HTTP response for metric %q: %v", metric.Name(), err)
		return []telegraf.Metric{metric}
	}

	var raw map[string]interface{}
	err = json.Unmarshal(body, &raw)
	if err != nil {
		var gqlErr GraphQLErrorResponse
		err = json.Unmarshal(body, &gqlErr)
		if err == nil && len(gqlErr.Errors) > 0 {
			for _, e := range gqlErr.Errors {
				g.Log.Errorf("GraphQL error response for metric %q: %s", metric.Name(), e.Message)
			}
		} else {
			text := strings.TrimSpace(string(body))
			g.Log.Errorf("GraphQL response for metric %q could not be parsed: %s", metric.Name(), text)
		}

		return []telegraf.Metric{metric}
	}

	data, ok := raw["data"].(map[string]interface{})
	if !ok {
		g.Log.Warnf("Missing 'data' field in GraphQL response for metric %q: %s", metric.Name(), string(body))
		return []telegraf.Metric{metric}
	}

	rootData, ok := data["__root_element"]
	if !ok {
		g.Log.Warnf("Missing '__root_element' in GraphQL response for metric %q", metric.Name())
		return []telegraf.Metric{metric}
	}

	result := make(map[string]interface{})

	switch v := rootData.(type) {
	case []interface{}:
		if len(v) == 0 {
			g.Log.Debugf("Empty result array for metric %q (no enrichment)", metric.Name())
			return []telegraf.Metric{metric}
		}
		firstItem := v[0]
		extractAliasedFields(firstItem, aliasMap, result)

	case map[string]interface{}:
		extractAliasedFields(v, aliasMap, result)

	default:
		// Unexpected type, skip
		g.Log.Debugf("Unexpected type for '__root_element': %T in metric %q", v, metric.Name())
		return []telegraf.Metric{metric}
	}

	for key, val := range result {
		metric.AddTag(key, fmt.Sprintf("%v", val))
	}

	g.Log.Debugf("Enriched metric %q with tags: %+v", metric.Name(), result)
	return []telegraf.Metric{metric}
}

func init() {
	processors.AddStreaming("graphql", func() telegraf.StreamingProcessor {
		return &GraphQLProcessor{}
	})
}
