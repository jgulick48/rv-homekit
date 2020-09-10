package automation

import (
	"log"
	"time"
)

func AutomateGeneratorStart(highValue float64, lowValue float64, socFunc func() (float64, bool), switchFunc func(bool), stateFunc func() bool) {
	go func() {
		automationTriggered := false
		for {
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
					switchFunc(true)
					automationTriggered = true
				}
			} else if automationTriggered && state > highValue {
				log.Printf("State of charge above threshold of %v, stopping generator.", highValue)
				switchFunc(false)
				automationTriggered = false
				continue
			}
		}
	}()
}
