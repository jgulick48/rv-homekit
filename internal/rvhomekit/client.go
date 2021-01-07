package rvhomekit

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/jgulick48/hc/accessory"

	"github.com/jgulick48/rv-homekit/internal/automation"
	"github.com/jgulick48/rv-homekit/internal/bmv"
	"github.com/jgulick48/rv-homekit/internal/metrics"
	"github.com/jgulick48/rv-homekit/internal/models"
	"github.com/jgulick48/rv-homekit/internal/openHab"
)

type client struct {
	config    models.Config
	habClient openHab.Client
	bmvClient *bmv.Client
}

type Client interface {
	GetAccessoriesFromOpenHab(things []openHab.EnrichedThingDTO) []*accessory.Accessory
}

func LoadClientConfig(filename string) models.Config {
	if filename == "" {
		filename = "./config.json"
	}
	configFile, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Printf("No config file found. Making new IDs")
		panic(err)
	}
	var config models.Config
	err = json.Unmarshal(configFile, &config)
	if err != nil {
		log.Printf("Invliad config file provided")
		panic(err)
	}
	return config
}

func NewClient(config models.Config, habClient openHab.Client, bmvClient *bmv.Client) Client {
	return &client{
		config:    config,
		habClient: habClient,
		bmvClient: bmvClient,
	}
}

func (c *client) GetAccessoriesFromOpenHab(things []openHab.EnrichedThingDTO) []*accessory.Accessory {
	var itemIDs map[string]uint64
	itemConfigFile, err := ioutil.ReadFile("./items.json")
	if err != nil {
		log.Printf("No config file found. Making new IDs")
		itemIDs = make(map[string]uint64)
	}
	if err = json.Unmarshal(itemConfigFile, &itemIDs); err != nil {
		log.Printf("Invalid config file format. Starting new.")
		itemIDs = make(map[string]uint64)
	}
	maxID := uint64(2)
	for _, id := range itemIDs {
		if maxID < id {
			maxID = id + 1
		}
	}
	accessories := make([]*accessory.Accessory, 0)
	id, ok := itemIDs["House Battery"]
	if !ok {
		id = maxID
		maxID++
	}
	accessories, ok = c.registerBatteryLevel(id, "House Battery", accessories)
	if ok {
		itemIDs["House Battery"] = id
	}
	for _, thing := range things {
		if !thing.Editable {
			continue
		}
		if thing.ThingTypeUID == "idsmyrv:hvac-thing" {
			id, ok := itemIDs[thing.UID]
			if !ok {
				maxID++
				id = maxID
				itemIDs[thing.UID] = id
			}
			accessories = c.registerThermostat(id, thing, accessories)
			continue
		}
		if thing.ThingTypeUID == "idsmyrv:generator-thing" {
			id, ok := itemIDs[thing.UID]
			if !ok {
				maxID++
				id = maxID
				itemIDs[thing.UID] = id
			}
			accessories = c.registerGenerator(id, thing, accessories)
			continue
		}
		for _, channel := range thing.Channels {
			registrationMethod, valid := c.getRegistrationMethod(channel)
			if valid {
				item, err := c.habClient.GetItem(channel.ConvertUIDToTingUID())
				if err != nil {
					log.Printf("Error getting item %s from OpenHab: %s", thing.Label, err)
					continue
				}
				id, ok := itemIDs[channel.UID]
				if !ok {
					maxID++
					id = maxID
					itemIDs[channel.UID] = id
				}
				accessories = registrationMethod(id, item, thing.Label, accessories)
				if channel.ChannelTypeUID == "idsmyrv:hsvcolor" {
					break
				}
			}
		}
	}
	itemConfigFile, err = json.Marshal(itemIDs)
	if err != nil {
		log.Printf("Error trying to create config file: %s", err)
	} else {
		err = ioutil.WriteFile("./items.json", itemConfigFile, 0644)
		if err != nil {
			log.Printf("Error trying to save config file: %s", err)
		}
	}

	return accessories
}

func (c *client) registerBatteryLevel(id uint64, name string, accessories []*accessory.Accessory) ([]*accessory.Accessory, bool) {
	ac := accessory.NewHumiditySensor(accessory.Info{
		Name: name,
		ID:   id,
	})
	var bmvClient bmv.Client
	if c.bmvClient != nil {
		bmvClient = *c.bmvClient
	} else {
		return accessories, false
	}
	go func() {
		var lastState float64
		for {
			soc, ok := bmvClient.GetBatteryStateOfCharge()
			if ok && soc != lastState {
				ac.HumiditySensor.CurrentRelativeHumidity.SetValue(soc)
				lastState = soc
			}
			if metrics.StatsEnabled {
				if ok {
					_ = metrics.Metrics.Gauge("battery.stateofcharge", soc, []string{fmt.Sprintf("name:%s", name)}, 1)
				}
				if current, ok := bmvClient.GetBatteryCurrent(); ok {
					_ = metrics.Metrics.Gauge("battery.current", current, []string{fmt.Sprintf("name:%s", name)}, 1)
				}
				if voltage, ok := bmvClient.GetBatteryVoltage(); ok {
					_ = metrics.Metrics.Gauge("battery.voltage", voltage, []string{fmt.Sprintf("name:%s", name)}, 1)
				}
				if amps, ok := bmvClient.GetConsumedAmpHours(); ok {
					_ = metrics.Metrics.Gauge("battery.ampHours", amps, []string{fmt.Sprintf("name:%s", name)}, 1)
				}
				if watts, ok := bmvClient.GetPower(); ok {
					_ = metrics.Metrics.Gauge("battery.watts", watts, []string{fmt.Sprintf("name:%s", name)}, 1)
				}
			}
			time.Sleep(10 * time.Second)
		}
	}()
	ac.HumiditySensor.CurrentRelativeHumidity.SetMinValue(0)
	ac.HumiditySensor.CurrentRelativeHumidity.SetMaxValue(100)
	accessories = append(accessories, ac.Accessory)
	return accessories, true
}

func (c *client) registerTankLevel(id uint64, item openHab.EnrichedItemDTO, name string, accessories []*accessory.Accessory) []*accessory.Accessory {
	ac := accessory.NewHumiditySensor(accessory.Info{
		Name: name,
		ID:   id,
	})
	go func() {
		for {
			lastState := ""
			item.GetCurrentValue()
			level, err := strconv.ParseFloat(item.State, 64)
			if err == nil && metrics.StatsEnabled {
				_ = metrics.Metrics.Gauge("tank.level", level, []string{fmt.Sprintf("name:%s", name)}, 1)
			}
			if item.State != lastState {
				if err == nil {
					ac.HumiditySensor.CurrentRelativeHumidity.SetValue(level)
				}
				lastState = item.State
			}
			time.Sleep(10 * time.Second)
			lastState = item.State

		}
	}()
	ac.HumiditySensor.CurrentRelativeHumidity.SetMinValue(0)
	ac.HumiditySensor.CurrentRelativeHumidity.SetMaxValue(100)
	accessories = append(accessories, ac.Accessory)
	return accessories
}

func (c *client) registerLightBulb(id uint64, item openHab.EnrichedItemDTO, name string, accessories []*accessory.Accessory) []*accessory.Accessory {
	lightbulb := accessory.NewLightbulb(accessory.Info{
		Name: name,
		ID:   id,
	})
	lightbulb.Lightbulb.On.OnValueRemoteUpdate(item.GetChangeFunction())
	go func() {
		lastValue := ""
		for {
			if item.State != lastValue {
				lightbulb.Lightbulb.On.SetValue(item.State == "ON")
			}
			time.Sleep(10 * time.Second)
			item.GetCurrentValue()
		}
	}()
	lightbulb.Lightbulb.On.SetValue(item.State == "ON")
	accessories = append(accessories, lightbulb.Accessory)
	return accessories
}

func (c *client) registerGenerator(id uint64, thing openHab.EnrichedThingDTO, accessories []*accessory.Accessory) []*accessory.Accessory {
	log.Printf("Initializing Generator.")
	ac := accessory.NewSwitch(accessory.Info{
		Name: thing.Label,
		ID:   id,
	})
	channels := make(map[string]openHab.ChannelDTO)
	for _, channel := range thing.Channels {
		channels[channel.UID] = channel
	}
	startStopThing, ok := getThingFromChannels(channels, thing.UID, "command", c.habClient)
	if !ok {
		log.Printf("Unable to get switch for %s, skipping generator.", thing.UID)
		return accessories
	}
	stateThing, ok := getThingFromChannels(channels, thing.UID, "state", c.habClient)
	if !ok {
		log.Printf("Unable to get current state for %s, skipping generator.", thing.UID)
		return accessories
	}
	ac.Switch.On.OnValueRemoteUpdate(startStopThing.GetChangeFunction())
	if c.bmvClient != nil {
		ac.AddBatteryLevel()
	}
	go func() {
		if c.bmvClient != nil {
			bmvClient := *c.bmvClient
			var lastState float64
			var lastCurrent float64
			for {
				if soc, ok := bmvClient.GetBatteryStateOfCharge(); ok && soc != lastState {
					ac.Battery.BatteryLevel.SetValue(int(soc))
					if soc < 10 {
						ac.Battery.StatusLowBattery.SetValue(1)
					} else {
						ac.Battery.StatusLowBattery.SetValue(0)
					}
					lastState = soc
				}
				if current, ok := bmvClient.GetBatteryCurrent(); ok && current != lastCurrent {
					chargeState := 0
					if current > 0 {
						chargeState = 1
					}
					ac.Battery.ChargingState.SetValue(chargeState)
				}
				time.Sleep(10 * time.Second)
			}
		}
	}()
	go func() {
		lastValue := ""
		for {
			if stateThing.State != lastValue {
				ac.Switch.On.SetValue(stateThing.GetCurrentState())
			}
			if metrics.StatsEnabled {
				_ = metrics.Metrics.Gauge("generator.status", float64(c.getGeneratorStatusFromString(stateThing.State)), []string{fmt.Sprintf("name:%s", thing.Label)}, 1)
			}
			time.Sleep(10 * time.Second)
			stateThing.GetCurrentValue()
		}
	}()
	if c.bmvClient != nil {
		bmvClient := *c.bmvClient
		if config, ok := c.config.Automation["generator"]; ok {
			automation.AutomateGeneratorStart(config, bmvClient, startStopThing.GetChangeFunction(), stateThing.GetCurrentState)
		}
	}
	accessories = append(accessories, ac.Accessory)
	return accessories
}

func (c *client) registerSwitch(id uint64, item openHab.EnrichedItemDTO, name string, accessories []*accessory.Accessory) []*accessory.Accessory {
	ac := accessory.NewSwitch(accessory.Info{
		Name: name,
		ID:   id,
	})
	ac.Switch.On.OnValueRemoteUpdate(item.GetChangeFunction())
	go func() {
		lastValue := ""
		for {
			if item.State != lastValue {
				ac.Switch.On.SetValue(item.State == "ON")
			}
			time.Sleep(10 * time.Second)
			item.GetCurrentValue()
		}
	}()
	accessories = append(accessories, ac.Accessory)
	return accessories
}

func (c *client) registerDimmer(id uint64, item openHab.EnrichedItemDTO, name string, accessories []*accessory.Accessory) []*accessory.Accessory {
	lightbulb := accessory.NewLightDimer(accessory.Info{
		Name: name,
		ID:   id,
	})
	lightbulb.LightDimer.On.OnValueRemoteUpdate(item.GetChangeFunction())
	lightbulb.LightDimer.Brightness.OnValueRemoteUpdate(item.ChangeDimmer)
	go func() {
		lastValue := ""
		for {
			if item.State != lastValue {
				lightbulb.LightDimer.On.SetValue(item.State != "0")
				brightness, err := strconv.ParseInt(item.State, 10, 64)
				if err == nil {
					lightbulb.LightDimer.Brightness.SetValue(int(brightness))
				}
			}
			time.Sleep(10 * time.Second)
			item.GetCurrentValue()
		}
	}()
	accessories = append(accessories, lightbulb.Accessory)
	return accessories
}

func (c *client) registerColoredLight(id uint64, item openHab.EnrichedItemDTO, name string, accessories []*accessory.Accessory) []*accessory.Accessory {
	ac := accessory.NewColoredLightbulb(accessory.Info{
		Name: name,
		ID:   id,
	})
	ac.Lightbulb.Hue.MaxValue = 255
	ac.Lightbulb.Hue.MinValue = 0
	ac.Lightbulb.Hue.OnValueRemoteUpdate(item.ChangeHueValue)
	ac.Lightbulb.Saturation.MinValue = 0
	ac.Lightbulb.Saturation.MaxValue = 100
	ac.Lightbulb.Saturation.OnValueRemoteUpdate(item.ChangeSaturationValue)
	ac.Lightbulb.Brightness.MinValue = 0
	ac.Lightbulb.Brightness.MaxValue = 100
	ac.Lightbulb.Brightness.OnValueRemoteUpdate(item.ChangeBrightnessValue)
	ac.Lightbulb.On.OnValueRemoteUpdate(item.ChangeSwitch)
	go func() {
		lastValue := ""
		for {
			if item.State != lastValue {
				hsv := strings.Split(item.State, ",")
				if len(hsv) != 3 {
					break
				}
				ac.Lightbulb.On.SetValue(hsv[2] != "0")
				if hue, err := strconv.ParseFloat(hsv[0], 64); err != nil {
					ac.Lightbulb.Hue.SetValue(hue)
				}
				if saturation, err := strconv.ParseFloat(hsv[1], 64); err != nil {
					ac.Lightbulb.Saturation.SetValue(saturation)
				}
				if brightness, err := strconv.ParseInt(hsv[0], 10, 64); err != nil {
					ac.Lightbulb.Brightness.SetValue(int(brightness))
				}
			}
			time.Sleep(10 * time.Second)
			item.GetCurrentValue()
		}
	}()
	accessories = append(accessories, ac.Accessory)
	return accessories
}

func (c *client) registerNull(_ uint64, _ openHab.EnrichedItemDTO, _ string, accessories []*accessory.Accessory) []*accessory.Accessory {
	return accessories
}

func (c *client) registerThermostat(id uint64, thing openHab.EnrichedThingDTO, accessories []*accessory.Accessory) []*accessory.Accessory {
	channels := make(map[string]openHab.ChannelDTO)
	for _, channel := range thing.Channels {
		channels[channel.UID] = channel
	}
	var units int
	currentTempThing, ok := getThingFromChannels(channels, thing.UID, "inside-temperature", c.habClient)
	if !ok {
		log.Printf("Unable to get current temp for %s, skipping thermostat.", thing.UID)
		return accessories
	}
	switch currentTempThing.Pattern {
	case "%d °F":
		units = 1
	}
	currentTemp, err := strconv.ParseFloat(currentTempThing.State, 64)
	if err != nil {
		log.Printf("Invalid state for current temprature. Got %s", currentTempThing.State)
		return accessories
	}
	modeThing, ok := getThingFromChannels(channels, thing.UID, "hvac-mode", c.habClient)
	if !ok {
		log.Printf("Unable to get current mode for %s, skipping thermostat.", thing.UID)
		return accessories
	}
	statusThing, ok := getThingFromChannels(channels, thing.UID, "status", c.habClient)
	if !ok {
		log.Printf("Unable to get current status for %s, skipping thermostat.", thing.UID)
		return accessories
	}
	highTempThing, ok := getThingFromChannels(channels, thing.UID, "high-temperature", c.habClient)
	if !ok {
		log.Printf("Unable to get high temp for %s, skipping thermostat.", thing.UID)
		return accessories
	}
	lowTempThing, ok := getThingFromChannels(channels, thing.UID, "low-temperature", c.habClient)
	if !ok {
		log.Printf("Unable to get low temp for %s, skipping thermostat.", thing.UID)
		return accessories
	}
	steps := float64(1)
	if units == 1 {
		steps = 1 / 1.8
	}
	var min, max float64
	if c.config.ThermostatRange.MaxValue != 0 {
		if c.config.ThermostatRange.Unit == "f" {
			min = (c.config.ThermostatRange.MinValue - 32) / 1.8
			max = (c.config.ThermostatRange.MaxValue - 32) / 1.8
		} else {
			min = c.config.ThermostatRange.MinValue
			max = c.config.ThermostatRange.MaxValue
		}
	} else {
		min = 10
		max = 38
	}
	ac := accessory.NewThermostat(accessory.Info{
		Name: thing.Label,
		ID:   id,
	}, currentTemp, min, max, steps)
	metricName := strings.Split(thing.Label, " ")
	go func() {
		currentTempState := ""
		currentHVACState := ""
		currentHVACMode := ""
		currentHighTempState := ""
		currentLowTempState := ""
		var currentTemp float64
		for {
			currentTempThing.GetCurrentValue()
			statusThing.GetCurrentValue()
			modeThing.GetCurrentValue()
			lowTempThing.GetCurrentValue()
			highTempThing.GetCurrentValue()
			switch currentTempThing.Pattern {
			case "%d °F":
				units = 1
				break
			}
			currentTemp, err = strconv.ParseFloat(currentTempThing.State, 64)
			if err != nil {
				log.Printf("Invalid state for current temprature. Got %s", currentTempThing.State)
			} else {
				if metrics.StatsEnabled {
					_ = metrics.Metrics.Gauge("hvac.temperature", currentTemp, []string{fmt.Sprintf("name:%s", metricName[0])}, 1)
				}
				if units == 1 {
					currentTemp = (currentTemp - 32) / 1.8
				}

				if currentTempState != currentTempThing.State {
					log.Printf("New temp for %s %v", thing.Label, currentTemp)
					ac.Thermostat.CurrentTemperature.SetValue(currentTemp)
				}

			}
			currentStatus := c.getHVACStatusFromString(statusThing.State)
			if metrics.StatsEnabled {
				_ = metrics.Metrics.Gauge("hvac.currentstatus", float64(c.getHVACStatusNameFromString(statusThing.State)), []string{fmt.Sprintf("name:%s", metricName[0])}, 1)
			}
			if currentHVACState != statusThing.State {
				ac.Thermostat.CurrentHeatingCoolingState.SetValue(currentStatus)
			}
			currentMode := getHVACModeFromString(modeThing.State)
			if metrics.StatsEnabled {
				_ = metrics.Metrics.Gauge("hvac.currentmode", float64(currentMode), []string{fmt.Sprintf("name:%s", metricName[0])}, 1)
			}
			if currentHVACMode != modeThing.State {
				ac.Thermostat.TargetHeatingCoolingState.SetValue(currentMode)
			}
			if currentHighTempState != highTempThing.State || currentLowTempState != lowTempThing.State {
				highTemp, err := strconv.ParseFloat(highTempThing.State, 64)
				if err != nil {
					log.Printf("Invalid state for high temp. Got %s", currentTempThing.State)
					break
				}
				lowTemp, err := strconv.ParseFloat(lowTempThing.State, 64)
				if err != nil {
					log.Printf("Invalid state for low temp. Got %s", currentTempThing.State)
					break
				}
				if units == 1 {
					highTemp = (highTemp - 32) / 1.8
					lowTemp = (lowTemp - 32) / 1.8
				}
				switch getHVACModeFromString(modeThing.State) {
				case 1:
					if currentLowTempState != lowTempThing.State {
						ac.Thermostat.TargetTemperature.SetValue(lowTemp)
					}
				case 2:
					if currentHighTempState != highTempThing.State {
						ac.Thermostat.TargetTemperature.SetValue(highTemp)
					}
				case 3:
					if currentTemp > highTemp {
						ac.Thermostat.TargetTemperature.SetValue(highTemp)
					} else {
						ac.Thermostat.TargetTemperature.SetValue(lowTemp)
					}
				}
			}

			currentTempState = currentTempThing.State
			currentHVACState = statusThing.State
			currentHVACMode = modeThing.State
			currentLowTempState = lowTempThing.State
			currentHighTempState = highTempThing.State
			time.Sleep(10 * time.Second)
		}
	}()
	ac.Thermostat.TemperatureDisplayUnits.SetValue(1)
	ac.Thermostat.TargetHeatingCoolingState.OnValueRemoteUpdate(modeThing.SetHVACToMode)
	ac.Thermostat.TargetTemperature.OnValueRemoteUpdate(func(target float64) {

		highTemp, err := strconv.ParseFloat(highTempThing.State, 64)
		if err != nil {
			log.Printf("Invalid state for high temp. Got %s", currentTempThing.State)
			return
		}
		lowTemp, err := strconv.ParseFloat(lowTempThing.State, 64)
		if err != nil {
			log.Printf("Invalid state for low temp. Got %s", currentTempThing.State)
			return
		}
		offset := float64(3)
		if units == 1 {
			target = (target * 1.8) + 32
			offset = 5
		}
		switch getHVACModeFromString(modeThing.State) {
		case 1:
			lowTempThing.SetTempValue(target)
			highTempThing.SetTempValue(target + offset)
		case 2:
			lowTempThing.SetTempValue(target - offset)
			highTempThing.SetTempValue(target)
		case 3:
			if target < lowTemp {
				lowTempThing.SetTempValue(target)
				highTempThing.SetTempValue(target + offset)
			} else if target > highTemp {
				lowTempThing.SetTempValue(target - offset)
				highTempThing.SetTempValue(target)
			} else {
				if target-lowTemp < highTemp-target {
					lowTempThing.SetTempValue(target)
					highTempThing.SetTempValue(target + offset)
				} else {
					lowTempThing.SetTempValue(target - offset)
					highTempThing.SetTempValue(target)
				}
			}
		}
	})
	accessories = append(accessories, ac.Accessory)
	return accessories
}

func (c *client) getRegistrationMethod(channel openHab.ChannelDTO) (func(id uint64, item openHab.EnrichedItemDTO, name string, accessories []*accessory.Accessory) []*accessory.Accessory, bool) {
	switch channel.ChannelTypeUID {
	case "idsmyrv:switch":
		return c.registerSwitch, true
	case "idsmyrv:switched-light":
		return c.registerLightBulb, true
	case "idsmyrv:dimmer":
		return c.registerDimmer, true
	case "idsmyrv:hsvcolor":
		return c.registerColoredLight, true
	case "idsmyrv:level":
		return c.registerTankLevel, true
	default:
		return c.registerNull, false
	}
}

func (c *client) getGeneratorStatusFromString(status string) int {
	switch status {
	case "OFF":
		return 0
	case "PRIMING":
		return 1
	case "STARTING":
		return 2
	case "RUNNING":
		return 3
	case "STOPPING":
		return 4
	default:
		return 0
	}
}

func (c *client) getHVACStatusFromString(status string) int {
	switch status {
	case "OFF", "IDLE":
		return 0
	case "COOLING":
		return 2
	case "HEAT_PUMP", "ELEC_FURNACE", "GAS_FURNACE", "GAS_OVERRIDE":
		return 1
	default:
		return 0
	}
}
func (c *client) getHVACStatusNameFromString(status string) int {
	switch status {
	case "OFF":
		return 0
	case "IDLE":
		return 1
	case "COOLING":
		return 2
	case "HEAT_PUMP":
		return 3
	case "ELEC_FURNACE":
		return 4
	case "GAS_FURNACE":
		return 5
	case "GAS_OVERRIDE":
		return 6
	case "DEAD_TIME":
		return 7
	case "LOAD_SHEDDING":
		return 8
	case "FAIL_OFF":
		return 9
	case "FAIL_IDLE":
		return 10
	case "FAIL_COOLING":
		return 11
	case "FAIL_HEAT_PUMP":
		return 12
	case "FAIL_ELEC_FURNACE":
		return 13
	case "FAIL_GAS_FURNACE":
		return 14
	case "FAIL_GAS_OVERRIDE":
		return 15
	case "FAIL_DEAD_TIME":
		return 16
	case "FAIL_SHEDDING":
		return 17
	default:
		return 0
	}
}

func getHVACModeFromString(mode string) int {
	switch mode {
	case "HEAT":
		return 1
	case "COOL":
		return 2
	case "HEATCOOL":
		return 3
	default:
		return 0
	}
}

func getThingFromChannels(channels map[string]openHab.ChannelDTO, thingID string, id string, client openHab.Client) (openHab.EnrichedItemDTO, bool) {
	channel, ok := channels[fmt.Sprintf("%s:%s", thingID, id)]
	if !ok {
		return openHab.EnrichedItemDTO{}, false
	}
	thing, err := client.GetItem(channel.ConvertUIDToTingUID())
	if err != nil {
		return openHab.EnrichedItemDTO{}, false
	}
	return thing, true
}
