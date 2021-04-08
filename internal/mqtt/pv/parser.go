package pv

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jgulick48/rv-homekit/internal/metrics"
	"github.com/jgulick48/rv-homekit/internal/models"
)

func NewPVClient() Client {
	client := Client{
		values: map[string]pvMetric{},
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

type pvMetric struct {
	name  string
	value float64
	tags  []string
}

func (c *Client) sendAllMetrics() {
	if metrics.StatsEnabled {
		for _, value := range c.values {
			metrics.SendGaugeMetric(value.name, value.tags, value.value)
		}
	}
}

type Client struct {
	mux    sync.Mutex
	values map[string]pvMetric
}

func (c Client) GetDataParser(segments []string, defaultParser func(topic []string, message models.Message) ([]string, float64)) func(topic []string, message models.Message) ([]string, float64) {
	if len(segments) < 5 {
		return defaultParser
	}
	switch segments[4] {
	case "Dc", "Yield":
		return c.ParseDCData
	case "Pv":
		return c.ParsePVData
	case "History":
		return c.ParseHistoryData
	default:
		return defaultParser
	}

}

func (c Client) ParseHistoryData(segments []string, message models.Message) ([]string, float64) {
	if !message.Value.Valid {
		return []string{}, 0
	}
	tags := []string{
		metrics.FormatTag("deployment", segments[1]),
		metrics.FormatTag("pvcharger.id", segments[3]),
	}
	tags, metricName, shouldParse := parseHistoryMeasurements(tags, segments)
	if !shouldParse {
		return []string{}, 0
	}
	c.mux.Lock()
	defer c.mux.Unlock()
	key := fmt.Sprintf("%s_%s", metricName, strings.Join(tags, "_"))
	c.values[key] = pvMetric{
		name:  metricName,
		value: message.Value.Float64,
		tags:  tags,
	}
	return append([]string{metricName}, tags...), message.Value.Float64
}

func (c Client) ParseDCData(segments []string, message models.Message) ([]string, float64) {
	if !message.Value.Valid {
		return []string{}, 0
	}
	tags := []string{
		metrics.FormatTag("deployment", segments[1]),
		metrics.FormatTag("pvcharger.id", segments[3]),
	}
	tags, metricName, shouldParse := parseDCLineMeasurements(tags, segments)
	if !shouldParse {
		return []string{}, 0
	}
	c.mux.Lock()
	defer c.mux.Unlock()
	key := fmt.Sprintf("%s_%s", metricName, strings.Join(tags, "_"))
	c.values[key] = pvMetric{
		name:  metricName,
		value: message.Value.Float64,
		tags:  tags,
	}
	return append([]string{metricName}, tags...), message.Value.Float64
}

func (c Client) ParsePVData(segments []string, message models.Message) ([]string, float64) {
	if !message.Value.Valid {
		return []string{}, 0
	}
	tags := []string{
		metrics.FormatTag("deployment", segments[1]),
		metrics.FormatTag("pvcharger.id", segments[3]),
	}
	tags, metricName, shouldParse := parsePVLineMeasurements(tags, segments)
	if !shouldParse {
		return []string{}, 0
	}
	c.mux.Lock()
	defer c.mux.Unlock()
	key := fmt.Sprintf("%s_%s", metricName, strings.Join(tags, "_"))
	c.values[key] = pvMetric{
		name:  metricName,
		value: message.Value.Float64,
		tags:  tags,
	}
	return append([]string{metricName}, tags...), message.Value.Float64
}

func parseDCLineMeasurements(tags []string, segments []string) ([]string, string, bool) {
	unit := ""
	switch segments[len(segments)-1] {
	case "Temperature":
		unit = "degrees"
	case "Current":
		unit = "current"
	case "Power":
		unit = "watts"
	case "Voltage":
		unit = "volts"
	case "ConsumedAmphours":
		unit = "ampHours"
	case "Soc":
		unit = "stateofcharge"
	case "TimeToGo":
		unit = "secondsRemaining"
	}
	if unit == "" {
		return tags, "", false
	}
	return tags, fmt.Sprintf("pv_charger_%s", unit), true
}

func parsePVLineMeasurements(tags []string, segments []string) ([]string, string, bool) {
	unit := ""
	switch segments[len(segments)-1] {
	case "I":
		unit = "current"
	case "V":
		unit = "volts"
	}
	if unit == "" {
		return tags, "", false
	}
	return tags, fmt.Sprintf("pv_array_%s", unit), true
}

func parseHistoryMeasurements(tags []string, segments []string) ([]string, string, bool) {
	unit := ""
	day := ""
	switch segments[len(segments)-2] {
	case "0":
		day = "today"
	case "1":
		day = "yesterday"
	}
	switch segments[len(segments)-1] {
	case "Yield":
		unit = "yield"
	}
	if unit == "" || day == "" {
		return tags, "", false
	}
	return tags, fmt.Sprintf("pv_history_%s_%s", unit, day), true
}
