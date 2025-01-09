//go:build !custom || inputs || inputs.cisco_ise

package all

import _ "github.com/influxdata/telegraf/plugins/inputs/cisco_ise" // register plugin
