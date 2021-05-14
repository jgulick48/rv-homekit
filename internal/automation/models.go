package automation

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

type AutomationState struct {
	LastStarted         int64 `json:"lastStarted"`
	LastStopped         int64 `json:"lastStopped"`
	AutomationTriggered bool  `json:"automationTriggered"`
}

func (a *AutomationState) LoadFromFile(filename string) {
	if filename == "" {
		filename = "./state.json"
	}
	configFile, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Printf("No config file found. Making new IDs")
		panic(err)
	}
	err = json.Unmarshal(configFile, &a)
	if err != nil {
		log.Printf("Invliad config file provided")
		panic(err)
	}
}

func (a *AutomationState) SaveToFile(filename string) {
	if filename == "" {
		filename = "./state.json"
	}
	data, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return
	}
	ioutil.WriteFile(filename, data, 0644)
}
