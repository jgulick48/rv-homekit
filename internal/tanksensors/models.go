package tanksensors

type Response struct {
	Sensors []Sensor `json:"sensors"`
}

type Sensor struct {
	Address          string             `json:"address"`
	SensorType       string             `json:"sensorType"`
	BatteryLevel     int                `json:"batteryLevel"`
	BatteryVoltage   float64            `json:"batteryVoltage"`
	TempCelsius      float64            `json:"tempCelsius"`
	TempFahrenheit   float64            `json:"tempFahrenheit"`
	TankLevelMM      float64            `json:"tankLevelMM"`
	TankLevelInches  float64            `json:"tankLevelInches"`
	TankLevelPercent map[string]float64 `json:"tankLevelPercent"`
}

func (s *Sensor) GetAddress() string {
	return s.Address
}

func (s *Sensor) GetTempCelsius() float64 {
	return s.TempCelsius
}
func (s *Sensor) GetTempFahrenheit() float64 {
	return s.TempFahrenheit
}

func (s *Sensor) GetTankLevelMM() float64 {
	return s.TankLevelMM
}

func (s *Sensor) GetTankLevelInches() float64 {
	return s.TankLevelInches
}
func (s *Sensor) GetReadQuality() float64 {
	return s.TankLevelMM
}
func (s *Sensor) GetRSSI() float64 {
	return s.TankLevelMM
}
func (s *Sensor) GetSensorType() string {
	return s.SensorType
}
func (s *Sensor) GetBatteryLevel() int {
	return s.BatteryLevel
}
func (s *Sensor) GetBatteryVoltage() float64 {
	return s.BatteryVoltage
}
func (s *Sensor) GetLevelPercent(tankType string) float64 {
	if level, ok := s.TankLevelPercent[tankType]; ok {
		return level
	}
	return 0
}
