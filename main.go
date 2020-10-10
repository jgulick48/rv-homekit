package main

import (
	"flag"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/jgulick48/hc"
	"github.com/jgulick48/hc/accessory"

	"github.com/jgulick48/rv-homekit/internal/bmv"
	"github.com/jgulick48/rv-homekit/internal/metrics"
	"github.com/jgulick48/rv-homekit/internal/openHab"
	"github.com/jgulick48/rv-homekit/internal/rvhomekit"
)

func main() {
	path, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}
	log.Print(path)
	configLocation := flag.String("configFile", "./config.json", "Location for the configuration file.")
	flag.Parse()
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

	hc.OnTermination(func() {
		<-t.Stop()
	})
	t.Start()
}
