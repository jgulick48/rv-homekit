package models

type Config struct {
	BridgeName    string                `json:"bridgeName"`
	OpenHabServer string                `json:"openHabServer"`
	PIN           string                `json:"pin"`
	Port          string                `json:"port"`
	BMVConfig     BMVConfig             `json:"bmvConfig"`
	Automation    map[string]Automation `json:"automation"`
	StatsServer   string                `json:"statsServer"`
}

type BMVConfig struct {
	Device string `json:"device"`
	Baud   int    `json:"baud"`
	Name   string `json:"name"`
}

type Automation struct {
	HighValue float64 `json:"highValue"`
	LowValue  float64 `json:"lowValue"`
	OffDelay  string  `json:"offDelay"`
	CoolDown  string  `json:"coolDown"`
}
