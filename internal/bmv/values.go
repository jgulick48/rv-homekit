package bmv

type Measurement struct {
	Name    string
	Units   string
	Channel string
}

var Parameters = map[string]Measurement{
	"V": {
		Name:    "Battery Voltage",
		Units:   "mV",
		Channel: "1",
	},
	"V2": {
		Name:    "Battery Voltage",
		Units:   "mV",
		Channel: "2",
	},
	"V3": {
		Name:    "Battery Voltage",
		Units:   "mV",
		Channel: "3",
	},
	"VS": {
		Name:    "Battery Voltage",
		Units:   "mV",
		Channel: "Aux",
	},
	"VM": {
		Name:    "Battery Voltage (Mid-point)",
		Units:   "mV",
		Channel: "",
	},
	"DM": {
		Name:    "Battery Deviation (Mid-point)",
		Units:   "mV",
		Channel: "Aux",
	},
	"VPV": {
		Name:    "Panel Voltage",
		Units:   "mV",
		Channel: "",
	},
	"PPV": {
		Name:    "Panel Power",
		Units:   "mV",
		Channel: "",
	},
	"I": {
		Name:    "Battery Current",
		Units:   "mA",
		Channel: "1",
	},
	"I2": {
		Name:    "Battery Current",
		Units:   "mA",
		Channel: "2",
	},
	"I3": {
		Name:    "Battery Current",
		Units:   "mA",
		Channel: "3",
	},
	"IL": {
		Name:    "Load Current",
		Units:   "mA",
		Channel: "",
	},
	"LOAD": {
		Name:    "Load Output State",
		Units:   "ON/OFF",
		Channel: "",
	},
	"T": {
		Name:    "Battery Temperature",
		Units:   "°C",
		Channel: "",
	},
	"P": {
		Name:    "Instantaneous Power",
		Units:   "W",
		Channel: "",
	},
	"CE": {
		Name:    "Consumed Amp Hours",
		Units:   "mAh",
		Channel: "",
	},
	"SOC": {
		Name:    "State-of-charge",
		Units:   "% (0-100)",
		Channel: "",
	},
	"TTG": {
		Name:    "Time To Go",
		Units:   "Minutes",
		Channel: "",
	},
	"Alarm": {
		Name:    "Alarm Condition Active",
		Units:   "ON/OFF",
		Channel: "",
	},
	"Relay": {
		Name:    "Relay State",
		Units:   "ON/OFF",
		Channel: "",
	},
	"AR": {
		Name:    "Alarm Reason",
		Units:   "",
		Channel: "",
	},
	"OR": {
		Name:    "Off Reason",
		Units:   "",
		Channel: "",
	},
	"H1": {
		Name:    "Depth of the deepest discharge",
		Units:   "mAh",
		Channel: "",
	},
	"H2": {
		Name:    "Depth of the last discharge",
		Units:   "mAh",
		Channel: "",
	},
	"H3": {
		Name:    "Depth of the average discharge",
		Units:   "mAh",
		Channel: "",
	},
	"H4": {
		Name:    "Number of charge cycles",
		Units:   "",
		Channel: "",
	},
	"H5": {
		Name:    "Number of full discharges",
		Units:   "",
		Channel: "",
	},
	"H6": {
		Name:    "Cumulative Amp Hours drawn",
		Units:   "mAh",
		Channel: "",
	},
	"H7": {
		Name:    "Minimum main (battery) voltage",
		Units:   "mV",
		Channel: "",
	},
	"H8": {
		Name:    "Maximum main (battery) voltage",
		Units:   "mV",
		Channel: "",
	},
	"H9": {
		Name:    "Number of seconds since last full charge",
		Units:   "s",
		Channel: "",
	},
	"H10": {
		Name:    "Number of automatic synchronizations",
		Units:   "",
		Channel: "",
	},
	"H11": {
		Name:    "Number of low main voltage alarms",
		Units:   "",
		Channel: "",
	},
	"H12": {
		Name:    "Number of high main voltage alarms",
		Units:   "",
		Channel: "",
	},
	"H13": {
		Name:    "Number of low auxiliary voltage alarms",
		Units:   "",
		Channel: "",
	},
	"H14": {
		Name:    "Number of high auxiliary voltage alarms",
		Units:   "",
		Channel: "",
	},
	"H15": {
		Name:    "Minimum auxiliary (battery) voltage",
		Units:   "mV",
		Channel: "",
	},
	"H16": {
		Name:    "Maximum auxiliary (battery) voltage",
		Units:   "mV",
		Channel: "",
	},
	"H17": {
		Name:    "Amount of discharged energy",
		Units:   "0.01 kWh",
		Channel: "",
	},
	"H18": {
		Name:    "Amount of charged energy",
		Units:   "0.01 kWh",
		Channel: "",
	},
	"H19": {
		Name:    "Yield total (user resettable counter)",
		Units:   "0.01 kWh",
		Channel: "",
	},
	"H20": {
		Name:    "Yield today",
		Units:   "0.01 kWh",
		Channel: "",
	},
	"H21": {
		Name:    "Maximum power today",
		Units:   "W",
		Channel: "",
	},
	"H22": {
		Name:    "Yield yesterday",
		Units:   "0.01 kWh",
		Channel: "",
	},
	"H23": {
		Name:    "Maximum power yesterday",
		Units:   "W",
		Channel: "",
	},
	"ERR": {
		Name:    "Error code",
		Units:   "",
		Channel: "",
	},
	"CS": {
		Name:    "State of Operation",
		Units:   "",
		Channel: "",
	},
	"BMV": {
		Name:    "Model description (deprecated)",
		Units:   "",
		Channel: "",
	},
	"FW": {
		Name:    "Firmware version (16 bit)",
		Units:   "",
		Channel: "",
	},
	"FWE": {
		Name:    "Firmware version (24 bit)",
		Units:   "",
		Channel: "",
	},
	"PID": {
		Name:    "Product ID",
		Units:   "",
		Channel: "",
	},
	"SER#": {
		Name:    "Serial number",
		Units:   "",
		Channel: "",
	},
	"HSDS": {
		Name:    "Day sequence number (0..364)",
		Units:   "",
		Channel: "",
	},
	"MODE": {
		Name:    "Device mode",
		Units:   "",
		Channel: "",
	},
	"AC_OUT_V": {
		Name:    "AC output voltage",
		Units:   "0.01 V",
		Channel: "",
	},
	"AC_OUT_I": {
		Name:    "AC output current",
		Units:   "0.1 A",
		Channel: "",
	},
	"AC_OUT_S": {
		Name:    "AC output apparent power",
		Units:   "VA",
		Channel: "",
	},
	"WARN": {
		Name:    "Warning reason",
		Units:   "",
		Channel: "",
	},
	"MPPT": {
		Name:    "Tracker operation mode",
		Units:   "",
		Channel: "",
	},
}
