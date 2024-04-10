package fortigate

import (
	"context"
	"fmt"
	"net/http"

	"github.com/influxdata/telegraf"
)

type CPU struct {
	User   int `json:"user"`
	Kernel int `json:"kernel"`
}
type Process struct {
	Pid      int    `json:"pid"`
	Command  string `json:"command"`
	State    string `json:"state"`
	Priority string `json:"priority"`
	Memory   int    `json:"memory"`
	Pss      int    `json:"pss"`
	CPU      CPU    `json:"cpu"`
}
type ProcessList struct {
	Processes       []Process `json:"processes"`
	TotalClockTicks int64     `json:"total_clock_ticks"`
}

type ProcessResponse struct {
	Response
	Results ProcessList `json:"results"`
}

func (f *Fortigate) GetSystemRunningProcesses(ctx context.Context, acc telegraf.Accumulator) error {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/v2/monitor/system/running-processes", f.Host), nil)
	if err != nil {
		return err
	}

	res := ProcessResponse{}
	if err := f.sendRequest(ctx, req, &res); err != nil {
		return err
	}

	for _, p := range res.Results.Processes {
		acc.AddFields(fmt.Sprintf("%s_%s", res.Path, res.Name), map[string]interface{}{
			"state":       p.State,
			"prioririty":  p.Priority,
			"memory":      p.Memory,
			"pss":         p.Pss,
			"cpu":         p.CPU,
			"total_ticks": res.Results.TotalClockTicks,
		}, map[string]string{
			"fgt_serial": res.Serial,
			"vdom":       res.Vdom,
			"name":       p.Command,
			"pid":        fmt.Sprint(p.Pid),
		})
	}

	return nil
}
