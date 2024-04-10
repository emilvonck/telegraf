package fortigate

import (
	"context"
	"fmt"
	"net/http"

	"github.com/influxdata/telegraf"
)

type SignalStrength struct {
	Value    int    `json:"value"`
	Severity string `json:"severity"`
}
type Snr struct {
	Value    int    `json:"value"`
	Severity string `json:"severity"`
}
type Band struct {
	Value    string `json:"value"`
	Severity string `json:"severity"`
}
type TransmissionRetry struct {
	Value    int    `json:"value"`
	Severity string `json:"severity"`
}
type TransmissionDiscard struct {
	Value    int    `json:"value"`
	Severity string `json:"severity"`
}
type ClientHealth struct {
	SignalStrength      SignalStrength      `json:"signal_strength"`
	Snr                 Snr                 `json:"snr"`
	Band                Band                `json:"band"`
	TransmissionRetry   TransmissionRetry   `json:"transmission_retry"`
	TransmissionDiscard TransmissionDiscard `json:"transmission_discard"`
}
type WiFiClient struct {
	IP                         string       `json:"ip"`
	IP6                        []string     `json:"ip6"`
	WtpName                    string       `json:"wtp_name"`
	WtpID                      string       `json:"wtp_id"`
	WtpRadio                   int          `json:"wtp_radio"`
	WtpIP                      string       `json:"wtp_ip"`
	WtpControlIP               string       `json:"wtp_control_ip"`
	WtpControlLocalIP          string       `json:"wtp_control_local_ip"`
	VapName                    string       `json:"vap_name"`
	Ssid                       string       `json:"ssid"`
	Mac                        string       `json:"mac"`
	One1KCapable               bool         `json:"11k_capable"`
	One1VCapable               bool         `json:"11v_capable"`
	One1RCapable               bool         `json:"11r_capable"`
	StaMaxrate                 int          `json:"sta_maxrate"`
	StaRxrate                  int          `json:"sta_rxrate"`
	StaRxrateMcs               int          `json:"sta_rxrate_mcs"`
	StaRxrateScore             int          `json:"sta_rxrate_score"`
	StaTxrate                  int          `json:"sta_txrate"`
	StaTxrateMcs               int          `json:"sta_txrate_mcs"`
	StaTxrateScore             int          `json:"sta_txrate_score"`
	Os                         string       `json:"os"`
	Authentication             string       `json:"authentication"`
	CaptivePortalAuthenticated int          `json:"captive_portal_authenticated"`
	DataRateBps                int          `json:"data_rate_bps"`
	DataRxrateBps              int          `json:"data_rxrate_bps"`
	DataTxrateBps              int          `json:"data_txrate_bps"`
	Snr                        int          `json:"snr"`
	IdleTime                   int          `json:"idle_time"`
	AssociationTime            int          `json:"association_time"`
	BandwidthTx                int          `json:"bandwidth_tx"`
	BandwidthRx                int          `json:"bandwidth_rx"`
	LanAuthenticated           bool         `json:"lan_authenticated"`
	Channel                    int          `json:"channel"`
	Signal                     int          `json:"signal"`
	Vci                        string       `json:"vci"`
	Host                       string       `json:"host"`
	Security                   int          `json:"security"`
	SecurityStr                string       `json:"security_str"`
	Encrypt                    int          `json:"encrypt"`
	Noise                      int          `json:"noise"`
	RadioType                  string       `json:"radio_type"`
	Mimo                       string       `json:"mimo"`
	VlanID                     int          `json:"vlan_id"`
	TxDiscardPercentage        int          `json:"tx_discard_percentage"`
	TxRetryPercentage          int          `json:"tx_retry_percentage"`
	Health                     ClientHealth `json:"health"`
	Hostname                   string       `json:"hostname,omitempty"`
	Manufacturer               string       `json:"manufacturer,omitempty"`
}

type WiFiClientList struct {
	Response              // Embedding Response struct
	Results  []WiFiClient `json:"results"`
}

func (f *Fortigate) GetWiFiClients(ctx context.Context, acc telegraf.Accumulator) error {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/v2/monitor/wifi/client", f.Host), nil)
	if err != nil {
		return err
	}

	res := WiFiClientList{}
	if err := f.sendRequest(ctx, req, &res); err != nil {
		return err
	}
	for _, wClient := range res.Results {
		acc.AddFields(res.Path+res.Name, map[string]interface{}{
			"ip":       wClient.IP,
			"rssi":     wClient.Signal,
			"snr":      wClient.Snr,
			"ap_name":  wClient.WtpName,
			"radio_id": wClient.WtpRadio,
		}, map[string]string{
			"fgt_serial": res.Serial,
			"vdom":       res.Vdom,
			"mac":        wClient.Mac,
		})
	}
	return nil
}
