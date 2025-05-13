package metrics

import (
	"fmt"
	"log"

	"github.com/DataDog/datadog-go/statsd"
)

var Metrics *statsd.Client
var StatsEnabled bool
var Debug bool

func FormatTag(key, value string) string {
	return fmt.Sprintf("%s:%s", key, value)
}
func SendGaugeMetric(name string, tags []string, value float64) {
	if StatsEnabled {
		err := Metrics.Gauge(name, value, tags, 1)
		if Debug {
			log.Printf("Sending gauge metric to %s with value %f and Tags %s", name, value, tags)
		}
		if err != nil {
			log.Printf("Got error trying to send metric %s", err.Error())
		}
	}
}
func SendGaugeMetricWithRate(name string, value float64, tags []string, rate float64) {
	if StatsEnabled {
		err := Metrics.Gauge(name, value, tags, rate)
		if Debug {
			log.Printf("Sending gauge metric to %s with value %f and Tags %s", name, value, tags)
		}
		if err != nil {
			log.Printf("Got error trying to send metric %s", err.Error())
		}
	}
}
