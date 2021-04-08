package bmv

import (
	"bytes"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tarm/serial"

	"github.com/jgulick48/rv-homekit/internal/models"
)

type ClientConfig struct {
	DeviceName string
	Baud       int
	Name       string
}

type Client interface {
	Close()
	GetBatteryStateOfCharge() (float64, bool)
	GetBatteryCurrent() (float64, bool)
	GetBatteryVoltage() (float64, bool)
	GetConsumedAmpHours() (float64, bool)
	GetBatteryTemperature() (float64, bool)
	GetPower() (float64, bool)
	GetTimeToGo() (float64, bool)
	GetChargeTimeRemaining() (float64, bool)
}

type client struct {
	config  models.BMVConfig
	mux     sync.Mutex
	data    map[string]string
	closing bool
}

func NewClient(config models.BMVConfig) Client {
	c := client{
		config:  config,
		data:    make(map[string]string),
		closing: false,
	}
	go c.startDataAcquisition()
	return &c
}

func (c *client) Close() {
	c.closing = true
}

func (c *client) startDataAcquisition() {
	sconf := &serial.Config{
		Name:        c.config.Device,
		Baud:        c.config.Baud,
		ReadTimeout: time.Second * 5,
	}
	s, err := serial.OpenPort(sconf)
	if err != nil {
		log.Fatal(err)
	}

	data := make([]byte, 1024)
	buf := bytes.NewBuffer(data)
	go func() {
		for {
			n, err := s.Read(data)
			if err != nil {
				log.Fatal(err)
			}
			buf.Write(data[:n])
		}
	}()
	for {
		if buf.Len() < 512 {
			time.Sleep(time.Second)
			continue
		}
		line, err := buf.ReadString(byte('\n'))
		if err != nil {
			log.Printf("error reading from buffer %s", err.Error())
			continue
		}
		kv := strings.Split(line, "\t")
		if len(kv) == 2 {
			c.updateValue(kv[0], kv[1])
			continue
		}
	}
}

func (c *client) updateValue(key string, value string) {
	c.mux.Lock()
	c.data[key] = strings.TrimSpace(value)
	c.mux.Unlock()
}

func (c *client) GetBatteryStateOfCharge() (float64, bool) {
	c.mux.Lock()
	defer c.mux.Unlock()
	value, ok := c.data["SOC"]
	if ok {
		soc, err := strconv.ParseFloat(value, 64)
		if err != nil {
			log.Printf("Error parsing value from map: %s", err.Error())
			return 0, false
		}
		return soc / 10, true
	} else {
		log.Printf("%v", c.data)
	}
	return 0, false
}

func (c *client) GetBatteryCurrent() (float64, bool) {
	c.mux.Lock()
	defer c.mux.Unlock()
	value, ok := c.data["I"]
	if ok {
		current, err := strconv.ParseFloat(value, 64)
		if err != nil {
			log.Printf("Error parsing value from map: %s", err.Error())
			return 0, false
		}
		return current / 1000, true
	}
	return 0, false
}

func (c *client) GetBatteryVoltage() (float64, bool) {
	c.mux.Lock()
	defer c.mux.Unlock()
	value, ok := c.data["V"]
	if ok {
		current, err := strconv.ParseFloat(value, 64)
		if err != nil {
			log.Printf("Error parsing value from map: %s", err.Error())
			return 0, false
		}
		return current / 1000, true
	}
	return 0, false
}
func (c *client) GetConsumedAmpHours() (float64, bool) {
	c.mux.Lock()
	defer c.mux.Unlock()
	value, ok := c.data["CE"]
	if ok {
		current, err := strconv.ParseFloat(value, 64)
		if err != nil {
			log.Printf("Error parsing value from map: %s", err.Error())
			return 0, false
		}
		return current / 1000, true
	}
	return 0, false
}
func (c *client) GetPower() (float64, bool) {
	c.mux.Lock()
	defer c.mux.Unlock()
	value, ok := c.data["P"]
	if ok {
		current, err := strconv.ParseFloat(value, 64)
		if err != nil {
			log.Printf("Error parsing value from map: %s", err.Error())
			return 0, false
		}
		return current, true
	}
	return 0, false
}

func (c *client) GetBatteryTemperature() (float64, bool) {
	c.mux.Lock()
	defer c.mux.Unlock()
	value, ok := c.data["T"]
	if ok {
		current, err := strconv.ParseFloat(value, 64)
		if err != nil {
			log.Printf("Error parsing value from map: %s", err.Error())
			return 0, false
		}
		return current, true
	}
	return 0, false
}

func (c *client) GetChargeTimeRemaining() (float64, bool) {
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

func (c *client) GetDeviceName() string {
	return c.config.Name
}

func (c *client) GetTimeToGo() (float64, bool) {
	return 0, false
}
