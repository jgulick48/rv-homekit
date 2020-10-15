package automation

import (
	"log"
	"time"

	"github.com/jgulick48/rv-homekit/internal/bmv"
	"github.com/jgulick48/rv-homekit/internal/models"
)

func AutomateGeneratorStart(paramaters models.Automation, client bmv.Client, switchFunc func(bool), stateFunc func() bool) {
	go func() {
		automationTriggered := false
		var lastStarted time.Time
		var lastStopped time.Time
		for {
			time.Sleep(time.Second * 10)
			state, ok := client.GetBatteryStateOfCharge()
			if !ok {
				continue
			}
			if state < paramaters.LowValue {
				if stateFunc() {
					if !automationTriggered {
						log.Printf("Generator already on, skipping start.")
					}
				} else {
					log.Printf("State of charge below threshold of %v, starting generator.", paramaters.LowValue)
					if paramaters.CoolDown.Duration > 0 {
						if time.Now().Before(lastStopped.Add(paramaters.CoolDown.Duration)) {
							log.Printf("Cooldown has not yet finished, waiting until at least %v to start generator.", lastStopped.Add(paramaters.CoolDown.Duration))
							continue
						}
					}
					switchFunc(true)
					automationTriggered = true
					lastStarted = time.Now()
				}
			} else if automationTriggered && shouldShutOff(paramaters, lastStarted, client) {
				if paramaters.OffDelay.Duration > 0 {
					log.Printf("Waiting %s before stopping generater.", paramaters.OffDelay)
					time.Sleep(paramaters.OffDelay.Duration)
				}
				lastStopped = time.Now()
				switchFunc(false)
				automationTriggered = false
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
