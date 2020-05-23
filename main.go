package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/jgulick48/hc"
	"github.com/jgulick48/hc/accessory"

	"github.com/jgulick48/rv-homekit/internal/openHab"
)

type Config struct {
	BridgeName    string `json:"bridgeName"`
	OpenHabServer string `json:"openHabServer"`
	PIN           string `json:"pin"`
}

func main() {
	var itemIDs map[string]uint64
	configFile, err := ioutil.ReadFile("./config.json")
	if err != nil {
		log.Printf("No config file found. Making new IDs")
		panic(err)
	}
	var config Config
	err = json.Unmarshal(configFile, &config)
	if err != nil {
		log.Printf("Invliad config file provided")
		panic(err)
	}
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
	bridge := accessory.NewBridge(accessory.Info{
		Name: config.BridgeName,
		ID:   1,
	})
	itemIDs["bridge"] = 1

	habClient := openHab.NewClient(config.OpenHabServer)
	things, err := habClient.GetThings()
	if err != nil {
		panic(err)
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
			accessories = registerThermostat(id, thing, habClient, accessories)
			continue
		}
		for _, channel := range thing.Channels {
			registrationMethod, valid := getRegistrationMethod(channel)
			if valid {
				item, err := habClient.GetItem(channel.ConvertUIDToTingUID())
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
	log.Printf("Found %v items", len(accessories))
	itemConfigFile, err = json.Marshal(itemIDs)
	if err != nil {
		log.Printf("Error trying to create config file: %s", err)
	} else {
		err = ioutil.WriteFile("./items.json", itemConfigFile, 0644)
		if err != nil {
			log.Printf("Error trying to save config file: %s", err)
		}
	}
	// configure the ip transport
	hcConfig := hc.Config{
		Pin: config.PIN,
	}
	t, err := hc.NewIPTransport(hcConfig, bridge.Accessory, accessories...)
	if err != nil {
		log.Panic(err)
	}

	hc.OnTermination(func() {
		<-t.Stop()
	})
	t.Start()
}

func registerTankLevel(id uint64, item openHab.EnrichedItemDTO, name string, accessories []*accessory.Accessory) []*accessory.Accessory {
	ac := accessory.NewHumiditySensor(accessory.Info{
		Name: name,
		ID:   id,
	})
	go func() {
		lastState := ""
		if item.State != lastState {
			level, err := strconv.ParseFloat(item.State, 64)
			if err == nil {
				ac.HumiditySensor.CurrentRelativeHumidity.SetValue(level)
			}

		}
	}()
	ac.HumiditySensor.CurrentRelativeHumidity.SetMinValue(0)
	ac.HumiditySensor.CurrentRelativeHumidity.SetMaxValue(100)
	accessories = append(accessories, ac.Accessory)
	return accessories
}

func registerLightBulb(id uint64, item openHab.EnrichedItemDTO, name string, accessories []*accessory.Accessory) []*accessory.Accessory {
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

func registerSwitch(id uint64, item openHab.EnrichedItemDTO, name string, accessories []*accessory.Accessory) []*accessory.Accessory {
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

func registerDimmer(id uint64, item openHab.EnrichedItemDTO, name string, accessories []*accessory.Accessory) []*accessory.Accessory {
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

func registerColoredLight(id uint64, item openHab.EnrichedItemDTO, name string, accessories []*accessory.Accessory) []*accessory.Accessory {
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

func registerNull(_ uint64, _ openHab.EnrichedItemDTO, _ string, accessories []*accessory.Accessory) []*accessory.Accessory {
	return accessories
}

func registerThermostat(id uint64, thing openHab.EnrichedThingDTO, client openHab.Client, accessories []*accessory.Accessory) []*accessory.Accessory {
	channels := make(map[string]openHab.ChannelDTO)
	for _, channel := range thing.Channels {
		channels[channel.UID] = channel
	}
	var units int
	currentTempThing, ok := getThingFromChannels(channels, thing.UID, "inside-temperature", client)
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
	modeThing, ok := getThingFromChannels(channels, thing.UID, "hvac-mode", client)
	if !ok {
		log.Printf("Unable to get current mode for %s, skipping thermostat.", thing.UID)
		return accessories
	}
	statusThing, ok := getThingFromChannels(channels, thing.UID, "status", client)
	if !ok {
		log.Printf("Unable to get current status for %s, skipping thermostat.", thing.UID)
		return accessories
	}
	highTempThing, ok := getThingFromChannels(channels, thing.UID, "high-temperature", client)
	if !ok {
		log.Printf("Unable to get high temp for %s, skipping thermostat.", thing.UID)
		return accessories
	}
	lowTempThing, ok := getThingFromChannels(channels, thing.UID, "low-temperature", client)
	if !ok {
		log.Printf("Unable to get low temp for %s, skipping thermostat.", thing.UID)
		return accessories
	}
	steps := float64(1)
	if units == 1 {
		steps = 1 / 1.8
	}
	ac := accessory.NewThermostat(accessory.Info{
		Name: thing.Label,
		ID:   id,
	}, currentTemp, 10, 38, steps)
	go func() {
		currentTempState := ""
		currentHVACState := ""
		currentHVACMode := ""
		currentHighTempState := ""
		currentLowTempState := ""
		var currentTemp float64
		for {
			if currentTempState != currentTempThing.State {
				switch currentTempThing.Pattern {
				case "%d °F":
					units = 1
					break
				}
				currentTemp, err = strconv.ParseFloat(currentTempThing.State, 64)
				if err != nil {
					log.Printf("Invalid state for current temprature. Got %s", currentTempThing.State)
				} else {
					if units == 1 {
						currentTemp = (currentTemp - 32) / 1.8
					}
					ac.Thermostat.CurrentTemperature.SetValue(currentTemp)
				}
			}
			if currentHVACState != statusThing.State {
				ac.Thermostat.CurrentHeatingCoolingState.SetValue(getHVACStatusFromString(statusThing.State))
			}
			if currentHVACMode != modeThing.State {
				ac.Thermostat.TargetHeatingCoolingState.SetValue(getHVACModeFromString(modeThing.State))
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

			time.Sleep(10 * time.Second)
			currentTempThing.GetCurrentValue()
			currentTempState = currentTempThing.State
			statusThing.GetCurrentValue()
			currentHVACState = statusThing.State
			modeThing.GetCurrentValue()
			currentHVACMode = modeThing.State
			lowTempThing.GetCurrentValue()
			currentLowTempState = lowTempThing.State
			highTempThing.GetCurrentValue()
			currentHighTempState = highTempThing.State
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

func getRegistrationMethod(channel openHab.ChannelDTO) (func(id uint64, item openHab.EnrichedItemDTO, name string, accessories []*accessory.Accessory) []*accessory.Accessory, bool) {
	switch channel.ChannelTypeUID {
	case "idsmyrv:switch":
		return registerSwitch, true
	case "idsmyrv:switched-light":
		return registerLightBulb, true
	case "idsmyrv:dimmer":
		return registerDimmer, true
	case "idsmyrv:hsvcolor":
		return registerColoredLight, true
	case "idsmyrv:level":
		return registerTankLevel, true
	default:
		return registerNull, false
	}
}

func getHVACStatusFromString(status string) int {
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
