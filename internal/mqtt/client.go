package mqtt

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/jgulick48/rv-homekit/internal/bmv"
	"github.com/jgulick48/rv-homekit/internal/models"
	"github.com/jgulick48/rv-homekit/internal/mqtt/battery"
	"github.com/jgulick48/rv-homekit/internal/mqtt/pv"
	"github.com/jgulick48/rv-homekit/internal/mqtt/vebus"
)

type Client interface {
	Close()
	Connect()
	GetBatteryClient() bmv.Client
	IsEnabled() bool
}

func NewClient(config models.MQTTConfiguration, debug bool) Client {
	if config.Host != "" {
		client := client{
			config:   config,
			done:     make(chan bool),
			messages: make(chan mqtt.Message),
			battery:  battery.NewBatteryClient(),
			vebus:    vebus.NewVeBusClient(),
			pv:       pv.NewPVClient(),
			debug:    debug,
		}
		return &client
	}
	return &client{config: config}
}

type client struct {
	config     models.MQTTConfiguration
	done       chan bool
	mqttClient mqtt.Client
	messages   chan mqtt.Message
	battery    battery.Client
	vebus      vebus.Client
	pv         pv.Client
	debug      bool
}

func (c *client) Close() {
	c.done <- true
}

func (c *client) IsEnabled() bool {
	return c.config.Host != ""
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
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = c.connectLostHandler
	c.mqttClient = mqtt.NewClient(opts)
	if token := c.mqttClient.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	c.sub()
	defer c.mqttClient.Disconnect(250)
	c.keepAlive()
}

func (c *client) keepAlive() {
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			token := c.mqttClient.Publish(fmt.Sprintf("R/%s/system/0/Serial", c.config.DeviceID), 0, false, "")
			token.Wait()
		}
	}
}

func (c *client) messagePubHandler(client mqtt.Client, msg mqtt.Message) {
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
	default:
		return DefaultParser
	}
}

func (c *client) GetBatteryClient() bmv.Client {
	return c.battery
}

func DefaultParser(segments []string, message models.Message) ([]string, float64) {
	return []string{}, 0
}
