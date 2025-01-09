package cisco_ise

import (
	_ "embed"
	"fmt"
	"time"

	isegosdk "github.com/CiscoISE/ciscoise-go-sdk/sdk"
	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/inputs"
)

var sampleConfig string

type CiscoISE struct {
	BaseURL       string          `toml:"base_url"`
	Hostname      string          `toml:"hostname"`
	Username      string          `toml:"username"`
	Password      string          `toml:"password"`
	Debug         string          `toml:"debug"`
	SslVerify     string          `toml:"ssl_verify"`
	UseAPIGateway string          `toml:"use_api_gateway"`
	UseCSRFToken  string          `toml:"use_csrf_token"`
	Log           telegraf.Logger `toml:"-"`

	client *isegosdk.Client
}

func (*CiscoISE) SampleConfig() string {
	return sampleConfig
}

func (c *CiscoISE) createCiscoISEClient() *isegosdk.Client {
	client, err := isegosdk.NewClientWithOptions(c.BaseURL, c.Username, c.Password, c.Debug, c.SslVerify, c.UseAPIGateway, c.UseCSRFToken)

	if err != nil {
		fmt.Println(err)
	}
	return client
}

func (c *CiscoISE) Gather(acc telegraf.Accumulator) error {

	if c.client == nil {
		c.client = c.createCiscoISEClient()
	}

	params := &isegosdk.GetSystemCertificatesQueryParams{}

	result, _, err := c.client.Certificates.GetSystemCertificates(c.Hostname, params)
	if err != nil {
		fmt.Println(err)
	}

	if result != nil && result.Response != nil {
		for _, row := range *result.Response {
			layout := "Mon Jan 02 15:04:05 MST 2006"
			parsedExpTime, err := time.Parse(layout, row.ExpirationDate)
			if err != nil {
				fmt.Println("Error parsing date:", err)
			}
			parsedValidFTime, err := time.Parse(layout, row.ValidFrom)
			if err != nil {
				fmt.Println("Error parsing date:", err)
			}
			acc.AddFields("ise_cert", map[string]interface{}{
				"issued_to":       row.IssuedTo,
				"expiration_date": parsedExpTime.Unix(),
				"used_by":         row.UsedBy,
				"issued_by":       row.IssuedBy,
				"valid_from":      parsedValidFTime.Unix(),
			}, map[string]string{
				"friendly_name": row.FriendlyName,
				"hostname":      c.Hostname,
			})
		}
	}
	return nil
}

func init() {
	inputs.Add("cisco_ise", func() telegraf.Input {
		return &CiscoISE{}
	})
}
