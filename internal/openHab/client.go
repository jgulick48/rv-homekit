package openHab

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

const (
	thingsEndpoint = "rest/things"
	itemEndpoint   = "rest/items"
)

type Client interface {
	GetThings() ([]EnrichedThingDTO, error)
	GetItem(uid string) (EnrichedItemDTO, error)
}

type client struct {
	openHabHost string
	httpClient  *http.Client
}

func NewClient(host string) Client {
	return &client{
		openHabHost: host,
		httpClient:  http.DefaultClient,
	}
}

// GetItems returns a list of items were retreived from the openHab host.
func (c *client) GetItems() ([]EnrichedItemDTO, error) {

	return []EnrichedItemDTO{}, nil
}

func (c *client) GetItem(uid string) (EnrichedItemDTO, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s/%s", c.openHabHost, itemEndpoint, uid), nil)
	if err != nil {
		log.Printf("Error creating request for item from OpenHAB: %s", err)
		return EnrichedItemDTO{}, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("Error making request for item from OpenHAB: %s", err)
		return EnrichedItemDTO{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("Invalid response from OpenHAB. Got %v expecting 200", resp.StatusCode)
		return EnrichedItemDTO{}, err
	}
	var item EnrichedItemDTO
	err = json.NewDecoder(resp.Body).Decode(&item)
	if err != nil {
		log.Printf("Unable to decod message from OpenHAB: %s", err)
		return EnrichedItemDTO{}, err
	}
	return item, nil
}

func (c *client) GetThings() ([]EnrichedThingDTO, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%s", c.openHabHost, thingsEndpoint), nil)
	if err != nil {
		log.Printf("Error creating request for things from OpenHAB: %s", err)
		return []EnrichedThingDTO{}, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("Error making request for things from OpenHAB: %s", err)
		return []EnrichedThingDTO{}, err
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("Invalid response from OpenHAB. Got %v expecting 200", resp.StatusCode)
		return []EnrichedThingDTO{}, err
	}
	defer resp.Body.Close()
	var things []EnrichedThingDTO
	err = json.NewDecoder(resp.Body).Decode(&things)
	if err != nil {
		log.Printf("Unable to decod message from OpenHAB: %s", err)
		return []EnrichedThingDTO{}, err
	}
	return things, nil
}
