package automation

import (
	"log"
	"time"

	"github.com/jgulick48/rv-homekit/internal/bmv"
	"github.com/jgulick48/rv-homekit/internal/models"
)

type Automation struct {
	parameters models.Automation
	bmvClient  bmv.Client
	switchFunc func(bool)
	stateFunc  func() bool
	state      AutomationState
	isEnabled  bool
}

func NewGeneratorAutomationClient(parameters models.Automation, client bmv.Client, switchFunc func(bool), stateFunc func() bool) Automation {
	automationState := AutomationState{
		LastStarted:         0,
		LastStopped:         0,
		AutomationTriggered: false,
	}
	automationState.LoadFromFile("")
	return Automation{
		state:      automationState,
		parameters: parameters,
		bmvClient:  client,
		switchFunc: switchFunc,
		stateFunc:  stateFunc,
	}
}

func (a *Automation) AutomateGeneratorStart() {
	a.isEnabled = true
	go func() {
		for {
			time.Sleep(time.Second * 10)
			state, ok := a.bmvClient.GetBatteryStateOfCharge()
			if !ok {
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
				continue
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
	if a.stateFunc() {
		if !a.state.AutomationTriggered {
			log.Printf("Generator already on, skipping start but setting triggered flag.")
		}
	} else {
		log.Printf("Generator not on, starting from manual automation trigger.")
		a.switchFunc(true)
		a.state.LastStarted = time.Now().Unix()
	}
	a.state.AutomationTriggered = true
	a.state.SaveToFile("")
}

func (a *Automation) StopAutoCharge() {
	if !a.stateFunc() {
		log.Printf("Generator already off, skipping stop")
	} else {
		log.Printf("Generator on, stopping from manual automation cancel")
		a.switchFunc(false)
		a.state.LastStopped = time.Now().Unix()
	}
	a.state.AutomationTriggered = false
	a.state.SaveToFile("")
}

func (a *Automation) IsAutomationRunning() bool {
	return a.state.AutomationTriggered
}
