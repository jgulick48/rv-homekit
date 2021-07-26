package rvhomekit

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	batteryAmpHours = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "batteryAmpHours",
			Help: "Number of blob storage operations waiting to be processed.",
		},
		[]string{
			"name",
		},
	)
	batteryStateOfCharge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "batteryStateOfCharge",
			Help: "Number of blob storage operations waiting to be processed.",
		},
		[]string{
			"name",
		},
	)
	batteryCurrent = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "batteryCurrent",
			Help: "Number of blob storage operations waiting to be processed.",
		},
		[]string{
			"name",
		},
	)
	batteryVolts = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "batteryVolts",
			Help: "Number of blob storage operations waiting to be processed.",
		},
		[]string{
			"name",
		},
	)
	batteryWatts = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "batteryWatts",
			Help: "Number of blob storage operations waiting to be processed.",
		},
		[]string{
			"name",
		},
	)

	batteryTemperature = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "batteryTemperature",
			Help: "Number of blob storage operations waiting to be processed.",
		},
		[]string{
			"name",
			"unit",
		},
	)
	batteryTimeRemaining = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "batteryTimeRemaining",
			Help: "Number of blob storage operations waiting to be processed.",
		},
		[]string{
			"name",
		},
	)
	batteryChargeTimeRemaining = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "batteryChargeTimeRemaining",
			Help: "Number of blob storage operations waiting to be processed.",
		},
		[]string{
			"name",
		},
	)
	tankLevel = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tankLevel",
			Help: "Number of blob storage operations waiting to be processed.",
		},
		[]string{
			"name",
			"type",
		},
	)
	tankLevelMM = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tankLevelMM",
			Help: "Number of blob storage operations waiting to be processed.",
		},
		[]string{
			"name",
		},
	)
	tankTempCelsius = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tankTempCelsius",
			Help: "Number of blob storage operations waiting to be processed.",
		},
		[]string{
			"name",
		},
	)
	tankTempFahrenheit = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tankTempFahrenheit",
			Help: "Number of blob storage operations waiting to be processed.",
		},
		[]string{
			"name",
		},
	)
	tankBatteryPercent = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tankBatteryPercent",
			Help: "Number of blob storage operations waiting to be processed.",
		},
		[]string{
			"name",
		},
	)
	tankBatteryVoltage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tankBatteryVoltage",
			Help: "Number of blob storage operations waiting to be processed.",
		},
		[]string{
			"name",
		},
	)
	tankSensorQuality = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tankSensorQuality",
			Help: "Number of blob storage operations waiting to be processed.",
		},
		[]string{
			"name",
		},
	)
	tankSensorRSSI = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tankSensorRSSI",
			Help: "Number of blob storage operations waiting to be processed.",
		},
		[]string{
			"name",
		},
	)
	batteryAutoChargeStarted = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "batteryAutoChargeStarted",
			Help: "Number of blob storage operations waiting to be processed.",
		},
		[]string{},
	)
	batteryAutoChargeState = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "batteryAutoChargeState",
			Help: "Number of blob storage operations waiting to be processed.",
		},
		[]string{},
	)

	generatorStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "generatorStatus",
			Help: "Number of blob storage operations waiting to be processed.",
		},
		[]string{
			"name",
		},
	)
	hvacTemperature = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "hvacTemperature",
			Help: "Number of blob storage operations waiting to be processed.",
		},
		[]string{
			"name",
		},
	)
	hvacCurrentStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "hvacCurrentStatus",
			Help: "Number of blob storage operations waiting to be processed.",
		},
		[]string{
			"name",
		},
	)
	hvacCurrentMode = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "hvacCurrentMode",
			Help: "Number of blob storage operations waiting to be processed.",
		},
		[]string{
			"name",
		},
	)
)
