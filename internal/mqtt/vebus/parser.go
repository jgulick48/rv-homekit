package vebus

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/jgulick48/rv-homekit/internal/metrics"
	"github.com/jgulick48/rv-homekit/internal/models"
)

func NewVeBusClient(dvccConfig models.CurrentLimitConfiguration, inputLimits models.CurrentLimitConfiguration, chargeCurrentFunc func(value float64), inputCurrentFunc func(value float64)) Client {
	client := Client{
		values:            map[string]vebusMetric{},
		mux:               sync.RWMutex{},
		dvccConfig:        dvccConfig,
		inputLimits:       inputLimits,
		chargeCurrentFunc: chargeCurrentFunc,
		inputCurrentFunc:  inputCurrentFunc,
		automation: Automation{
			HpDevices:             make(map[string]hpDevice, 0),
			LastShutdownTime:      0,
			ShutdownDueToPowerOut: false,
		},
	}
	prometheus.MustRegister(acMeasurements)
	client.LoadFromFile("")
	go func() {
		timer := time.NewTicker(10 * time.Second)
		for range timer.C {
			client.sendAllMetrics()
		}
	}()
	return client
}

var (
	acMeasurements = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "acMeasurement",
			Help: "Number of blob storage operations waiting to be processed.",
		},
		[]string{
			// Which user has requested the operation?
			"line",
			// Of what type is the operation?
			"deployment",
			"vebus_id",
			"measurementType",
			"direction",
		},
	)
)

type vebusMetric struct {
	name  string
	value float64
	tags  []string
}

type Client struct {
	mux               sync.RWMutex
	values            map[string]vebusMetric
	automation        Automation
	dvccConfig        models.CurrentLimitConfiguration
	inputLimits       models.CurrentLimitConfiguration
	chargeCurrentFunc func(value float64)
	inputCurrentFunc  func(value float64)
}

type Automation struct {
	HpDevices             map[string]hpDevice `json:"HpDevices"`
	LastShutdownTime      float64             `json:"LastShutdownTime"`
	ShutdownDueToPowerOut bool                `json:"ShutdownDueToPowerOut"`
}

type HPDevice interface {
	GetState() (string, error)
	SetState(state string)
	InHPState() bool
}

type hpDevice struct {
	HPDevice `json:"-"`
	Name     string `json:"name"`
	State    string `json:"state"`
}

func (c *Client) GetAmperageOut() float64 {
	maxOut := float64(0)
	c.mux.RLock()
	for _, value := range c.values {
		if value.name == "ac_out_current" {
			if value.value > maxOut {
				maxOut = value.value
			}
		}
	}
	c.mux.RUnlock()
	return maxOut
}

func (c *Client) GetCurrentLimit() float64 {
	maxOut := float64(0)
	c.mux.RLock()
	for _, value := range c.values {
		if value.name == "ac_in_current_limit" {
			if value.value > maxOut {
				maxOut = value.value
			}
		}
	}
	c.mux.RUnlock()
	return maxOut
}

func (c *Client) LoadFromFile(filename string) {
	if filename == "" {
		filename = "./hpItems.json"
	}
	configFile, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Printf("No state file found. Making new IDs")
	}
	err = json.Unmarshal(configFile, &c.automation)
	if err != nil {
		log.Printf("Invliad config file provided")
	}
}

func (c *Client) SaveToFile(filename string) {
	if filename == "" {
		filename = "./hpItems.json"
	}
	data, err := json.MarshalIndent(c.automation, "", "  ")
	if err != nil {
		return
	}
	ioutil.WriteFile(filename, data, 0644)
}

func (c *Client) RegisterHPDevice(id, name string, item HPDevice) {
	device, ok := c.automation.HpDevices[id]
	if !ok {
		enabled, _ := item.GetState()
		c.automation.HpDevices[id] = hpDevice{
			Name:     name,
			HPDevice: item,
			State:    enabled,
		}
	} else {
		device.HPDevice = item
		device.Name = name
		c.automation.HpDevices[id] = device
	}
	c.SaveToFile("")
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
		c.mux.RLock()
		for _, value := range c.values {
			metrics.SendGaugeMetric(value.name, value.tags, value.value)
			formatPrometheusMetric(value.name, value.tags, value.value)
		}
		c.mux.RUnlock()
	}
}

func formatPrometheusMetric(name string, tags []string, value float64) {
	labels := make(prometheus.Labels)
	parts := strings.Split(name, "_")
	if len(parts) != 3 {
		return
	}
	labels["direction"] = parts[1]
	labels["measurementType"] = parts[2]
	for _, tag := range tags {
		tagParts := strings.Split(tag, ":")
		if len(tagParts) != 2 {
			continue
		}
		labels[tagParts[0]] = tagParts[1]
	}
	if _, ok := labels["line"]; !ok {
		labels["line"] = "L1"
	}
	acMeasurements.With(labels).Set(value)
}

func (c *Client) ParseACData(segments []string, message models.Message) ([]string, float64) {
	if !message.Value.Valid {
		return []string{}, 0
	}
	tags := []string{
		metrics.FormatTag("deployment", segments[1]),
		metrics.FormatTag("vebus_id", segments[3]),
	}
	var metricName string
	var shouldSend bool
	switch segments[5] {
	case "ActiveIn":
		c.checkForShutdown(segments, message.Value.Float64)
		tags, metricName, shouldSend = c.parseACLineMeasurements(tags, segments)
	case "Out":
		tags, metricName, shouldSend = c.parseACLineMeasurements(tags, segments)
	case "In":
		tags, metricName, shouldSend = c.parseACLineMeasurements(tags, segments)
	}
	if metricName == "" || !shouldSend {
		return []string{}, 0
	}
	key := fmt.Sprintf("%s_%s", metricName, strings.Join(tags, "_"))
	c.mux.Lock()
	c.values[key] = vebusMetric{
		name:  metricName,
		value: message.Value.Float64,
		tags:  tags,
	}
	c.mux.Unlock()
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
	case "CurrentLimit":
		unit = "current_limit"
	}
	if unit == "" {
		return tags, "", false
	}
	return tags, fmt.Sprintf("ac_%s_%s", strings.ToLower(segments[5]), unit), true
}

func (c *Client) checkForShutdown(segments []string, value float64) {
	switch segments[len(segments)-1] {
	case "V":
		if value > 105 {
			if c.automation.ShutdownDueToPowerOut {
				if float64(time.Now().Unix()) > c.automation.LastShutdownTime+(time.Minute.Seconds()) {
					c.resetHPDevices()
					c.automation.ShutdownDueToPowerOut = false
				}
			}
		} else {
			c.automation.LastShutdownTime = float64(time.Now().Unix())
			if !c.automation.ShutdownDueToPowerOut {
				log.Printf("Shutting high power devices down due to power failure. Voltage at %v", value)
				c.shutdownHPDevices()
				c.automation.ShutdownDueToPowerOut = true
			}
		}
	}
}

func (c *Client) shutdownHPDevices() {
	//Name of topic for max charge current settings (N/d41243b4f71d/settings/0/Settings/SystemSetup/MaxChargeCurrent)
	if c.dvccConfig.LowCurrentMax == 0 {
		log.Printf("Skipping config of max charge current due to value being %v", c.dvccConfig.LowCurrentMax)
	} else {
		if c.chargeCurrentFunc != nil {
			c.chargeCurrentFunc(c.dvccConfig.LowCurrentMax)
		}
	}
	if c.inputLimits.LowCurrentMax == 0 {
		log.Printf("Skipping config of max input current due to value being %v", c.dvccConfig.LowCurrentMax)
	} else {
		if c.inputCurrentFunc != nil {
			c.inputCurrentFunc(c.inputLimits.LowCurrentMax)
		}
	}
	for id, item := range c.automation.HpDevices {
		log.Printf("Checking %s to see if shutdown is needed due to power failure.\n", id)
		if !item.InHPState() {
			log.Printf("Item %s is not in high power state, not changing status", id)
			continue
		}
		isEnabled, err := item.GetState()
		if err != nil {
			fmt.Errorf("failed to get status from item %s due to error: %s", id, err.Error())
			continue
		}
		item.State = isEnabled
		log.Printf("Setting item %s enabled to false from %v due to power failure.", id, isEnabled)
		item.SetState("OFF")
	}
	c.SaveToFile("")
}

func (c *Client) resetHPDevices() {
	for id, item := range c.automation.HpDevices {
		if item.State != "OFF" {
			log.Printf("Setting item %s enabled to %v from False due to power restoration.", id, item.State)
			item.SetState(item.State)
		}
	}
	if c.dvccConfig.HighCurrentMax != 0 {
		go func() {
			time.Sleep(c.dvccConfig.StartDelay.Duration)
			t := time.NewTicker(c.dvccConfig.StepTime.Duration)
			steps := 1
			if c.dvccConfig.Steps == 0 {
				c.dvccConfig.Steps = 1
			}
			stepValue := math.Round((c.dvccConfig.HighCurrentMax - c.dvccConfig.LowCurrentMax) / float64(c.dvccConfig.Steps))
			for range t.C {
				log.Printf("Processing step %v of %v", steps, c.dvccConfig.Steps)
				if steps >= c.dvccConfig.Steps {
					c.chargeCurrentFunc(c.dvccConfig.HighCurrentMax)
					t.Stop()
					return
				}
				c.chargeCurrentFunc(c.dvccConfig.LowCurrentMax + (float64(steps) * stepValue))
				steps++
			}
		}()
	}
	if c.inputLimits.HighCurrentMax != 0 {
		go func() {
			time.Sleep(c.inputLimits.StartDelay.Duration)
			t := time.NewTicker(c.inputLimits.StepTime.Duration)
			steps := 1
			if c.inputLimits.Steps == 0 {
				c.inputLimits.Steps = 1
			}
			stepValue := math.Round((c.inputLimits.HighCurrentMax - c.inputLimits.LowCurrentMax) / float64(c.inputLimits.Steps))
			for range t.C {
				log.Printf("Processing step %v of %v", steps, c.inputLimits.Steps)
				if steps >= c.inputLimits.Steps {
					c.inputCurrentFunc(c.inputLimits.HighCurrentMax)
					t.Stop()
					return
				}
				c.inputCurrentFunc(c.inputLimits.LowCurrentMax + (float64(steps) * stepValue))
				steps++
			}
		}()
	}
}
