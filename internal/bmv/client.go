package bmv

import (
	"bytes"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tarm/serial"
)

type ClientConfig struct {
	DeviceName string
	Baud       int
}

type Client interface {
	Close()
	GetBatteryStateOfCharge() (float64, bool)
}

type client struct {
	config  ClientConfig
	mux     sync.Mutex
	data    map[string]string
	closing bool
}

func NewClient(config ClientConfig) Client {
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
		Name:        c.config.DeviceName,
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