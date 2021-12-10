package main

import (
	"flag"
	"github.com/jgulick48/rv-homekit/internal/openevse"
	"github.com/jgulick48/rv-homekit/internal/tanksensors"
	"log"
	"net/http"
	"net/http/httputil"
	_ "net/http/pprof"
	"net/url"
	"os"
	"time"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/jgulick48/hc"
	"github.com/jgulick48/hc/accessory"
	"github.com/mitchellh/panicwrap"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/jgulick48/rv-homekit/internal/bmv"
	"github.com/jgulick48/rv-homekit/internal/metrics"
	"github.com/jgulick48/rv-homekit/internal/mqtt"
	"github.com/jgulick48/rv-homekit/internal/openHab"
	"github.com/jgulick48/rv-homekit/internal/rvhomekit"
)

var configLocation = flag.String("configFile", "./config.json", "Location for the configuration file.")

func main() {

	startService()
	exitStatus, err := panicwrap.BasicWrap(panicHandler)
	if err != nil {
		// Something went wrong setting up the panic wrapper. Unlikely,
		// but possible.
		panic(err)
	}

	// If exitStatus >= 0, then we're the parent process and the panicwrap
	// re-executed ourselves and completed. Just exit with the proper status.
	if exitStatus >= 0 {
		os.Exit(exitStatus)
	}
}

func panicHandler(output string) {
	// output contains the full output (including stack traces) of the
	// panic. Put it in a file or something.
	log.Printf("The child panicked:\n\n%s\n", output)
	os.Exit(1)
}

func startService() {
	path, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}
	log.Print(path)
	config := rvhomekit.LoadClientConfig(*configLocation)
	if config.StatsServer != "" {
		metrics.Metrics, err = statsd.New(config.StatsServer)
		if err != nil {
			log.Printf("Error creating stats client %s", err.Error())
		} else {
			metrics.StatsEnabled = true
		}
	}
	var bmvClient *bmv.Client
	if config.BMVConfig.Device != "" {
		client := bmv.NewClient(config.BMVConfig)
		bmvClient = &client

	}
	habClient := openHab.NewClient(config.OpenHabServer)
	things, err := habClient.GetThings()
	if err != nil {
		panic(err)
	}
	var tankSensors tanksensors.Client
	if config.TankSensors.Enabled {
		tankSensors = tanksensors.NewTankSensorClient(config.TankSensors.APIAddress)
	}
	mqttClient := mqtt.NewClient(config.MQTTConfiguration, config.DVCCConfiguration, config.Debug)
	openEVSEClient := openevse.NewClient(mqttClient.GetVEBusClient(), config.EVSEConfiguration, http.DefaultClient)
	rvHomeKitClient := rvhomekit.NewClient(config, habClient, bmvClient, tankSensors, mqttClient)
	accessories := rvHomeKitClient.GetAccessoriesFromOpenHab(things)
	rvHomeKitClient.SaveClientConfig(*configLocation)
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
	syncTimer := time.Second * 10
	if duration, err := time.ParseDuration(config.SyncTimer); err == nil {
		syncTimer = duration
	}
	ticker := time.NewTicker(syncTimer)
	defer ticker.Stop()
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				rvHomeKitClient.RunSyncFunctions()
			}
		}
	}()
	go func() {
		origin, _ := url.Parse(config.OpenHabServer)

		director := func(req *http.Request) {
			req.Header.Add("X-Forwarded-Host", req.Host)
			req.Header.Add("X-Origin-Host", origin.Host)
			req.URL.Scheme = "http"
			req.URL.Host = origin.Host
		}

		proxy := &httputil.ReverseProxy{Director: director}

		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			proxy.ServeHTTP(w, r)
		})

		log.Fatal(http.ListenAndServe(":8080", nil))
	}()
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":2112", nil)
	}()
	hc.OnTermination(func() {
		<-t.Stop()
		ticker.Stop()
		openEVSEClient.Stop()
		done <- true
	})
	t.Start()
}
