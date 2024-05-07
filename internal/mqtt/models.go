package mqtt

type SensorJSON struct {
	UniqueId          string       `json:"unique_id"`
	Name              string       `json:"name"`
	StateTopic        string       `json:"state_topic"`
	StateClass        string       `json:"state_class"`
	DeviceClass       string       `json:"device_class"`
	ValueTemplate     string       `json:"value_template"`
	UnitOfMeasurement string       `json:"unit_of_measurement"`
	Device            SensorDevice `json:"device"`
}

type SensorDevice struct {
	Manufacturer string   `json:"manufacturer"`
	Name         string   `json:"name"`
	Identifiers  []string `json:"identifiers"`
}
