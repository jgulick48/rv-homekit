package models

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/guregu/null"
)

type Config struct {
	BridgeName            string                `json:"bridgeName"`
	OpenHabServer         string                `json:"openHabServer"`
	CrashOnDeviceMismatch bool                  `json:"crashOnDeviceMismatch"`
	DVCCConfiguration     DVCCConfiguration     `json:"dvccConfiguration"`
	Debug                 bool                  `json:"debug"`
	MQTTConfiguration     MQTTConfiguration     `json:"mqttConfiguration"`
	PIN                   string                `json:"pin"`
	Port                  string                `json:"port"`
	BMVConfig             BMVConfig             `json:"bmvConfig"`
	Automation            map[string]Automation `json:"automation"`
	StatsServer           string                `json:"statsServer"`
	ThermostatRange       TemperatureRange      `json:"thermostatRange"`
	TankSensors           MopkeaProCheck        `json:"tankSensors"`
	SyncTimer             string                `json:"syncTimer"`
	GeneratorOffDelay     time.Duration         `json:"generatorOffDelay"`
}

type MQTTConfiguration struct {
	UseVRM   bool   `json:"useVRM"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	DeviceID string `json:"deviceId"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type DVCCConfiguration struct {
	HighChargeCurrentMax float64  `json:"highChargeCurrentMax"`
	LowChargeCurrentMax  float64  `json:"lowChargeCurrentMax"`
	Steps                int      `json:"steps"`
	StartDelay           Duration `json:"startDelay"`
	StepTime             Duration `json:"stepTime"`
}

type BMVConfig struct {
	Device string `json:"device"`
	Baud   int    `json:"baud"`
	Name   string `json:"name"`
}

type Automation struct {
	HighValue        float64  `json:"highValue"`
	LowValue         float64  `json:"lowValue"`
	OffDelay         Duration `json:"offDelay"`
	CoolDown         Duration `json:"coolDown"`
	MinOn            Duration `json:"minOn"`
	MaxOn            Duration `json:"maxOn"`
	MinChargeCurrent float64  `json:"minChargeCurrent"`
}

type TemperatureRange struct {
	MinValue float64 `json:"minValue"`
	MaxValue float64 `json:"maxValue"`
	Unit     string  `json:"unit"`
}

type MopkeaProCheck struct {
	Enabled         bool                `json:"enabled"`
	DefaultTankType string              `json:"defaultTankType"`
	Devices         []MopekaLevelSensor `json:"devices"`
	APIAddress      string              `json:"apiAddress"`
}
type MopekaLevelSensor struct {
	Address    string  `json:"address"`
	Name       string  `json:"name"`
	Type       string  `json:"type"`
	MaxHeight  float64 `json:"maxHeight"`
	Discovered bool    `json:"-"`
}

type Duration struct {
	time.Duration
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		d.Duration = time.Duration(value)
		return nil
	case string:
		var err error
		d.Duration, err = time.ParseDuration(value)
		if err != nil {
			return err
		}
		return nil
	default:
		return errors.New("invalid duration")
	}
}

type Message struct {
	Value null.Float `json:"value"`
}

type Metric struct {
	Tags  []string
	Value float64
}
