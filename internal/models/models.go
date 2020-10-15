package models

import (
	"encoding/json"
	"errors"
	"time"
)

type Config struct {
	BridgeName    string                `json:"bridgeName"`
	OpenHabServer string                `json:"openHabServer"`
	PIN           string                `json:"pin"`
	Port          string                `json:"port"`
	BMVConfig     BMVConfig             `json:"bmvConfig"`
	Automation    map[string]Automation `json:"automation"`
}

type BMVConfig struct {
	Device string `json:"device"`
	Baud   int    `json:"baud"`
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
