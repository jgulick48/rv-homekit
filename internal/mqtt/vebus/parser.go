package vebus

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jgulick48/rv-homekit/internal/metrics"
	"github.com/jgulick48/rv-homekit/internal/models"
)

func NewVeBusClient() Client {
	client := Client{
		values: map[string]vebusMetric{},
		mux:    sync.Mutex{},
	}
	go func() {
		timer := time.NewTicker(10 * time.Second)
		for range timer.C {
			client.sendAllMetrics()
		}
	}()
	return client
}

type vebusMetric struct {
	name  string
	value float64
	tags  []string
}

type Client struct {
	mux    sync.Mutex
	values map[string]vebusMetric
}

func (c *Client) GetDataParser(segments []string, defaultParser func(topic []string, message models.Message) ([]string, float64)) func(topic []string, message models.Message) ([]string, float64) {
	if len(segments) < 5 {
		return defaultParser
	}
	switch segments[4] {
	case "Ac":
		return c.ParseACData
	default:
		return defaultParser
	}

}

func (c *Client) sendAllMetrics() {
	if metrics.StatsEnabled {
		for _, value := range c.values {
			metrics.SendGaugeMetric(value.name, value.tags, value.value)
		}
	}
}

func (c *Client) ParseACData(segments []string, message models.Message) ([]string, float64) {
	if !message.Value.Valid {
		return []string{}, 0
	}
	tags := []string{
		metrics.FormatTag("deployment", segments[1]),
		metrics.FormatTag("vebus.id", segments[3]),
	}
	var metricName string
	var shouldSend bool
	switch segments[5] {
	case "ActiveIn", "Out":
		tags, metricName, shouldSend = c.parseACLineMeasurements(tags, segments)
	}
	if metricName == "" || !shouldSend {
		return []string{}, 0
	}
	c.mux.Lock()
	defer c.mux.Unlock()
	key := fmt.Sprintf("%s_%s", metricName, strings.Join(tags, "_"))
	c.values[key] = vebusMetric{
		name:  metricName,
		value: message.Value.Float64,
		tags:  tags,
	}
	return append([]string{metricName}, tags...), message.Value.Float64
}

func (c *Client) parseACLineMeasurements(tags []string, segments []string) ([]string, string, bool) {
	if len(segments) == 8 {
		tags = append(tags, metrics.FormatTag("line", segments[6]))
	}
	unit := ""
	switch segments[len(segments)-1] {
	case "F":
		unit = "frequency"
	case "I":
		unit = "current"
	case "P":
		unit = "power"
	case "V":
		unit = "volts"
	}
	if unit == "" {
		return tags, "", false
	}
	return tags, fmt.Sprintf("ac_%s_%s", strings.ToLower(segments[5]), unit), true
}
