package fortios

import (
	"context"
	_ "embed"
	"sync"

	"github.com/emilvonck/fortios-go"
	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/config"
	"github.com/influxdata/telegraf/plugins/inputs"
)

var sampleConfig string

type gatherFunc func(*FortiOS, context.Context, telegraf.Accumulator, string) error

type FortiOS struct {
	Host        string          `toml:"host"`
	ApiKey      string          `toml:"api_key"`
	Timeout     config.Duration `toml:"timeout"`
	GatherFuncs map[string]gatherFunc

	client *fortios.Client
}

func (*FortiOS) SampleConfig() string {
	return sampleConfig
}

func (f *FortiOS) createFortiosClient() *fortios.Client {
	return fortios.NewClient(f.ApiKey, f.Host)
}

func (f *FortiOS) Gather(acc telegraf.Accumulator) error {
	ctx := context.Background()

	if f.client == nil {
		f.client = f.createFortiosClient()
	}
	if f.GatherFuncs == nil {
		f.GatherFuncs = map[string]gatherFunc{
			"wifi_clients":   gatherWifiClients,
			"managed_switch": gatherManagedSwitch,
		}
	}

	var wg sync.WaitGroup
	for gName, gFunc := range f.GatherFuncs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			gFunc(f, ctx, acc, gName)
		}()
		wg.Wait()
	}
	return nil

}

func gatherWifiClients(f *FortiOS, ctx context.Context, acc telegraf.Accumulator, mName string) error { // gather
	data, err := f.client.GetWifiClient(ctx)
	if err != nil {
		return err
	}
	for idx := range data.Results {

		acc.AddFields(mName, map[string]interface{}{
			"access_point":               data.Results[idx].WtpName,
			"bandwidth_tx":               data.Results[idx].BandwidthTx,
			"bandwidth_rx":               data.Results[idx].BandwidthRx,
			"client_ip":                  data.Results[idx].IP,
			"ssid":                       data.Results[idx].Ssid,
			"channel":                    data.Results[idx].Channel,
			"noise":                      data.Results[idx].Noise,
			"health_signal_strength_val": data.Results[idx].Health.SignalStrength.Value,
			"health_signal_strength_sev": data.Results[idx].Health.SignalStrength.Severity,
			"health_snr_val":             data.Results[idx].Health.Snr.Value,
			"health_snr_sev":             data.Results[idx].Health.Snr.Severity,
			"health_band_val":            data.Results[idx].Health.Band.Value,
			"health_band_sev":            data.Results[idx].Health.Band.Severity,
			"health_tx_discard_val":      data.Results[idx].Health.TransmissionDiscard.Value,
			"health_tx_discard_sev":      data.Results[idx].Health.TransmissionDiscard.Severity,
			"health_tx_retry_val":        data.Results[idx].Health.TransmissionRetry.Value,
			"health_tx_retry_sev":        data.Results[idx].Health.TransmissionRetry.Severity,
			"11k_capable":                data.Results[idx].One1KCapable,
			"11v_capable":                data.Results[idx].One1VCapable,
			"11r_capable":                data.Results[idx].One1RCapable,
			"sta_maxrate":                data.Results[idx].StaMaxrate,
			"sta_rxrate":                 data.Results[idx].StaRxrate,
			"sta_rxrate_mcs":             data.Results[idx].StaRxrateMcs,
			"sta_rxrate_score":           data.Results[idx].StaRxrateScore,
			"sta_txrate":                 data.Results[idx].StaTxrate,
			"sta_txrate_mcs":             data.Results[idx].StaTxrateMcs,
			"sta_txrate_score":           data.Results[idx].StaTxrateScore,
		}, map[string]string{
			"host": f.Host,
			"mac":  data.Results[idx].Mac,
			"user": data.Results[idx].User,
		})
	}
	return nil
}

func gatherManagedSwitch(f *FortiOS, ctx context.Context, acc telegraf.Accumulator, mName string) error { // gather
	data, err := f.client.GetManagedSwitch(ctx)
	if err != nil {
		return err
	}
	for sIdx := range data.Results {
		for pIdx := range data.Results[sIdx].Ports {
			acc.AddFields(mName, map[string]interface{}{
				"switch_status":     data.Results[sIdx].Status,
				"switch_os_version": data.Results[sIdx].OsVersion,
				//FortilinkPort
				"port_vlan": data.Results[sIdx].Ports[pIdx].Vlan,
				//FgtPeerPortName
				//FgtPeerDeviceName
				//IslPeerDeviceName
				//IslPeerPortName
				//IslPeerTrunkName
				//MclagIcl
				//Mclag
				"port_status":                  data.Results[sIdx].Ports[pIdx].Status,
				"port_duplex":                  data.Results[sIdx].Ports[pIdx].Duplex,
				"port_speed":                   data.Results[sIdx].Ports[pIdx].Speed,
				"port_poe_capable":             data.Results[sIdx].Ports[pIdx].PoeCapable,
				"port_port_power":              data.Results[sIdx].Ports[pIdx].PortPower,
				"port_power_status":            data.Results[sIdx].Ports[pIdx].PowerStatus,
				"port_transceiver_vendor":      data.Results[sIdx].Ports[pIdx].Transceiver.Vendor,
				"port_transceiver_pn":          data.Results[sIdx].Ports[pIdx].Transceiver.VendorPartNumber,
				"port_stp_status":              data.Results[sIdx].Ports[pIdx].StpStatus,
				"port_igmp_snooping_group_cnt": data.Results[sIdx].Ports[pIdx].IgmpSnoopingGroup.GroupCount,
				"port_dhcp_snooping_untrust":   data.Results[sIdx].Ports[pIdx].DhcpSnooping.Untrusted,
			}, map[string]string{
				"host":      f.Host,
				"switch":    data.Results[sIdx].Name,
				"serial":    data.Results[sIdx].Serial,
				"interface": data.Results[sIdx].Ports[pIdx].Interface,
			})

		}
	}
	return nil
}

func init() {
	inputs.Add("fortios", func() telegraf.Input {
		return &FortiOS{}
	})
}
