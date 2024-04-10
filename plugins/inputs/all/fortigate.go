//go:build !custom || inputs || inputs.fortigate

package all

import _ "github.com/influxdata/telegraf/plugins/inputs/fortigate" // register plugin
