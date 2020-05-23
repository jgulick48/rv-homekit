package openHab

import (
	"strings"
)

func (c *ChannelDTO) ConvertUIDToTingUID() string {
	uid := strings.Replace(c.UID, ":", "_", -1)
	uid = strings.Replace(uid, "-", "_", -1)
	return uid
}

func (c *ChannelDTO) IsSwitch() bool {
	return c.ChannelTypeUID == "idsmyrv:switch"
}

func (c *ChannelDTO) IsLight() bool {
	return c.ChannelTypeUID == "idsmyrv:switched-light"
}

func (c *ChannelDTO) IsDimmer() bool {
	return c.ChannelTypeUID == "idsmyrv:dimmer"
}

func (c *ChannelDTO) IsRGB() bool {
	return c.ChannelTypeUID == "idsmyrv:hsvcolor"
}
