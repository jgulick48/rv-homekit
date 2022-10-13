package metrics

import (
	"fmt"
	"log"

	"github.com/DataDog/datadog-go/v5/statsd"
)

var Metrics *statsd.Client
var StatsEnabled bool

func FormatTag(key, value string) string {
	return fmt.Sprintf("%s:%s", key, value)
}
func SendGaugeMetric(name string, tags []string, value float64) {
	if StatsEnabled {
		err := Metrics.Gauge(name, value, tags, 1)
		if err != nil {
			log.Printf("Got error trying to send metric %s", err.Error())
		}
	}
}
