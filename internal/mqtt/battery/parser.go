package battery

import (
	"fmt"
	"sync"

	"github.com/jgulick48/rv-homekit/internal/metrics"
	"github.com/jgulick48/rv-homekit/internal/models"
)

func NewBatteryClient() Client {
	return Client{
		values: map[string]float64{
			"battery_secondsRemaining": 0,
		},
		mux: sync.RWMutex{},
	}
}

type Client struct {
	mux    sync.RWMutex
	values map[string]float64
}

func (c Client) Close() {
	panic("implement me")
}

func (c Client) GetBatteryStateOfCharge() (float64, bool) {
	c.mux.RLock()
	value, ok := c.values["battery_stateofcharge"]
	c.mux.RUnlock()
	return value, ok
}

func (c Client) GetBatteryCurrent() (float64, bool) {
	c.mux.RLock()
	value, ok := c.values["battery_current"]
	c.mux.RUnlock()
	return value, ok
}

func (c Client) GetBatteryVoltage() (float64, bool) {
	c.mux.RLock()
	value, ok := c.values["battery_volts"]
	c.mux.RUnlock()
	return value, ok
}

func (c Client) GetConsumedAmpHours() (float64, bool) {
	c.mux.RLock()
	value, ok := c.values["battery_ampHours"]
	c.mux.RUnlock()
	return value, ok
}

func (c Client) GetBatteryTemperature() (float64, bool) {
	c.mux.RLock()
	value, ok := c.values["battery_degrees"]
	c.mux.RUnlock()
	return value, ok
}

func (c Client) GetPower() (float64, bool) {
	c.mux.RLock()
	value, ok := c.values["battery_watts"]
	c.mux.RUnlock()
	return value, ok
}

func (c Client) GetTimeToGo() (float64, bool) {
	c.mux.RLock()
	value, ok := c.values["battery_secondsRemaining"]
	c.mux.RUnlock()
	return value, ok
}
func (c Client) GetChargeTimeRemaining() (float64, bool) {
	ampHours, ok := c.GetConsumedAmpHours()
	if !ok {
		return 0, false
	}
	current, ok := c.GetBatteryCurrent()
	if !ok {
		return 0, false
	}
	if current < 0 {
		return -1, true
	}
	return (-ampHours / current) * 3600, true
}

func (c Client) GetDataParser(segments []string, defaultParser func(topic []string, message models.Message) ([]string, float64)) func(topic []string, message models.Message) ([]string, float64) {
	if len(segments) < 5 {
		return defaultParser
	}
	switch segments[4] {
	case "Dc", "Soc", "TimeToGo", "ConsumedAmphours":
		return c.ParseDCData
	default:
		return defaultParser
	}

}

func (c Client) ParseDCData(segments []string, message models.Message) ([]string, float64) {
	if !message.Value.Valid {
		return []string{}, 0
	}
	tags := []string{
		metrics.FormatTag("deployment", segments[1]),
		metrics.FormatTag("battery_id", segments[3]),
	}
	tags, metricName, shouldParse := parseDCLineMeasurements(tags, segments)
	if !shouldParse {
		return []string{}, 0
	}
	c.mux.Lock()
	c.values[metricName] = message.Value.Float64
	c.mux.Unlock()
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
	return tags, fmt.Sprintf("battery_%s", unit), true
}
