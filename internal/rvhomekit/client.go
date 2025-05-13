package rvhomekit

import (
	"encoding/json"
	"fmt"
	"github.com/jgulick48/rv-homekit/internal/openevse"
	"github.com/jgulick48/rv-homekit/internal/tanksensors"
	"io/ioutil"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/jgulick48/hc/accessory"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/jgulick48/rv-homekit/internal/automation"
	"github.com/jgulick48/rv-homekit/internal/bmv"
	"github.com/jgulick48/rv-homekit/internal/metrics"
	"github.com/jgulick48/rv-homekit/internal/models"
	"github.com/jgulick48/rv-homekit/internal/mqtt"
	"github.com/jgulick48/rv-homekit/internal/openHab"
)

type client struct {
	config      models.Config
	habClient   openHab.Client
	bmvClient   *bmv.Client
	tankSensors tanksensors.Client
	mqttClient  mqtt.Client
	evseClient  *openevse.Client
	syncFuncs   []func()
}

type Client interface {
	GetAccessoriesFromOpenHab(things []openHab.EnrichedThingDTO) []*accessory.Accessory
	SaveClientConfig(filename string)
	RunSyncFunctions()
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

func (c *client) SaveClientConfig(filename string) {
	if filename == "" {
		filename = "./config.json"
	}
	data, err := json.MarshalIndent(c.config, "", "  ")
	if err != nil {
		return
	}
	ioutil.WriteFile(filename, data, 0644)
}

func NewClient(config models.Config, habClient openHab.Client, bmvClient *bmv.Client, tankSensors tanksensors.Client, mqttClient mqtt.Client, evseClient *openevse.Client) Client {
	prometheus.MustRegister(
		batteryAmpHours,
		batteryAutoChargeStarted,
		batteryAutoChargeState,
		batteryChargeTimeRemaining,
		batteryCurrent,
		batteryStateOfCharge,
		batteryTemperature,
		batteryTimeRemaining,
		batteryVolts,
		batteryWatts,
		tankBatteryPercent,
		tankBatteryVoltage,
		tankLevel,
		tankLevelMM,
		tankSensorQuality,
		tankSensorRSSI,
		tankTempCelsius,
		tankTempFahrenheit,
		generatorStatus,
		hvacCurrentMode,
		hvacCurrentStatus,
		hvacTemperature,
	)
	return &client{
		config:      config,
		habClient:   habClient,
		bmvClient:   bmvClient,
		tankSensors: tankSensors,
		mqttClient:  mqttClient,
		evseClient:  evseClient,
		syncFuncs:   make([]func(), 0),
	}
}

func (c *client) RunSyncFunctions() {
	start := time.Now()
	for _, syncFunc := range c.syncFuncs {
		syncFunc()
	}
	end := time.Now()
	if metrics.StatsEnabled {
		metrics.SendGaugeMetricWithRate("syncFunc.duration", float64(end.Sub(start).Milliseconds()), []string{fmt.Sprintf("name:%s", c.config.BridgeName)}, 1)
	}
	log.Printf("Sync functions took %f seconds", end.Sub(start).Seconds())
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
	id, ok = itemIDs["EVSE"]
	if !ok {
		id = maxID
		maxID++
	}
	if c.config.EVSEConfiguration.Enabled && c.config.EVSEConfiguration.Address != "" {
		accessories, ok = c.registerEVSE(id, c.evseClient, "EVSE", accessories)
	}
	if ok {
		itemIDs["EVSE"] = id
	}
	var foundTankSensors int
	if c.tankSensors != nil {
		itemIDs, accessories, maxID, foundTankSensors = c.registerTankSensors(itemIDs, accessories)
	} else {
		log.Printf("Tank sensors not configured skipping.")
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
			fmt.Printf("Found %s : %s\n", thing.Label, thing.UID)

			continue
		}
		if thing.ThingTypeUID == "idsmyrv:generator-thing" {
			id, ok := itemIDs[thing.UID]
			if !ok {
				maxID++
				id = maxID
				itemIDs[thing.UID] = id
			}
			var generatorAutomation automation.Automation
			accessories, generatorAutomation = c.registerGenerator(id, thing, accessories)
			fmt.Printf("Found %s : %s\n", thing.Label, thing.UID)
			if generatorAutomation.IsEnabled() {
				id, ok := itemIDs["BatteryAutoCharge"]
				if !ok {
					maxID++
					id = maxID
					itemIDs["BatteryAutoCharge"] = id
				}
				accessories = c.registerGeneratorAutomation(id, generatorAutomation, accessories)
			}
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
				fmt.Printf("Found %s : %s\n", thing.Label, thing.UID)
				if channel.ChannelTypeUID == "idsmyrv:hsvcolor" {
					break
				}
			}
		}
	}
	itemConfigFile, err = json.MarshalIndent(itemIDs, "", "  ")
	if err != nil {
		log.Printf("Error trying to create config file: %s", err)
	} else {
		err = ioutil.WriteFile("./items.json", itemConfigFile, 0644)
		if err != nil {
			log.Printf("Error trying to save config file: %s", err)
		}
	}
	if c.config.CrashOnDeviceMismatch && len(accessories) != (len(itemIDs)+foundTankSensors) {
		for _, i := range accessories {
			log.Printf("%s, %v", i.Info.Name.GetValue(), i.ID)
		}
		log.Fatalf("Found %v items expected to find %v exiting due to config. Expected:\n%s", len(accessories), len(itemIDs), itemConfigFile)
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
	} else if c.mqttClient.IsEnabled() {
		go func() {
			for {
				time.Sleep(time.Second * 10)
				c.mqttClient.Connect()
				log.Printf("Mqtt connection lost, reconnecting.")
			}
		}()
		bmvClient = c.mqttClient.GetBatteryClient()
	} else {
		return accessories, false
	}
	var lastState float64
	syncFunc := func() {
		soc, ok := bmvClient.GetBatteryStateOfCharge()
		if ok && soc != lastState {
			ac.HumiditySensor.CurrentRelativeHumidity.SetValue(soc)
			lastState = soc
		}
		if metrics.StatsEnabled {
			if ok {
				metrics.SendGaugeMetricWithRate("battery.stateofcharge", soc, []string{fmt.Sprintf("name:%s", name)}, 1)
				batteryStateOfCharge.WithLabelValues(name).Set(soc)
			}
			if current, ok := bmvClient.GetBatteryCurrent(); ok {
				metrics.SendGaugeMetricWithRate("battery.current", current, []string{fmt.Sprintf("name:%s", name)}, 1)
				batteryCurrent.WithLabelValues(name).Set(current)
			}
			if voltage, ok := bmvClient.GetBatteryVoltage(); ok {
				metrics.SendGaugeMetricWithRate("battery.voltage", voltage, []string{fmt.Sprintf("name:%s", name)}, 1)
				batteryVolts.WithLabelValues(name).Set(voltage)
			}
			if amps, ok := bmvClient.GetConsumedAmpHours(); ok {
				metrics.SendGaugeMetricWithRate("battery.ampHours", amps, []string{fmt.Sprintf("name:%s", name)}, 1)
				batteryAmpHours.WithLabelValues(name).Set(amps)
			}
			if watts, ok := bmvClient.GetPower(); ok {
				metrics.SendGaugeMetricWithRate("battery.watts", watts, []string{fmt.Sprintf("name:%s", name)}, 1)
				batteryWatts.WithLabelValues(name).Set(watts)
			}
			if temp, ok := bmvClient.GetBatteryTemperature(); ok {
				metrics.SendGaugeMetricWithRate("battery.celsius", temp, []string{fmt.Sprintf("name:%s", name)}, 1)
				batteryTemperature.WithLabelValues(name, "celsius").Set(temp)
				metrics.SendGaugeMetricWithRate("battery.fahrenheit", (temp*1.8)+32, []string{fmt.Sprintf("name:%s", name)}, 1)
				batteryTemperature.WithLabelValues(name, "fahrenheit").Set((temp * 1.8) + 32)
			}
			if timeRemaining, ok := bmvClient.GetTimeToGo(); ok {
				metrics.SendGaugeMetricWithRate("battery.timeRemaining", timeRemaining, []string{fmt.Sprintf("name:%s", name)}, 1)
				batteryTimeRemaining.WithLabelValues(name).Set(timeRemaining)
			}
			if chargeTimeRemaining, ok := bmvClient.GetChargeTimeRemaining(); ok {
				metrics.SendGaugeMetricWithRate("battery.chargeTimeRemaining", chargeTimeRemaining, []string{fmt.Sprintf("name:%s", name)}, 1)
				batteryChargeTimeRemaining.WithLabelValues(name).Set(chargeTimeRemaining)
			}
		}
	}
	syncFunc()
	c.syncFuncs = append(c.syncFuncs, syncFunc)
	ac.HumiditySensor.CurrentRelativeHumidity.SetMinValue(0)
	ac.HumiditySensor.CurrentRelativeHumidity.SetMaxValue(100)
	accessories = append(accessories, ac.Accessory)
	return accessories, true
}

func (c *client) registerTankSensors(itemIds map[string]uint64, accessories []*accessory.Accessory) (map[string]uint64, []*accessory.Accessory, uint64, int) {
	foundSensors := 0
	maxID := uint64(0)
	for _, id := range itemIds {
		if id > maxID {
			maxID = id + 1
		}
	}
	devices := c.tankSensors.GetDevices()
	log.Printf("Found %v tank sensors.", len(devices))
	for i, device := range devices {
		deviceConfig, ok := c.matchLevelSensorWithConfig(device.GetAddress())
		if !ok {
			if !c.config.TankSensors.RegisterNew {
				continue
			}
			deviceConfig = models.MopekaLevelSensor{
				Address:   device.GetAddress(),
				Name:      fmt.Sprintf("Tank %v", i+1),
				Type:      c.config.TankSensors.DefaultTankType,
				MaxHeight: 0,
			}
			c.config.TankSensors.Devices = append(c.config.TankSensors.Devices, deviceConfig)
		}
		var id uint64
		if id, ok = itemIds[device.GetAddress()]; !ok {
			id = maxID
			maxID += 2
		}
		accessories = c.registerTankSensor(id, accessories, deviceConfig)
		itemIds[device.GetAddress()] = id
		deviceConfig.Discovered = true
		foundSensors++
	}
	undiscovered := 0
	for _, deviceConfig := range c.config.TankSensors.Devices {
		if !deviceConfig.Discovered {
			var id uint64
			var ok bool
			if id, ok = itemIds[deviceConfig.Address]; !ok {
				id = maxID
				maxID += 2
			}
			accessories = c.registerTankSensor(id, accessories, deviceConfig)
			itemIds[deviceConfig.Address] = id
			undiscovered++
			foundSensors++
		}
	}
	log.Printf("Registered %v undiscovered devices.", undiscovered)
	return itemIds, accessories, maxID, foundSensors
}

func (c *client) matchLevelSensorWithConfig(address string) (models.MopekaLevelSensor, bool) {
	for i, device := range c.config.TankSensors.Devices {
		if strings.ToLower(device.Address) == strings.ToLower(address) {
			c.config.TankSensors.Devices[i].Discovered = true
			return device, true
		}
	}
	return models.MopekaLevelSensor{
		Address: address,
	}, false
}

func (c *client) registerTankSensor(id uint64, accessories []*accessory.Accessory, deviceConfig models.MopekaLevelSensor) []*accessory.Accessory {
	name := deviceConfig.Name
	ac1 := accessory.NewHumiditySensor(accessory.Info{
		Name:         fmt.Sprintf("%s Level", name),
		SerialNumber: "",
		ID:           id,
	})
	ac2 := accessory.NewTemperatureSensor(accessory.Info{
		Name:         name,
		SerialNumber: deviceConfig.Address,
		ID:           id + 1,
	}, 0, -40, 100, 1)
	temp := float64(10)
	level := float64(100)
	ac2.TempSensor.CurrentTemperature.SetValue(temp)
	ac1.HumiditySensor.CurrentRelativeHumidity.SetValue(level)
	syncFunc := func() {
		if device, ok := c.tankSensors.GetDevice(strings.ToLower(deviceConfig.Address)); ok {
			lastTemp := temp
			lastLevel := level
			if temp = device.GetTempCelsius(); lastTemp != temp {
				ac2.TempSensor.CurrentTemperature.SetValue(temp)
			}
			if level = device.GetLevelPercent(deviceConfig.Type); lastLevel != level {
				ac1.HumiditySensor.CurrentRelativeHumidity.SetValue(level)
				log.Printf("got new tank level of %v for %s", level, name)
			}
			if metrics.StatsEnabled {
				metrics.SendGaugeMetricWithRate("tank.level", level, []string{fmt.Sprintf("name:%s", name), fmt.Sprintf("type:%s", device.GetSensorType())}, 1)
				tankLevel.WithLabelValues(name, device.GetSensorType()).Set(level)
				metrics.SendGaugeMetricWithRate("tank.levelMM", device.GetTankLevelMM(), []string{fmt.Sprintf("name:%s", name)}, 1)
				tankLevelMM.WithLabelValues(name).Set(device.GetTankLevelMM())
				metrics.SendGaugeMetricWithRate("tank.tempCelsius", temp, []string{fmt.Sprintf("name:%s", name)}, 1)
				tankTempCelsius.WithLabelValues(name).Set(temp)
				metrics.SendGaugeMetricWithRate("tank.tempFahrenheit", device.GetTempFahrenheit(), []string{fmt.Sprintf("name:%s", name)}, 1)
				tankTempFahrenheit.WithLabelValues(name).Set(device.GetTempFahrenheit())
				metrics.SendGaugeMetricWithRate("tank.batteryPercent", float64(device.GetBatteryLevel()), []string{fmt.Sprintf("name:%s", name)}, 1)
				tankBatteryPercent.WithLabelValues(name).Set(float64(device.GetBatteryLevel()))
				metrics.SendGaugeMetricWithRate("tank.batteryVoltage", device.GetBatteryVoltage(), []string{fmt.Sprintf("name:%s", name)}, 1)
				tankBatteryVoltage.WithLabelValues(name).Set(device.GetBatteryVoltage())
				metrics.SendGaugeMetricWithRate("tank.sensorQuality", device.GetReadQuality(), []string{fmt.Sprintf("name:%s", name)}, 1)
				tankSensorQuality.WithLabelValues(name).Set(device.GetReadQuality())
				metrics.SendGaugeMetricWithRate("tank.sensorRSSI", float64(device.GetRSSI()), []string{fmt.Sprintf("name:%s", name)}, 1)
				tankSensorRSSI.WithLabelValues(name).Set(float64(device.GetRSSI()))
			}
		} else {
			if c.config.Debug {
				log.Printf("Device with name of %s and address of %s was not found.", deviceConfig.Name, deviceConfig.Address)
			}
		}
	}
	syncFunc()
	c.syncFuncs = append(c.syncFuncs, syncFunc)
	ac1.HumiditySensor.CurrentRelativeHumidity.SetMinValue(0)
	ac1.HumiditySensor.CurrentRelativeHumidity.SetMaxValue(100)
	accessories = append(accessories, ac1.Accessory)
	accessories = append(accessories, ac2.Accessory)
	return accessories
}

func (c *client) registerTankLevel(id uint64, item openHab.EnrichedItemDTO, name string, accessories []*accessory.Accessory) []*accessory.Accessory {
	ac := accessory.NewHumiditySensor(accessory.Info{
		Name: name,
		ID:   id,
	})
	lastState := ""
	syncFunc := func() {
		item.GetCurrentValue()
		level, err := strconv.ParseFloat(item.State, 64)
		if err == nil && metrics.StatsEnabled {
			metrics.SendGaugeMetricWithRate("tank.level", level, []string{fmt.Sprintf("name:%s", name)}, 1)
			tankLevel.WithLabelValues(name, "OneControl").Set(level)
		}
		if item.State != lastState {
			if err == nil {
				ac.HumiditySensor.CurrentRelativeHumidity.SetValue(level)
			}
			lastState = item.State
		}
		lastState = item.State
	}
	syncFunc()
	c.syncFuncs = append(c.syncFuncs, syncFunc)
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
	lastValue := ""
	syncFunc := func() {
		item.GetCurrentValue()
		if item.State != lastValue {
			lightbulb.Lightbulb.On.SetValue(item.State == "ON")
		}
		lastValue = item.State
	}
	syncFunc()
	c.syncFuncs = append(c.syncFuncs, syncFunc)
	lightbulb.Lightbulb.On.SetValue(item.State == "ON")
	accessories = append(accessories, lightbulb.Accessory)
	return accessories
}

func (c *client) registerGeneratorAutomation(id uint64, generatorAutomation automation.Automation, accessories []*accessory.Accessory) []*accessory.Accessory {
	if !generatorAutomation.IsEnabled() {
		return accessories
	}
	ac := accessory.NewSwitch(accessory.Info{
		Name: "Battery AutoCharge",
		ID:   id,
	})
	ac.Switch.On.OnValueRemoteUpdate(func(shouldStart bool) {
		if shouldStart {
			generatorAutomation.StartAutoCharge()
		} else {
			generatorAutomation.StopAutoCharge()
		}
	})
	lastState := false
	ac.Switch.On.SetValue(generatorAutomation.IsAutomationRunning())
	syncFunc := func() {
		if generatorAutomation.IsAutomationRunning() != lastState {
			ac.Switch.On.SetValue(generatorAutomation.IsAutomationRunning())
			lastState = generatorAutomation.IsAutomationRunning()
		}
		if metrics.StatsEnabled {
			value := float64(0)
			if lastState {
				value = 1
			}
			metrics.SendGaugeMetricWithRate("battery.autocharge.started", value, []string{}, 1)
			batteryAutoChargeStarted.WithLabelValues().Set(value)
			if generatorAutomation.IsEnabled() {
				metrics.SendGaugeMetricWithRate("battery.autocharge.state", 1, []string{}, 1)
				batteryAutoChargeState.WithLabelValues().Set(1)
			} else {
				metrics.SendGaugeMetricWithRate("battery.autocharge.state", 0, []string{}, 1)
				batteryAutoChargeState.WithLabelValues().Set(0)
			}
		}
	}
	syncFunc()
	c.syncFuncs = append(c.syncFuncs, syncFunc)
	accessories = append(accessories, ac.Accessory)
	return accessories
}

func (c *client) registerGenerator(id uint64, thing openHab.EnrichedThingDTO, accessories []*accessory.Accessory) ([]*accessory.Accessory, automation.Automation) {
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
		return accessories, automation.Automation{}
	}
	stateThing, ok := getThingFromChannels(channels, thing.UID, "state", c.habClient)
	if !ok {
		log.Printf("Unable to get current state for %s, skipping generator.", thing.UID)
		return accessories, automation.Automation{}
	}
	ac.Switch.On.OnValueRemoteUpdate(func(state bool) {
		changeStateFunc := startStopThing.GetChangeFunction()
		if !state {
			time.Sleep(c.config.GeneratorOffDelay.Duration)
		}
		changeStateFunc(state)
	})
	if c.bmvClient != nil {
		ac.AddBatteryLevel()
	}
	var lastState float64
	var lastCurrent float64
	syncFunc := func() {
		if c.bmvClient != nil {
			bmvClient := *c.bmvClient
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
				lastCurrent = current
			}
		}
	}
	syncFunc()
	c.syncFuncs = append(c.syncFuncs, syncFunc)
	lastValue := ""
	syncFunc2 := func() {
		stateThing.GetCurrentValue()
		if stateThing.State != lastValue {
			ac.Switch.On.SetValue(stateThing.GetCurrentState())
		}
		if metrics.StatsEnabled {
			metrics.SendGaugeMetricWithRate("generator.status", float64(c.getGeneratorStatusFromString(stateThing.State)), []string{fmt.Sprintf("name:%s", thing.Label)}, 1)
			generatorStatus.WithLabelValues(thing.Label).Set(float64(c.getGeneratorStatusFromString(stateThing.State)))
		}
		lastValue = stateThing.State
	}
	syncFunc2()
	var generatorAutomation automation.Automation
	c.syncFuncs = append(c.syncFuncs, syncFunc2)
	if c.bmvClient != nil {
		bmvClient := *c.bmvClient
		if config, ok := c.config.Automation["generator"]; ok {
			generatorAutomation = automation.NewGeneratorAutomationClient(config, bmvClient, c.mqttClient, c.config.DVCCConfiguration, c.config.InputLimitConfiguration, startStopThing.GetChangeFunction(), stateThing.GetCurrentState)
			generatorAutomation.AutomateGeneratorStart()
		}
	} else if c.mqttClient.IsEnabled() {
		bmvClient := c.mqttClient.GetBatteryClient()
		if config, ok := c.config.Automation["generator"]; ok {
			generatorAutomation = automation.NewGeneratorAutomationClient(config, bmvClient, c.mqttClient, c.config.DVCCConfiguration, c.config.InputLimitConfiguration, startStopThing.GetChangeFunction(), stateThing.GetCurrentState)
			generatorAutomation.AutomateGeneratorStart()
		}
	}
	accessories = append(accessories, ac.Accessory)
	return accessories, generatorAutomation
}

func (c *client) registerSwitch(id uint64, item openHab.EnrichedItemDTO, name string, accessories []*accessory.Accessory) []*accessory.Accessory {
	ac := accessory.NewSwitch(accessory.Info{
		Name: name,
		ID:   id,
	})
	ac.Switch.On.OnValueRemoteUpdate(item.GetChangeFunction())
	if name == "Electric Water Heater" && c.mqttClient.IsEnabled() {
		c.mqttClient.RegisterOpenHabHPDevice(&item)
	}
	lastValue := ""
	syncFunc := func() {
		item.GetCurrentValue()
		if item.State != lastValue {
			ac.Switch.On.SetValue(item.State == "ON")
		}
		lastValue = item.State
	}
	syncFunc()
	c.syncFuncs = append(c.syncFuncs, syncFunc)
	accessories = append(accessories, ac.Accessory)
	return accessories
}

func (c *client) registerEVSE(id uint64, item *openevse.Client, name string, accessories []*accessory.Accessory) ([]*accessory.Accessory, bool) {
	if item == nil {
		return accessories, false
	}
	ac := accessory.NewSwitch(accessory.Info{
		Name: name,
		ID:   id,
	})
	ac.Switch.On.OnValueRemoteUpdate(item.Enable)
	if c.mqttClient.IsEnabled() {
		if c.config.EVSEConfiguration.Enabled && c.config.EVSEConfiguration.EnableControl {
			c.mqttClient.RegisterEVSEHPDevice(item)
		}
	}
	lastValue := ""
	syncFunc := func() {
		state, _ := item.GetState()
		if state != lastValue {
			ac.Switch.On.SetValue(state == "ON")
		}
		lastValue = state
	}
	syncFunc()
	c.syncFuncs = append(c.syncFuncs, syncFunc)
	accessories = append(accessories, ac.Accessory)
	return accessories, true
}

func (c *client) registerDimmer(id uint64, item openHab.EnrichedItemDTO, name string, accessories []*accessory.Accessory) []*accessory.Accessory {
	lightbulb := accessory.NewLightDimer(accessory.Info{
		Name: name,
		ID:   id,
	})
	lightbulb.LightDimer.On.OnValueRemoteUpdate(item.GetChangeFunction())
	lightbulb.LightDimer.Brightness.OnValueRemoteUpdate(item.ChangeDimmer)
	lastValue := ""
	syncFunc := func() {
		item.GetCurrentValue()
		if item.State != lastValue {
			lightbulb.LightDimer.On.SetValue(item.State != "0")
			brightness, err := strconv.ParseInt(item.State, 10, 64)
			if err == nil {
				lightbulb.LightDimer.Brightness.SetValue(int(brightness))
			}
		}
		lastValue = item.State
	}
	syncFunc()
	c.syncFuncs = append(c.syncFuncs, syncFunc)
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
	lastValue := ""
	syncFunc := func() {
		item.GetCurrentValue()
		if item.State != lastValue {
			hsv := strings.Split(item.State, ",")
			if len(hsv) != 3 {
				return
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
		lastValue = item.State
	}
	syncFunc()
	c.syncFuncs = append(c.syncFuncs, syncFunc)
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
	if c.mqttClient.IsEnabled() {
		c.mqttClient.RegisterOpenHabHPDevice(&modeThing)
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
	currentTempState := ""
	currentHVACState := ""
	currentHVACMode := ""
	currentHighTempState := ""
	currentLowTempState := ""
	syncFunc := func() {
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
				metrics.SendGaugeMetricWithRate("hvac.temperature", currentTemp, []string{fmt.Sprintf("name:%s", metricName[0])}, 1)
				hvacTemperature.WithLabelValues(metricName[0]).Set(currentTemp)
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
			metrics.SendGaugeMetricWithRate("hvac.currentstatus", float64(c.getHVACStatusNameFromString(statusThing.State)), []string{fmt.Sprintf("name:%s", metricName[0])}, 1)
			hvacCurrentStatus.WithLabelValues(metricName[0]).Set(float64(c.getHVACStatusNameFromString(statusThing.State)))
		}
		if currentHVACState != statusThing.State {
			ac.Thermostat.CurrentHeatingCoolingState.SetValue(currentStatus)
		}
		currentMode := getHVACModeFromString(modeThing.State)
		if metrics.StatsEnabled {
			metrics.SendGaugeMetricWithRate("hvac.currentmode", float64(currentMode), []string{fmt.Sprintf("name:%s", metricName[0])}, 1)
			hvacCurrentMode.WithLabelValues(metricName[0]).Set(float64(currentMode))
		}
		if currentHVACMode != modeThing.State {
			ac.Thermostat.TargetHeatingCoolingState.SetValue(currentMode)
		}
		if currentHighTempState != highTempThing.State || currentLowTempState != lowTempThing.State {
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
				ac.Thermostat.CoolingThresholdTemperature.SetValue(highTemp)
				ac.Thermostat.HeatingThresholdTemperature.SetValue(lowTemp)
			default:
				if currentHighTempState != highTempThing.State {
					ac.Thermostat.TargetTemperature.SetValue(highTemp)
				}
			}
		}

		currentTempState = currentTempThing.State
		currentHVACState = statusThing.State
		currentHVACMode = modeThing.State
		currentLowTempState = lowTempThing.State
		currentHighTempState = highTempThing.State
	}
	syncFunc()
	c.syncFuncs = append(c.syncFuncs, syncFunc)
	ac.Thermostat.TemperatureDisplayUnits.SetValue(1)
	ac.Thermostat.TargetHeatingCoolingState.OnValueRemoteUpdate(modeThing.SetHVACToMode)
	ac.Thermostat.HeatingThresholdTemperature.OnValueRemoteUpdate(func(target float64) {
		if units == 1 {
			target = (target * 1.8) + 32
			target = math.Round(target)
		}
		log.Printf("Got new target temprature to heat to of %v", target)
		switch getHVACModeFromString(modeThing.State) {
		case 1:
			lowTempThing.SetTempValue(target)
		case 3:
			lowTempThing.SetTempValue(target)
		}
	})
	ac.Thermostat.CoolingThresholdTemperature.OnValueRemoteUpdate(func(target float64) {
		if units == 1 {
			target = (target * 1.8) + 32
			target = math.Round(target)
		}
		log.Printf("Got new target temprature to cool to of %v", target)
		switch getHVACModeFromString(modeThing.State) {
		case 1:
			highTempThing.SetTempValue(target)
		case 3:
			highTempThing.SetTempValue(target)
		}
	})
	ac.Thermostat.TargetTemperature.OnValueRemoteUpdate(func(target float64) {
		offset := float64(3)
		if units == 1 {
			target = (target * 1.8) + 32
			target = math.Round(target)
			offset = 5
		}
		log.Printf("Got new target temprature for state %s to of %v", modeThing.State, target)
		switch getHVACModeFromString(modeThing.State) {
		case 1:
			lowTempThing.SetTempValue(target)
			highTempThing.SetTempValue(target + offset)
		case 2:
			lowTempThing.SetTempValue(target - offset)
			highTempThing.SetTempValue(target)
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
