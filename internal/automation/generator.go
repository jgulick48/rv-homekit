package automation

import (
	"log"
	"time"
)

func AutomateGeneratorStart(highValue float64, lowValue float64, delay time.Duration, coolDown time.Duration, socFunc func() (float64, bool), switchFunc func(bool), stateFunc func() bool) {
	go func() {
		automationTriggered := false
		for {
			var lastStopped time.Time
			time.Sleep(time.Second * 10)
			state, ok := socFunc()
			if !ok {
				continue
			}
			if state < lowValue {
				if stateFunc() {
					if !automationTriggered {
						log.Printf("Generator already on, skipping start.")
					}
				} else {
					log.Printf("State of charge below threshold of %v, starting generator.", lowValue)
					if coolDown > 0 {
						if time.Now().Before(lastStopped.Add(coolDown)) {
							log.Printf("Cooldown has not yet finished, waiting until at least %v to start generator.", lastStopped.Add(coolDown))
							continue
						}
					}
					switchFunc(true)
					automationTriggered = true
				}
			} else if automationTriggered && state > highValue {
				log.Printf("State of charge above threshold of %v", highValue)
				if delay > 0 {
					log.Printf("Waiting %v before stopping generater.")
					time.Sleep(delay)
				}
				lastStopped = time.Now()
				switchFunc(false)
				automationTriggered = false
				continue
			}
		}
	}()
}
