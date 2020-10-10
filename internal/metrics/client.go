package metrics

import "github.com/DataDog/datadog-go/statsd"

var Metrics *statsd.Client
var StatsEnabled bool
