package mqtt

import (
	"encoding/json"
	"fmt"
	"github.com/jgulick48/rv-homekit/internal/openevse"
	"log"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/jgulick48/rv-homekit/internal/bmv"
	"github.com/jgulick48/rv-homekit/internal/models"
	"github.com/jgulick48/rv-homekit/internal/mqtt/battery"
	"github.com/jgulick48/rv-homekit/internal/mqtt/pv"
	"github.com/jgulick48/rv-homekit/internal/mqtt/vebus"
	"github.com/jgulick48/rv-homekit/internal/openHab"
)

type Client interface {
	Close()
	Connect()
	GetBatteryClient() bmv.Client
	GetVEBusClient() vebus.Client
	IsEnabled() bool
	RegisterOpenHabHPDevice(item *openHab.EnrichedItemDTO)
	RegisterEVSEHPDevice(item *openevse.Client)
	SetMaxChargeCurrent(value float64)
	SetMaxInputCurrent(value float64)
}

func NewClient(config models.MQTTConfiguration, dvccConfig models.CurrentLimitConfiguration, inputConfig models.CurrentLimitConfiguration, shoreDetection models.ShoreDetection, debug bool) Client {
	if config.UseVRM {
		if config.DeviceID != "" {
			sum := 0
			for char := range []rune(config.DeviceID) {
				sum = sum + char
			}
			config.Host = fmt.Sprintf("mqtt%v.victronenergy.com", sum%128)
			config.Port = 443
			log.Printf("Got host of %s", config.Host)
		}
	}
	if config.Host != "" {
		c := client{
			config:       config,
			dvccConfig:   dvccConfig,
			done:         make(chan bool),
			messages:     make(chan mqtt.Message),
			battery:      battery.NewBatteryClient(),
			pv:           pv.NewPVClient(),
			debug:        debug,
			lastReceived: time.Now(),
		}
		c.vebus = vebus.NewVeBusClient(dvccConfig, inputConfig, shoreDetection, c.SetMaxChargeCurrent, c.SetMaxInputCurrent)
		return &c
	}
	return &client{config: config}
}

type client struct {
	config       models.MQTTConfiguration
	dvccConfig   models.CurrentLimitConfiguration
	done         chan bool
	mqttClient   mqtt.Client
	messages     chan mqtt.Message
	battery      battery.Client
	vebus        vebus.Client
	pv           pv.Client
	debug        bool
	hasDVCC      bool
	hasMaxInput  bool
	lastReceived time.Time
}

func (c *client) Close() {
	c.done <- true
}

func (c *client) IsEnabled() bool {
	return c.config.Host != ""
}

func (c *client) RegisterOpenHabHPDevice(item *openHab.EnrichedItemDTO) {
	if item != nil {
		c.vebus.RegisterHPDevice(item.Name, item.Label, item)
	}
}
func (c *client) RegisterEVSEHPDevice(item *openevse.Client) {
	if item != nil {
		c.vebus.RegisterHPDevice("EVSE", "EVSE", item)
	}
}

func (c *client) Connect() {
	go func() {
		for message := range c.messages {
			c.ProcessData(message.Topic(), message.Payload())
		}
	}()
	log.Printf("Connecting to %s", fmt.Sprintf("tcp://%s:%d", c.config.Host, c.config.Port))
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", c.config.Host, c.config.Port))
	opts.SetClientID("go_mqtt_client")
	opts.SetDefaultPublishHandler(c.messagePubHandler)
	if c.config.Username != "" && c.config.Password != "" {
		opts.SetUsername(c.config.Username)
		opts.SetPassword(c.config.Password)
	}
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = c.connectLostHandler
	c.mqttClient = mqtt.NewClient(opts)
	if token := c.mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Printf("Error connecting to mqtt client: %s", token.Error())
	}
	c.sub()
	defer c.mqttClient.Disconnect(250)
	c.publishHASensors()
	c.keepAlive()
}
func (c *client) keepAlive() {
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			if time.Now().Add(-1 * time.Minute).After(c.lastReceived) {
				log.Printf("Metrics are getting stale, exiting connection.")
				return
			}
			token := c.mqttClient.Publish(fmt.Sprintf("R/%s/keepalive", c.config.DeviceID), 0, false, "[\"#\"]")
			token.Wait()
		}
	}
}

func (c *client) publishHASensors() {
	log.Println("Publishing HAS sensors to mqtt")
	sensorDevice := SensorDevice{
		Manufacturer: "Victron",
		Name:         c.config.DeviceID,
		Identifiers:  []string{c.config.DeviceID},
	}
	if body, err := json.Marshal(SensorJSON{
		UniqueId:          fmt.Sprintf("victron_%s_battery_%s_soc", c.config.DeviceID, "288"),
		Name:              "State of Charge",
		StateTopic:        fmt.Sprintf("N/%s/battery/%s/Soc", c.config.DeviceID, "288"),
		StateClass:        "measurement",
		DeviceClass:       "battery",
		ValueTemplate:     "{{ value_json.value }}",
		UnitOfMeasurement: "%",
		Device:            sensorDevice,
	}); err == nil {
		log.Println("Publishing SOC sensor to mqtt")
		token := c.mqttClient.Publish(fmt.Sprintf("homeassistant/sensor/%s/soc/config", c.config.DeviceID), 0, true, body)
		token.Wait()
	}
	if body, err := json.Marshal(SensorJSON{
		UniqueId:          fmt.Sprintf("victron_%s_%s_energy_invertertoacout", c.config.DeviceID, "276"),
		Name:              "Inverter Output Energy",
		StateTopic:        fmt.Sprintf("N/%s/vebus/%s/Energy/InverterToAcOut", c.config.DeviceID, "276"),
		StateClass:        "total_increasing",
		DeviceClass:       "energy",
		ValueTemplate:     "{{ value_json.value }}",
		UnitOfMeasurement: "kWh",
		Device:            sensorDevice,
	}); err == nil {
		log.Println("Publishing InverterToAcOut sensor to mqtt")
		token := c.mqttClient.Publish(fmt.Sprintf("homeassistant/sensor/%s/e_eps/config", c.config.DeviceID), 0, true, body)
		token.Wait()
	}
	if body, err := json.Marshal(SensorJSON{
		UniqueId:          fmt.Sprintf("victron_%s_%s_energy_acin1toacout", c.config.DeviceID, "276"),
		Name:              "Inverter Pass Through Energy L1",
		StateTopic:        fmt.Sprintf("N/%s/vebus/%s/Energy/AcIn1ToAcOut", c.config.DeviceID, "276"),
		StateClass:        "total_increasing",
		DeviceClass:       "energy",
		ValueTemplate:     "{{ value_json.value }}",
		UnitOfMeasurement: "kWh",
		Device:            sensorDevice,
	}); err == nil {
		log.Println("Publishing AcIn1ToAcOut sensor to mqtt")
		token := c.mqttClient.Publish(fmt.Sprintf("homeassistant/sensor/%s/e_pass/config", c.config.DeviceID), 0, true, body)
		token.Wait()
	}
	if body, err := json.Marshal(SensorJSON{
		UniqueId:          fmt.Sprintf("victron_%s_%s_energy_acin1toinverter", c.config.DeviceID, "276"),
		Name:              "Grid Charge Total L1",
		StateTopic:        fmt.Sprintf("N/%s/vebus/%s/Energy/AcIn1ToInverter", c.config.DeviceID, "276"),
		StateClass:        "total_increasing",
		DeviceClass:       "energy",
		ValueTemplate:     "{{ value_json.value }}",
		UnitOfMeasurement: "kWh",
		Device:            sensorDevice,
	}); err == nil {
		log.Println("Publishing AcIn1ToInverter sensor to mqtt")
		token := c.mqttClient.Publish(fmt.Sprintf("homeassistant/sensor/%s/e_inv_in/config", c.config.DeviceID), 0, true, body)
		token.Wait()
	}
	if body, err := json.Marshal(SensorJSON{
		UniqueId:          fmt.Sprintf("victron_%s_%s_energy_acouttoacin1", c.config.DeviceID, "276"),
		Name:              "Grid Output Total L1",
		StateTopic:        fmt.Sprintf("N/%s/vebus/%s/Energy/AcOutToAcIn1", c.config.DeviceID, "276"),
		StateClass:        "total_increasing",
		DeviceClass:       "energy",
		ValueTemplate:     "{{ value_json.value }}",
		UnitOfMeasurement: "kWh",
		Device:            sensorDevice,
	}); err == nil {
		log.Println("Publishing AcOutToAcIn1 sensor to mqtt")
		token := c.mqttClient.Publish(fmt.Sprintf("homeassistant/sensor/%s/e_inv_out_l1/config", c.config.DeviceID), 0, true, body)
		token.Wait()
	}
	for i := 0; i < 2; i++ {
		if body, err := json.Marshal(SensorJSON{
			UniqueId:          fmt.Sprintf("victron_%s_pv%v_yield", c.config.DeviceID, i),
			Name:              fmt.Sprintf("Solar Energy MPPT %v", i),
			StateTopic:        fmt.Sprintf("N/%s/solarcharger/%v/Yield/User", c.config.DeviceID, i),
			StateClass:        "total_increasing",
			DeviceClass:       "energy",
			ValueTemplate:     "{{ value_json.value }}",
			UnitOfMeasurement: "kWh",
			Device:            sensorDevice,
		}); err == nil {
			log.Println("Publishing Yield sensor to mqtt")
			token := c.mqttClient.Publish(fmt.Sprintf("homeassistant/sensor/%s/e_pv%v/config", c.config.DeviceID, i), 0, true, body)
			token.Wait()
		}
	}
	if body, err := json.Marshal(SensorJSON{
		UniqueId:          fmt.Sprintf("victron_%s_battery_%s_chargedEnergy", c.config.DeviceID, "288"),
		Name:              "Charged Energy",
		StateTopic:        fmt.Sprintf("N/%s/battery/%s/History/ChargedEnergy", c.config.DeviceID, "288"),
		StateClass:        "total_increasing",
		DeviceClass:       "energy",
		ValueTemplate:     "{{ value_json.value }}",
		UnitOfMeasurement: "kWh",
		Device:            sensorDevice,
	}); err == nil {
		log.Println("Publishing ChargedEngergy sensor to mqtt")
		token := c.mqttClient.Publish(fmt.Sprintf("homeassistant/sensor/%s/batt_chrg/config", c.config.DeviceID), 0, true, body)
		token.Wait()
	}
	if body, err := json.Marshal(SensorJSON{
		UniqueId:          fmt.Sprintf("victron_%s_battery_%s_dischargedEnergy", c.config.DeviceID, "288"),
		Name:              "Discharged Energy",
		StateTopic:        fmt.Sprintf("N/%s/battery/%s/History/DischargedEnergy", c.config.DeviceID, "288"),
		StateClass:        "total_increasing",
		DeviceClass:       "energy",
		ValueTemplate:     "{{ value_json.value }}",
		UnitOfMeasurement: "kWh",
		Device:            sensorDevice,
	}); err == nil {
		log.Println("Publishing dischargedEngergy sensor to mqtt")
		token := c.mqttClient.Publish(fmt.Sprintf("homeassistant/sensor/%s/batt_dischrg/config", c.config.DeviceID), 0, true, body)
		token.Wait()
	}
}

func (c *client) messagePubHandler(client mqtt.Client, msg mqtt.Message) {
	c.lastReceived = time.Now()
	c.messages <- msg
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	log.Println("Connected")
}

func (c *client) connectLostHandler(client mqtt.Client, err error) {
	log.Printf("Connect lost: %v", err)
	c.done <- true
}

func (c *client) sub() {
	topic := "#"
	token := c.mqttClient.Subscribe(topic, 1, nil)
	token.Wait()
	log.Printf("Subscribed to topic: %s", topic)
}

func (c *client) ProcessData(topic string, message []byte) error {
	var payload models.Message
	err := json.Unmarshal(message, &payload)
	if err != nil {
		return err
	}
	segments := strings.Split(topic, "/")
	parser := c.GetDataParser(segments)
	parser(segments, payload)
	if c.debug {
		log.Printf("Got message from topic: %s %s", topic, message)
	}
	return nil
}

func (c *client) GetDataParser(segments []string) func(topic []string, message models.Message) ([]string, float64) {
	switch segments[2] {
	case "vebus":
		return c.vebus.GetDataParser(segments, DefaultParser)
	case "battery":
		return c.battery.GetDataParser(segments, DefaultParser)
	case "solarcharger":
		return c.pv.GetDataParser(segments, DefaultParser)
	case "system":
		return c.SystemSettingsParser
	default:
		return DefaultParser
	}
}

func (c *client) GetBatteryClient() bmv.Client {
	return c.battery
}

func (c *client) GetVEBusClient() vebus.Client {
	return c.vebus
}

func DefaultParser(segments []string, message models.Message) ([]string, float64) {
	return []string{}, 0
}
func (c *client) SystemSettingsParser(segments []string, message models.Message) ([]string, float64) {
	if !message.Value.Valid {
		return []string{}, 0
	}
	if len(segments) < 6 {
		return []string{}, 0
	}
	if segments[4] == "Control" && segments[5] == "Dvcc" {
		c.hasDVCC = message.Value.Float64 == 1
	}
	return []string{}, 0
}

func (c *client) SetMaxChargeCurrent(value float64) {
	//Name of topic for max charge current settings (N/d41243b4f71d/settings/0/Settings/SystemSetup/MaxChargeCurrent)
	if !c.hasDVCC {
		log.Printf("System not configured for DVCC skipping max current setting")
		return
	}
	if value < 0 {
		return
	}
	log.Printf("Setting max charge current to %v", value)
	if !c.mqttClient.IsConnected() {
		go c.mqttClient.Connect()
	}
	token := c.mqttClient.Publish(fmt.Sprintf("W/%s/settings/0/Settings/SystemSetup/MaxChargeCurrent", c.config.DeviceID), 0, false, fmt.Sprintf("{\"value\": %v}", value))
	token.Wait()
	if token.Error() != nil {
		log.Printf("Error setting mach charge current %s", token.Error())
	}
}
func (c *client) SetMaxInputCurrent(value float64) {
	//Name of topic for max charge current settings (N/d41243b4f71d/vebus/276/Ac/ActiveIn/CurrentLimit)
	if value < 0 {
		return
	}
	log.Printf("Setting max input current to %v", value)
	if !c.mqttClient.IsConnected() {
		go c.mqttClient.Connect()
	}
	token := c.mqttClient.Publish(fmt.Sprintf("W/%s/vebus/276/Ac/ActiveIn/CurrentLimit", c.config.DeviceID), 0, false, fmt.Sprintf("{\"value\": %v}", value))
	token.Wait()
	if token.Error() != nil {
		log.Printf("Error setting mach charge current %s", token.Error())
	}
}
