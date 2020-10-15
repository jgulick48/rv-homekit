package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var configString = `{
  "bridgeName": "Big Blue",
  "openHabServer": "http://192.168.1.4:8080",
  "pin": "00102003",
  "port": "12321",
  "bmvConfig": {
    "device": "/dev/ttyUSB0",
    "baud": 19200
  },
  "automation": {
    "generator": {
      "lowValue": 10,
      "highValue": 99.9,
      "offDelay": "5m",
      "coolDown": "30m",
      "maxOn": "3h",
	  "minOn": "1h",
      "minChargeCurrent": 1
    }
  }
}`

var expectedConfig = Config{
	BridgeName:    "Big Blue",
	OpenHabServer: "http://192.168.1.4:8080",
	PIN:           "00102003",
	Port:          "12321",
	BMVConfig: BMVConfig{
		Device: "/dev/ttyUSB0",
		Baud:   19200,
	},
	Automation: map[string]Automation{
		"generator": {
			HighValue:        99.9,
			LowValue:         10,
			OffDelay:         Duration{5 * time.Minute},
			CoolDown:         Duration{30 * time.Minute},
			MinOn:            Duration{time.Hour},
			MaxOn:            Duration{3 * time.Hour},
			MinChargeCurrent: 1,
		},
	},
}

func Test_ConfigParse(t *testing.T) {
	var actualConfig Config
	err := json.Unmarshal([]byte(configString), &actualConfig)
	assert.NoError(t, err)
	assert.Equal(t, expectedConfig, actualConfig)
}
