package fortigate

import (
	"context"
	"fmt"
	"net/http"

	"github.com/influxdata/telegraf"
)

type Status struct {
	ModelName     string `json:"model_name"`
	ModelNumber   string `json:"model_number"`
	Model         string `json:"model"`
	Hostname      string `json:"hostname"`
	LogDiskStatus string `json:"log_disk_status"`
}

type StatusResponse struct {
	Response
	Results Status `json:"results"`
}

func (f *Fortigate) GetSystemStatus(ctx context.Context, acc telegraf.Accumulator) error {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/v2/monitor/system/status", f.Host), nil)
	if err != nil {
		return err
	}

	res := StatusResponse{}
	if err := f.sendRequest(ctx, req, &res); err != nil {
		return err
	}
	acc.AddFields(fmt.Sprintf("%s_%s", res.Path, res.Name), map[string]interface{}{
		"model_name":          res.Results.ModelName,
		"model_number":        res.Results.ModelNumber,
		"model":               res.Results.Model,
		"log_disk_status":     res.Results.LogDiskStatus,
		"configured_hostname": res.Results.Hostname,
		"version":             res.Version,
	}, map[string]string{
		"fgt_serial": res.Serial,
		"vdom":       res.Vdom,
	})
	return nil
}
