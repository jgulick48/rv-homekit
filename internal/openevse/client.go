package openevse

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jgulick48/rv-homekit/internal/metrics"
	"github.com/jgulick48/rv-homekit/internal/models"
	"github.com/jgulick48/rv-homekit/internal/mqtt/vebus"
	"log"
	"math"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	inverterStats vebus.Client
	httpClient    *http.Client
	config        models.EVSEConfiguration
	done          chan bool
}

func (c *Client) GetState() (string, error) {
	isEnabled, err := c.IsEnabled()
	if isEnabled {
		return "ON", err
	}
	return "OFF", err
}

func (c *Client) SetState(state string) {
	if !c.config.EnableControl {
		return
	}
	switch state {
	case "ON":
		c.Enable(true)
	case "OFF":
		c.Enable(false)
	}
	return
}

func (c *Client) InHPState() bool {
	if !c.config.EnableControl {
		return false
	}
	isEnabled, _ := c.IsEnabled()
	return isEnabled
}

func NewClient(veClient vebus.Client, config models.EVSEConfiguration, httpClient *http.Client) Client {
	client := Client{
		inverterStats: veClient,
		httpClient:    httpClient,
		config:        config,
	}
	var ticker, ticker2 *time.Ticker
	ticker = time.NewTicker(10 * time.Second)
	ticker2 = time.NewTicker(30 * time.Second)
	if !config.EnableControl {
		ticker2.Stop()
	}
	client.done = make(chan bool)
	go func() {
		for {
			select {
			case <-client.done:
				return
			case <-ticker.C:
				if client.config.Enabled {
					client.getAndReportStatus()
				}
			case <-ticker2.C:
				if client.config.Enabled {
					client.evaluateChargeLimit()
				}
			}
		}
	}()
	return client
}

func (c *Client) Stop() {
	c.done <- true
}

func (c *Client) IsEnabled() (bool, error) {
	statusMap, err := c.getStatus()
	if err != nil {
		return false, err
	}
	status, ok := statusMap["status"]
	if !ok {
		return false, errors.New("status was not included in message")
	}
	isEnabled := status == "active"
	return isEnabled, nil
}

func (c *Client) Enable(shouldEnable bool) {
	isEnabled, _ := c.IsEnabled()
	if shouldEnable && !isEnabled {
		c.deleteOverride()
	} else if !shouldEnable && isEnabled {
		c.setOverride()
	}
}

func (c *Client) GetChargeLimitSetting() (int, error) {
	result, err := c.getConfig()
	if err != nil {
		return 0, err
	}
	return result.MaxCurrentSoft, nil
}

func (c *Client) SetChargeLimitSetting(limit int) {
	if limit < 6 {
		limit = 6
	}
	if limit > c.config.MaxChargeCurrent {
		limit = c.config.MaxChargeCurrent
	}
	log.Printf("Setting new charge current limit to %v", limit)
	config := Config{
		MaxCurrentSoft: limit,
	}
	result, err := c.setConfig(config)
	if err != nil {
		log.Printf("Error setting charge limit setting %s", err)
		return
	}
	retMessage := strings.Split(result.Msg, " ")
	if retMessage[0] == "done" {
		log.Printf("Updated charging limit with result %s", result.Msg)
	}
}

func (c *Client) setOverride() error {
	body := bytes.NewBuffer([]byte("{state: \"disabled\"}"))
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/override", c.config.Address), body)
	var response OverrideResult
	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("Error making request for item from openEVSE: %s", err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		log.Printf("Invalid response from openEVSE. Got %v expecting 200", resp.StatusCode)
		return err
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		log.Printf("Unable to decod message from openEVSE: %s", err)
		return err
	}
	return nil
}

func (c *Client) deleteOverride() error {
	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/override", c.config.Address), nil)
	var response OverrideResult
	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("Error making request for item from openEVSE: %s", err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		log.Printf("Invalid response from openEVSE. Got %v expecting 200", resp.StatusCode)
		return err
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		log.Printf("Unable to decod message from openEVSE: %s", err)
		return err
	}
	return nil
}

func (c *Client) processGetRequest(rapi string) (CommandResult, error) {
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/r?json=1&rapi=%s", c.config.Address, rapi), nil)
	var response CommandResult
	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("Error making request for item from openEVSE: %s", err)
		return response, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("Invalid response from openEVSE. Got %v expecting 200", resp.StatusCode)
		return response, err
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		log.Printf("Unable to decod message from openEVSE: %s", err)
		return response, err
	}
	return response, nil
}

func (c *Client) getConfig() (Config, error) {
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/config", c.config.Address), nil)
	var response Config
	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("Error making request for item from openEVSE: %s", err)
		return response, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("Invalid response from openEVSE. Got %v expecting 200", resp.StatusCode)
		return response, errors.New(resp.Status)
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		log.Printf("Unable to decod message from openEVSE: %s", err)
		return response, err
	}
	return response, nil
}

func (c *Client) setConfig(config Config) (ConfigUpdateResponse, error) {
	bodyBytes, _ := json.Marshal(config)
	body := bytes.NewBuffer(bodyBytes)
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/config", c.config.Address), body)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("Error making request for item from openEVSE: %s", err)
		return ConfigUpdateResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("Invalid response from openEVSE. Got %v expecting 200", resp.StatusCode)
		return ConfigUpdateResponse{}, errors.New(resp.Status)
	}
	var response ConfigUpdateResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		log.Printf("Unable to decod message from openEVSE: %s", err)
		return ConfigUpdateResponse{}, err
	}
	return response, nil
}

func (c *Client) getAndReportStatus() {
	response, err := c.getStatus()
	if err != nil {
		log.Printf("Unable to decod message from openEVSE: %s", err)
		return
	}
	if metrics.StatsEnabled {
		for key, value := range response {
			switch value.(type) {
			case float64:
				metrics.SendGaugeMetric(fmt.Sprintf("openevse_%s", key), []string{}, value.(float64))
			case bool:
				gaugeValue := 0
				if value.(bool) {
					gaugeValue = 1
				}
				metrics.SendGaugeMetric(fmt.Sprintf("openevse_%s", key), []string{}, float64(gaugeValue))
			case string:
				metrics.SendGaugeMetric(fmt.Sprintf("openevse_%s", key), []string{fmt.Sprintf("%s:%s", key, value.(string))}, float64(1))
			default:
				log.Printf("Got unrecognized type for record %s got %T", key, value)
				continue
			}
		}
	}
	return
}

func (c *Client) getStatus() (map[string]interface{}, error) {
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/status", c.config.Address), nil)
	var response map[string]interface{}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("Error making request for item from openEVSE: %s", err)
		return response, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("Invalid response from openEVSE. Got %v expecting 200", resp.StatusCode)
		return response, nil
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		log.Printf("Unable to decod message from openEVSE: %s", err)
		return response, err
	}
	return response, nil
}

func (c *Client) evaluateChargeLimit() {
	response, err := c.getStatus()
	if err != nil {
		log.Printf("Unable to decod message from openEVSE: %s", err)
		return
	}
	currentChargeCurrent := float64(0)
	if value, ok := response["amp"]; ok {
		switch value.(type) {
		case float64:
			currentChargeCurrent = value.(float64) / 1000
		default:
			log.Printf("Got invalid type for record got %T", value)
			return
		}
	}
	currentChargeLimit, err := c.GetChargeLimitSetting()
	if err != nil {
		log.Printf("Error getting charge limit %s:", err)
		return
	}
	if metrics.StatsEnabled {
		metrics.SendGaugeMetric("openevse_limit", []string{}, float64(currentChargeLimit))
	}
	currentTotalLoad := c.inverterStats.GetAmperageOut()
	currentLoad := currentTotalLoad - currentChargeCurrent
	maxLoad := c.inverterStats.GetCurrentLimit()
	availableLoad := maxLoad - currentLoad
	newChargeLimit := int(math.Floor(availableLoad - float64(c.config.MinCurrentBuffer)))
	if newChargeLimit < 6 {
		newChargeLimit = 6
	}
	if newChargeLimit > c.config.MaxChargeCurrent {
		newChargeLimit = c.config.MaxChargeCurrent
	}
	if math.Abs(float64(currentChargeLimit-newChargeLimit)) > 2 {

		log.Printf("Got current charge limit of %v, current load %v, max load %v and available load %v, setting charge limit to %v", currentChargeLimit, currentTotalLoad, maxLoad, availableLoad, newChargeLimit)
		c.SetChargeLimitSetting(newChargeLimit)
	}

}
