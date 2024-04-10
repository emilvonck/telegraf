//go:generate ../../../tools/readme_config_includer/generator
package fortigate

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/config"
	"github.com/influxdata/telegraf/plugins/common/tls"
	"github.com/influxdata/telegraf/plugins/inputs"
)

//go:embed sample.conf
var sampleConfig string

type Fortigate struct {
	Host    string          `toml:"host"`
	ApiKey  string          `toml:"api_key"`
	Log     telegraf.Logger `toml:"-"`
	Timeout config.Duration `toml:"timeout"`
	tls.ClientConfig

	client *http.Client
}

type Response struct {
	HTTPMethod string      `json:"http_method"`
	Results    interface{} `json:"results"`
	Vdom       string      `json:"vdom"`
	Path       string      `json:"path"`
	Name       string      `json:"name"`
	Action     string      `json:"action"`
	Status     string      `json:"status"`
	Serial     string      `json:"serial"`
	Version    string      `json:"version"`
	Build      int         `json:"build"`
}

func (*Fortigate) SampleConfig() string {
	return sampleConfig
}

// Init is for setup, and validating config.
func (f *Fortigate) Init() error {
	return nil
}

type Probes func(*Fortigate, context.Context, telegraf.Accumulator) error

func (f *Fortigate) Gather(acc telegraf.Accumulator) error {
	functionMap := map[string]Probes{
		"wifi_managed_ap":          (*Fortigate).GetManagedAp,
		"wifi_client":              (*Fortigate).GetWiFiClients,
		"system_status":            (*Fortigate).GetSystemStatus,
		"system_running_processes": (*Fortigate).GetSystemRunningProcesses,
		// Add more functions to the map as needed
	}
	ctx := context.Background()

	if f.client == nil {
		tlsCfg, err := f.ClientConfig.TLSConfig()
		if err != nil {
			return err
		}
		f.client = &http.Client{
			Transport: &http.Transport{
				ResponseHeaderTimeout: time.Duration(f.Timeout),
				TLSClientConfig:       tlsCfg,
			},
			Timeout: time.Duration(f.Timeout),
		}
	}

	// Create a wait group to synchronize the execution of goroutines
	var wg sync.WaitGroup

	// Iterate over the function map and execute each function concurrently
	for key, fn := range functionMap {
		wg.Add(1)
		go func(key string, fn Probes) {
			defer wg.Done()
			if err := fn(f, ctx, acc); err != nil {
				acc.AddError(fmt.Errorf("error executing %s: %v", key, err))
			}
		}(key, fn)
	}

	// Wait for all goroutines to finish
	wg.Wait()

	return nil
}

func (f *Fortigate) sendRequest(ctx context.Context, req *http.Request, v interface{}) error {
	req = req.WithContext(ctx)
	req.Header.Set("Accept", "application/json; charset=utf-8")

	var bearer = "Bearer " + f.ApiKey

	req.Header.Add("Authorization", bearer)

	res, err := f.client.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	// Try to unmarshall into errorResponse
	if res.StatusCode != http.StatusOK {
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("unknown error, status code: %d", res.StatusCode)
		}
		return fmt.Errorf("error: %s", string(b))
	}

	// Unmarshall and populate v
	if err = json.NewDecoder(res.Body).Decode(&v); err != nil {
		return err
	}

	return nil
}

func init() {
	inputs.Add("fortigate", func() telegraf.Input { return &Fortigate{} })
}
