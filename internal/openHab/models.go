package openHab

const (
	STATUS_UNINITIALIZED = "UNINITIALIZED"
	STATUS_INITIALIZING  = "INITIALIZING"
	STATUS_UNKNOWN       = "UNKNOWN"
	STATUS_ONLINE        = "ONLINE"
	STATUS_OFFLINE       = "OFFLINE"
	STATUS_REMOVING      = "REMOVING"
	STATUS_REMOVED       = "REMOVED"

	STATUS_DETAIL_NONE                          = "NONE"
	STATUS_DETAIL_HANDLER_MISSING_ERROR         = "HANDLER_MISSING_ERROR"
	STATUS_DETAIL_HANDLER_REGISTERING_ERROR     = "HANDLER_REGISTERING_ERROR"
	STATUS_DETAIL_HANDLER_INITIALIZING_ERROR    = "HANDLER_INITIALIZING_ERROR"
	STATUS_DETAIL_HANDLER_CONFIGURATION_PENDING = "HANDLER_CONFIGURATION_PENDING"
	STATUS_DETAIL_CONFIGURATION_PENDING         = "CONFIGURATION_PENDING"
	STATUS_DETAIL_COMMUNICATION_ERROR           = "COMMUNICATION_ERROR"
	STATUS_DETAIL_CONFIGURATION_ERROR           = "CONFIGURATION_ERROR"
	STATUS_DETAIL_BRIDGE_OFFLINE                = "BRIDGE_OFFLINE"
	STATUS_DETAIL_FIRMWARE_UPDATING             = "FIRMWARE_UPDATING"
	STATUS_DETAIL_DUTY_CYCLE                    = "DUTY_CYCLE"
	STATUS_DETAIL_BRIDGE_UNINITIALIZED          = "BRIDGE_UNINITIALIZED"
)

type EnrichedThingDTO struct {
	Label          string                 `json:"label"`
	BridgeUID      string                 `json:"bridgeUID"`
	Configuration  map[string]interface{} `json:"configuration"`
	Properties     map[string]interface{} `json:"properties"`
	UID            string                 `json:"UID"`
	ThingTypeUID   string                 `json:"thingTypeUID"`
	Channels       []ChannelDTO           `json:"channels"`
	Location       string                 `json:"location"`
	StatusInfo     ThingStatusInfo        `json:"statusInfo"`
	FirmwareStatus FirmwareStatusDTO      `json:"firmwareStatus"`
	Editable       bool                   `json:"editable"`
}

type ChannelDTO struct {
	UID            string                 `json:"uid"`
	ID             string                 `json:"id"`
	ChannelTypeUID string                 `json:"channelTypeUID"`
	ItemType       string                 `json:"itemType"`
	Kind           string                 `json:"kind"`
	Label          string                 `json:"label"`
	Description    string                 `json:"description"`
	DefaultTags    []string               `json:"defaultTags"`
	Properties     map[string]interface{} `json:"properties"`
	Configuration  map[string]interface{} `json:"configuration"`
}

type ThingStatusInfo struct {
	Status       string `json:"status"`
	StatusDetail string `json:"statusDetail"`
	Description  string `json:"description"`
}

type FirmwareStatusDTO struct {
	Status           string `json:"status"`
	UpdatableVersion string `json:"updatableVersion"`
}

type EnrichedItemDTO struct {
	Type             string   `json:"type"`
	Name             string   `json:"name"`
	Label            string   `json:"label"`
	Category         string   `json:"category"`
	Tags             []string `json:"tags"`
	GroupNames       []string `json:"group_names"`
	Link             string   `json:"link"`
	State            string   `json:"state"`
	TransformedState string   `json:"transformedState"`
	StateDescription `json:"stateDescription"`
}

type StateDescription struct {
	Minimum  int64         `json:"minimum"`
	Maximum  int64         `json:"maximum"`
	Step     int64         `json:"step"`
	Pattern  string        `json:"pattern"`
	ReadOnly bool          `json:"readOnly"`
	Options  []StateOption `json:"options"`
}

type StateOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}
