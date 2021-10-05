package automation

import (
	"github.com/jgulick48/rv-homekit/internal/mqtt"
	"log"
	"sync"
	"time"

	"github.com/jgulick48/rv-homekit/internal/bmv"
	"github.com/jgulick48/rv-homekit/internal/models"
)

type Automation struct {
	parameters models.Automation
	dvccConfig models.DVCCConfiguration
	bmvClient  bmv.Client
	mqttClient *mqtt.Client
	switchFunc func(bool)
	stateFunc  func() bool
	state      State
	isEnabled  bool
	mutex      sync.Mutex
	starter    chan bool
	startTime  chan time.Time
	stopTime   chan time.Time
}

func NewGeneratorAutomationClient(parameters models.Automation, client bmv.Client, mqttClient *mqtt.Client, dvccConfig models.DVCCConfiguration, switchFunc func(bool), stateFunc func() bool) Automation {
	automationState := State{
		LastStarted:         0,
		LastStopped:         0,
		AutomationTriggered: false,
	}
	automationState.LoadFromFile("")
	return Automation{
		state:      automationState,
		parameters: parameters,
		dvccConfig: dvccConfig,
		bmvClient:  client,
		mqttClient: mqttClient,
		switchFunc: switchFunc,
		stateFunc:  stateFunc,
		mutex:      sync.Mutex{},
		starter:    make(chan bool),
		startTime:  make(chan time.Time),
		stopTime:   make(chan time.Time),
	}
}

func (a *Automation) AutomateGeneratorStart() {
	a.isEnabled = true
	ticker := time.NewTicker(time.Second * 10)
	go func() {
		for {
			select {
			case stopTime := <-a.stopTime:
				a.state.LastStopped = stopTime.Unix()
			case startTime := <-a.startTime:
				a.state.LastStarted = startTime.Unix()
			case automationStarted := <-a.starter:
				a.state.AutomationTriggered = automationStarted
			case <-ticker.C:
				a.mutex.Lock()
				state, ok := a.bmvClient.GetBatteryStateOfCharge()
				if !ok {
					a.mutex.Unlock()
					continue
				}
				if state < a.parameters.LowValue {
					if a.stateFunc() {
						if !a.state.AutomationTriggered {
							log.Printf("Generator already on, skipping start.")
						}
					} else {
						log.Printf("State of charge below threshold of %v, starting generator.", a.parameters.LowValue)
						if a.parameters.CoolDown.Duration > 0 {
							if time.Now().Before(time.Unix(a.state.LastStopped, 0).Add(a.parameters.CoolDown.Duration)) {
								log.Printf("Cooldown has not yet finished, waiting until at least %v to start generator.", time.Unix(a.state.LastStopped, 0).Add(a.parameters.CoolDown.Duration))
								a.mutex.Unlock()
								continue
							}
						}
						a.switchFunc(true)
						a.state.AutomationTriggered = true
						a.state.LastStarted = time.Now().Unix()
						a.state.SaveToFile("")
					}
				} else if a.state.AutomationTriggered && shouldShutOff(a.parameters, time.Unix(a.state.LastStarted, 0), a.bmvClient) {
					if a.parameters.OffDelay.Duration > 0 {
						log.Printf("Waiting %s before stopping generater.", a.parameters.OffDelay)
						time.Sleep(a.parameters.OffDelay.Duration)
					}
					a.state.LastStopped = time.Now().Unix()
					a.switchFunc(false)
					a.state.AutomationTriggered = false
					a.state.SaveToFile("")
					a.mutex.Unlock()
					continue
				}
				a.mutex.Unlock()
			}

		}
	}()
}

func shouldShutOff(params models.Automation, startTime time.Time, client bmv.Client) bool {
	if time.Now().Before(startTime.Add(params.MinOn.Duration)) {
		return false
	}
	if params.MaxOn.Duration != 0 && time.Now().After(startTime.Add(params.MaxOn.Duration)) {
		log.Printf("Generator has been running for %s which is longer than %s, signaling generator to shut off.", time.Now().Sub(startTime), params.MaxOn)
		return true
	}
	state, ok := client.GetBatteryStateOfCharge()
	if !ok {
		log.Print("Unable to get battery state of charge, signaling generator to shut off.")
		return true
	}
	if state > params.HighValue {
		log.Printf("Battery is now at %v which is higher than %v, signaling generator to shut off.", state, params.HighValue)
		return true
	}
	chargeCurrent, ok := client.GetBatteryCurrent()
	if !ok {
		log.Print("Unable to get battery current, signaling generator to shut off.")
		return true
	}
	if chargeCurrent < params.MinChargeCurrent {
		log.Printf("Battery current is now at %v which is lower than %v, signaling generator to shut off.", chargeCurrent, params.MinChargeCurrent)
		return true
	}

	return false
}

func (a *Automation) IsEnabled() bool {
	return a.isEnabled
}

func (a *Automation) StartAutoCharge() {
	a.mutex.Lock()
	if a.stateFunc() {
		if !a.state.AutomationTriggered {
			log.Printf("Generator already on, skipping start but setting triggered flag.")
		}
	} else {
		log.Printf("Generator not on, starting from manual automation trigger.")
		a.switchFunc(true)
		a.state.LastStarted = time.Now().Unix()
		a.startTime <- time.Now()
	}
	a.state.AutomationTriggered = true
	a.starter <- true
	a.state.SaveToFile("")
	a.mutex.Unlock()
}

func (a *Automation) StopAutoCharge() {
	a.mutex.Lock()
	if !a.stateFunc() {
		log.Printf("Generator already off, skipping stop")
	} else {
		if a.dvccConfig.LowChargeCurrentMax != 0 && a.mqttClient != nil {
			mqttClient := *a.mqttClient
			mqttClient.SetMaxChargeCurrent(a.dvccConfig.LowChargeCurrentMax)
			log.Printf("Got signal to turn off. Setting DVCC max charge current to %v and waiting 30 seconds", a.dvccConfig.LowChargeCurrentMax)
			go func() {
				time.Sleep(time.Second * 30)
				log.Printf("Generator on, stopping from manual automation cancel")
				a.switchFunc(false)
				a.state.LastStopped = time.Now().Unix()
				a.stopTime <- time.Now()
			}()
		} else {
			log.Printf("Generator on, stopping from manual automation cancel")
			a.switchFunc(false)
			a.state.LastStopped = time.Now().Unix()
			a.stopTime <- time.Now()
		}
	}
	a.state.AutomationTriggered = false
	a.starter <- false
	a.state.SaveToFile("")
	a.mutex.Unlock()
}

func (a *Automation) IsAutomationRunning() bool {
	return a.state.AutomationTriggered
}
