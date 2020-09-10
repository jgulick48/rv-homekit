package main

import (
	"log"

	"github.com/jgulick48/hc"
	"github.com/jgulick48/hc/accessory"

	"github.com/jgulick48/rv-homekit/internal/bmv"
	"github.com/jgulick48/rv-homekit/internal/openHab"
	"github.com/jgulick48/rv-homekit/internal/rvhomekit"
)

func main() {
	config := rvhomekit.LoadClientConfig()
	var bmvClient *bmv.Client
	if config.BMVConfig.Device != "" {
		config := bmv.ClientConfig{
			DeviceName: config.BMVConfig.Device,
			Baud:       config.BMVConfig.Baud,
		}
		client := bmv.NewClient(config)
		bmvClient = &client

	}
	habClient := openHab.NewClient(config.OpenHabServer)
	things, err := habClient.GetThings()
	if err != nil {
		panic(err)
	}
	rvHomeKit := rvhomekit.NewClient(config, habClient, bmvClient)
	accessories := rvHomeKit.GetAccessoriesFromOpenHab(things)
	bridge := accessory.NewBridge(accessory.Info{
		Name: config.BridgeName,
		ID:   1,
	})

	log.Printf("Found %v items", len(accessories))

	// configure the ip transport
	hcConfig := hc.Config{
		Pin: config.PIN,
	}
	if config.Port != "" {
		hcConfig.Port = config.Port
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
