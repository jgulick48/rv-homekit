package tanksensors

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type Client interface {
	GetDevice(address string) (Sensor, bool)
	GetDevices() []Sensor
}

type client struct {
	httpClient *http.Client
	apiAddress string
}

func (c *client) GetDevice(address string) (Sensor, bool) {
	devices := c.GetDevices()
	for _, sensor := range devices {
		if sensor.Address == address {
			return sensor, true
		}
	}
	return Sensor{}, false
}

func (c *client) GetDevices() []Sensor {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s", c.apiAddress, "sensors"), nil)
	if err != nil {
		log.Printf("error generating request to get sensors: %s", err)
		return nil
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("Error making request to get sensors: %s", err.Error())
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Printf("unexpected status code from server, %v", resp.StatusCode)
		return nil
	}
	var response Response
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		log.Printf("error parsing request to get sensors: %s", err)
		return nil
	}
	return response.Sensors
}

func NewTankSensorClient(apiAddress string) Client {
	return &client{
		http.DefaultClient,
		apiAddress,
	}
}
