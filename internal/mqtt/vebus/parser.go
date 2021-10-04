package vebus

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/jgulick48/rv-homekit/internal/metrics"
	"github.com/jgulick48/rv-homekit/internal/models"
	"github.com/jgulick48/rv-homekit/internal/openHab"
)

func NewVeBusClient() Client {
	client := Client{
		values: map[string]vebusMetric{},
		mux:    sync.RWMutex{},
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
	mux        sync.RWMutex
	values     map[string]vebusMetric
	automation Automation
}

type Automation struct {
	HpDevices             map[string]hpDevice `json:"HpDevices"`
	LastShutdownTime      float64             `json:"LastShutdownTime"`
	ShutdownDueToPowerOut bool                `json:"ShutdownDueToPowerOut"`
}

type hpDevice struct {
	item  openHab.EnrichedItemDTO
	State string `json:"state"`
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

func (c *Client) RegisterHPDevice(item *openHab.EnrichedItemDTO) {
	device, ok := c.automation.HpDevices[item.Name]
	if !ok {
		c.automation.HpDevices[item.Name] = hpDevice{
			item:  *item,
			State: item.State,
		}
	} else {
		device.item = *item
		c.automation.HpDevices[item.Name] = device
	}
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
	for _, item := range c.automation.HpDevices {
		item.item.GetCurrentValue()
		item.State = item.item.State
		log.Printf("Setting item %s to OFF from %s due to power failure.", item.item.Name, item.item.State)
		item.item.SetItemState("OFF")
	}
	c.SaveToFile("")
}

func (c *Client) resetHPDevices() {
	for _, item := range c.automation.HpDevices {
		if item.State != "OFF" {
			log.Printf("Setting item %s to %s from OFF due to power restoration.", item.item.Name, item.item.State)
			item.item.SetItemState(item.State)
		}
	}
}
