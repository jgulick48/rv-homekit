package openevse

type CommandResult struct {
	CMD string `json:"cmd,omitempty"`
	RET string `json:"ret,omitempty"`
}

type StatusResult struct {
	Amp      int64  `json:"amp,omitempty"`
	Watthour int64  `json:"watthour,omitempty"`
	Wattsec  int64  `json:"wattsec,omitempty"`
	Status   string `json:"status,omitempty"`
}

type OverrideResult struct {
	Msg string `json:"msg,omitempty"`
}

type Config struct {
	MqttSupportedProtocols       []string `json:"mqtt_supported_protocols,omitempty,omitempty"`
	HttpSupportedProtocols       []string `json:"http_supported_protocols,omitempty"`
	Buildenv                     string   `json:"buildenv,omitempty"`
	Version                      string   `json:"version,omitempty"`
	WifiSerial                   string   `json:"wifi_serial,omitempty"`
	Protocol                     string   `json:"protocol,omitempty"`
	Espinfo                      string   `json:"espinfo,omitempty"`
	Espflash                     int      `json:"espflash,omitempty"`
	Firmware                     string   `json:"firmware,omitempty"`
	EvseSerial                   string   `json:"evse_serial,omitempty"`
	DiodeCheck                   bool     `json:"diode_check,omitempty"`
	GfciCheck                    bool     `json:"gfci_check,omitempty"`
	GroundCheck                  bool     `json:"ground_check,omitempty"`
	RelayCheck                   bool     `json:"relay_check,omitempty"`
	VentCheck                    bool     `json:"vent_check,omitempty"`
	TempCheck                    bool     `json:"temp_check,omitempty"`
	MaxCurrentSoft               int      `json:"max_current_soft,omitempty"`
	Service                      int      `json:"service,omitempty"`
	Scale                        int      `json:"scale,omitempty"`
	Offset                       int      `json:"offset,omitempty"`
	MinCurrentHard               int      `json:"min_current_hard,omitempty"`
	MaxCurrentHard               int      `json:"max_current_hard,omitempty"`
	Ssid                         string   `json:"ssid,omitempty"`
	Pass                         string   `json:"pass,omitempty"`
	ApSsid                       string   `json:"ap_ssid,omitempty"`
	ApPass                       string   `json:"ap_pass,omitempty"`
	Lang                         string   `json:"lang,omitempty"`
	WwwUsername                  string   `json:"www_username,omitempty"`
	WwwPassword                  string   `json:"www_password,omitempty"`
	WwwCertificateId             string   `json:"www_certificate_id,omitempty"`
	Hostname                     string   `json:"hostname,omitempty"`
	SntpHostname                 string   `json:"sntp_hostname,omitempty"`
	TimeZone                     string   `json:"time_zone,omitempty"`
	LimitDefaultType             string   `json:"limit_default_type,omitempty"`
	LimitDefaultValue            int      `json:"limit_default_value,omitempty"`
	EmoncmsServer                string   `json:"emoncms_server,omitempty"`
	EmoncmsNode                  string   `json:"emoncms_node,omitempty"`
	EmoncmsApikey                string   `json:"emoncms_apikey,omitempty"`
	EmoncmsFingerprint           string   `json:"emoncms_fingerprint,omitempty"`
	MqttServer                   string   `json:"mqtt_server,omitempty"`
	MqttPort                     int      `json:"mqtt_port,omitempty"`
	MqttTopic                    string   `json:"mqtt_topic,omitempty"`
	MqttUser                     string   `json:"mqtt_user,omitempty"`
	MqttPass                     string   `json:"mqtt_pass,omitempty"`
	MqttCertificateId            string   `json:"mqtt_certificate_id,omitempty"`
	MqttSolar                    string   `json:"mqtt_solar,omitempty"`
	MqttGridIe                   string   `json:"mqtt_grid_ie,omitempty"`
	MqttVrms                     string   `json:"mqtt_vrms,omitempty"`
	MqttLivePwr                  string   `json:"mqtt_live_pwr,omitempty"`
	MqttVehicleSoc               string   `json:"mqtt_vehicle_soc,omitempty"`
	MqttVehicleRange             string   `json:"mqtt_vehicle_range,omitempty"`
	MqttVehicleEta               string   `json:"mqtt_vehicle_eta,omitempty"`
	MqttAnnounceTopic            string   `json:"mqtt_announce_topic,omitempty"`
	OcppServer                   string   `json:"ocpp_server,omitempty"`
	OcppChargeBoxId              string   `json:"ocpp_chargeBoxId,omitempty"`
	OcppAuthkey                  string   `json:"ocpp_authkey,omitempty"`
	OcppIdtag                    string   `json:"ocpp_idtag,omitempty"`
	Ohm                          string   `json:"ohm,omitempty"`
	DivertType                   int      `json:"divert_type,omitempty"`
	DivertPVRatio                float64  `json:"divert_PV_ratio,omitempty"`
	DivertAttackSmoothingTime    int      `json:"divert_attack_smoothing_time,omitempty"`
	DivertDecaySmoothingTime     int      `json:"divert_decay_smoothing_time,omitempty"`
	DivertMinChargeTime          int      `json:"divert_min_charge_time,omitempty"`
	CurrentShaperMaxPwr          int      `json:"current_shaper_max_pwr,omitempty"`
	CurrentShaperSmoothingTime   int      `json:"current_shaper_smoothing_time,omitempty"`
	CurrentShaperMinPauseTime    int      `json:"current_shaper_min_pause_time,omitempty"`
	CurrentShaperDataMaxinterval int      `json:"current_shaper_data_maxinterval,omitempty"`
	VehicleDataSrc               int      `json:"vehicle_data_src,omitempty"`
	TeslaAccessToken             string   `json:"tesla_access_token,omitempty"`
	TeslaRefreshToken            string   `json:"tesla_refresh_token,omitempty"`
	TeslaCreatedAt               float64  `json:"tesla_created_at,omitempty"`
	TeslaExpiresIn               float64  `json:"tesla_expires_in,omitempty"`
	TeslaVehicleId               string   `json:"tesla_vehicle_id,omitempty"`
	RfidStorage                  string   `json:"rfid_storage,omitempty"`
	LedBrightness                int      `json:"led_brightness,omitempty"`
	SchedulerStartWindow         int      `json:"scheduler_start_window,omitempty"`
	Flags                        int      `json:"flags,omitempty"`
	FlagsChanged                 int      `json:"flags_changed,omitempty"`
	EmoncmsEnabled               bool     `json:"emoncms_enabled,omitempty"`
	MqttEnabled                  bool     `json:"mqtt_enabled,omitempty"`
	MqttRejectUnauthorized       bool     `json:"mqtt_reject_unauthorized,omitempty"`
	MqttRetained                 bool     `json:"mqtt_retained,omitempty"`
	OhmEnabled                   bool     `json:"ohm_enabled,omitempty"`
	SntpEnabled                  bool     `json:"sntp_enabled,omitempty"`
	TeslaEnabled                 bool     `json:"tesla_enabled,omitempty"`
	DivertEnabled                bool     `json:"divert_enabled,omitempty"`
	CurrentShaperEnabled         bool     `json:"current_shaper_enabled,omitempty"`
	PauseUsesDisabled            bool     `json:"pause_uses_disabled,omitempty"`
	MqttVehicleRangeMiles        bool     `json:"mqtt_vehicle_range_miles,omitempty"`
	OcppEnabled                  bool     `json:"ocpp_enabled,omitempty"`
	OcppAuthAuto                 bool     `json:"ocpp_auth_auto,omitempty"`
	OcppAuthOffline              bool     `json:"ocpp_auth_offline,omitempty"`
	OcppSuspendEvse              bool     `json:"ocpp_suspend_evse,omitempty"`
	OcppEnergizePlug             bool     `json:"ocpp_energize_plug,omitempty"`
	RfidEnabled                  bool     `json:"rfid_enabled,omitempty"`
	FactoryWriteLock             bool     `json:"factory_write_lock,omitempty"`
	IsThreephase                 bool     `json:"is_threephase,omitempty"`
	WizardPassed                 bool     `json:"wizard_passed,omitempty"`
	DefaultState                 bool     `json:"default_state,omitempty"`
	MqttProtocol                 string   `json:"mqtt_protocol,omitempty"`
	ChargeMode                   string   `json:"charge_mode,omitempty"`
}
type ConfigUpdateResponse struct {
	ConfigVersion int    `json:"config_version"`
	Msg           string `json:"msg"`
}
