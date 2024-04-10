package fortigate

import (
	"context"
	"fmt"
	"net/http"

	"github.com/influxdata/telegraf"
)

type ChannelUtilization struct {
	Value    int    `json:"value"`
	Severity string `json:"severity"`
}
type ClientCount struct {
	Value    int    `json:"value"`
	Severity string `json:"severity"`
}
type InterferingSsids struct {
	Value    int    `json:"value"`
	Severity string `json:"severity"`
}
type InfraInterferingSsids struct {
	Value    int    `json:"value"`
	Severity string `json:"severity"`
}
type Overall struct {
	Value    int    `json:"value"`
	Severity string `json:"severity"`
}
type Health struct {
	ChannelUtilization    ChannelUtilization    `json:"channel_utilization"`
	ClientCount           ClientCount           `json:"client_count"`
	InterferingSsids      InterferingSsids      `json:"interfering_ssids"`
	InfraInterferingSsids InfraInterferingSsids `json:"infra_interfering_ssids"`
	Overall               Overall               `json:"overall"`
}

type Radio struct {
	RadioID                   int    `json:"radio_id"`
	DetectedRogueAps          int    `json:"detected_rogue_aps,omitempty"`
	DetectedRogueInfraAps     int    `json:"detected_rogue_infra_aps,omitempty"`
	ClientCount               int    `json:"client_count,omitempty"`
	OperChan                  int    `json:"oper_chan,omitempty"`
	OperTxpower               int    `json:"oper_txpower,omitempty"`
	ChannelUtilizationPercent int    `json:"channel_utilization_percent,omitempty"`
	RadioMaxRateStandardMbps  int    `json:"radio_max_rate_standard_mbps,omitempty"`
	RadioMaxRateMbps          int    `json:"radio_max_rate_mbps,omitempty"`
	NoiseFloor                int    `json:"noise_floor,omitempty"`
	BandwidthRx               int    `json:"bandwidth_rx,omitempty"`
	BandwidthTx               int    `json:"bandwidth_tx,omitempty"`
	BytesRx                   int    `json:"bytes_rx,omitempty"`
	BytesTx                   int    `json:"bytes_tx,omitempty"`
	InterferingAps            int    `json:"interfering_aps,omitempty"`
	TxRetriesPercent          int    `json:"tx_retries_percent,omitempty"`
	MacErrorsRx               int    `json:"mac_errors_rx,omitempty"`
	MacErrorsTx               int    `json:"mac_errors_tx,omitempty"`
	AutoTxpowerHigh           int    `json:"auto_txpower_high,omitempty"`
	AutoTxpowerLow            int    `json:"auto_txpower_low,omitempty"`
	TxDiscardPercentage       int    `json:"tx_discard_percentage,omitempty"`
	Health                    Health `json:"health,omitempty"`
}

type ManagedAP struct {
	Name                  string  `json:"name"`
	Serial                string  `json:"serial"`
	ApProfile             string  `json:"ap_profile"`
	BleProfile            string  `json:"ble_profile"`
	State                 string  `json:"state"`
	ConnectingFrom        string  `json:"connecting_from"`
	Status                string  `json:"status"`
	RegionCode            string  `json:"region_code"`
	ApGroup               string  `json:"ap_group"`
	Clients               int     `json:"clients"`
	OsVersion             string  `json:"os_version"`
	BoardMac              string  `json:"board_mac"`
	JoinTimeRaw           int     `json:"join_time_raw"`
	LastRebootTimeRaw     int     `json:"last_reboot_time_raw"`
	RebootLastDay         bool    `json:"reboot_last_day"`
	ConnectionState       string  `json:"connection_state"`
	ImageDownloadProgress int     `json:"image_download_progress"`
	Radio                 []Radio `json:"radio"`
	Health                Health  `json:"health"`
	LedBlink              bool    `json:"led_blink"`
	CPUUsage              int     `json:"cpu_usage"`
	MemFree               int     `json:"mem_free"`
	MemTotal              int     `json:"mem_total"`
}

type ManagedAPList struct {
	Response             // Embedding Response struct
	Results  []ManagedAP `json:"results"`
}

func (f *Fortigate) GetManagedAp(ctx context.Context, acc telegraf.Accumulator) error {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/v2/monitor/wifi/managed_ap", f.Host), nil)
	if err != nil {
		return err
	}

	res := ManagedAPList{}
	if err := f.sendRequest(ctx, req, &res); err != nil {
		return err
	}

	for _, aP := range res.Results {
		acc.AddFields(fmt.Sprintf("%s_%s_stats", res.Path, res.Name), map[string]interface{}{
			"ap_profile":              aP.ApProfile,
			"ble_profile":             aP.BleProfile,
			"state":                   aP.State,
			"ip":                      aP.ConnectingFrom,
			"status":                  aP.Status,
			"region_code":             aP.RegionCode,
			"ap_group":                aP.ApGroup,
			"clients":                 aP.Clients,
			"os_version":              aP.OsVersion,
			"board_mac":               aP.BoardMac,
			"join_time_raw":           aP.JoinTimeRaw,
			"last_reboot_time_raw":    aP.LastRebootTimeRaw,
			"reboot_last_day":         aP.RebootLastDay,
			"connection_state":        aP.ConnectionState,
			"image_download_progress": aP.ImageDownloadProgress,
			"health":                  aP.Health.Overall.Severity,
			"led_blink":               aP.LedBlink,
			"cpu_usage":               aP.CPUUsage,
			"mem_free":                aP.MemFree,
			"mem_total":               aP.MemTotal,
			"fgt_build":               res.Build,
			"fgt_version":             res.Version,
		}, map[string]string{
			"ap_serial":  aP.Serial,
			"fgt_serial": res.Serial,
			"ap_name":    aP.Name,
			"vdom":       res.Vdom,
		})
		for _, aPRadio := range aP.Radio {
			acc.AddFields(fmt.Sprintf("%s_%s_radio_stats", res.Path, res.Name), map[string]interface{}{
				"detected_rogue_aps":             aPRadio.DetectedRogueAps,
				"detected_rogue_infra_aps":       aPRadio.DetectedRogueInfraAps,
				"client_count":                   aPRadio.ClientCount,
				"oper_chan":                      aPRadio.OperChan,
				"oper_txpower":                   aPRadio.OperTxpower,
				"channel_utilization_percent":    aPRadio.ChannelUtilizationPercent,
				"radio_max_rate_standard_mbps":   aPRadio.RadioMaxRateStandardMbps,
				"radio_max_rate_mbps":            aPRadio.RadioMaxRateMbps,
				"noise_floor":                    aPRadio.NoiseFloor,
				"bandwidth_rx":                   aPRadio.BandwidthRx,
				"bandwidth_tx":                   aPRadio.BandwidthTx,
				"bytes_rx":                       aPRadio.BytesRx,
				"bytes_tx":                       aPRadio.BandwidthTx,
				"interfering_aps":                aPRadio.InterferingAps,
				"tx_retries_percent":             aPRadio.TxRetriesPercent,
				"mac_errors_rx":                  aPRadio.MacErrorsRx,
				"mac_errors_tx":                  aPRadio.MacErrorsTx,
				"auto_txpower_high":              aPRadio.AutoTxpowerHigh,
				"auto_txpower_low":               aPRadio.AutoTxpowerLow,
				"tx_discard_percentage":          aPRadio.TxDiscardPercentage,
				"health_channel_utilization":     aPRadio.Health.ChannelUtilization.Severity,
				"health_client_count":            aPRadio.Health.ClientCount.Severity,
				"health_interfering_ssids":       aPRadio.Health.InterferingSsids.Severity,
				"health_infra_interfering_ssids": aPRadio.Health.InfraInterferingSsids.Severity,
				"health_overall":                 aPRadio.Health.Overall.Severity,
			}, map[string]string{
				"ap_serial":  aP.Serial,
				"fgt_serial": res.Serial,
				"ap_name":    aP.Name,
				"vdom":       res.Vdom,
				"radio_id":   fmt.Sprint(aPRadio.RadioID),
			})
		}
	}
	return nil
}
