package openHab

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
)

func (i *EnrichedItemDTO) GetChangeFunction() func(bool) {
	switch i.Type {
	case "Switch":
		return i.ChangeSwitch
	case "Dimmer":
		return i.SwitchDimmer
	default:
		return i.ChangeDefault
	}
}

func (i *EnrichedItemDTO) ChangeDefault(on bool) {
	if on == true {
		log.Println("Switch is on")
	} else {
		log.Println("Switch is off")
	}
}

func (i *EnrichedItemDTO) ChangeSwitch(on bool) {
	if on == true {
		changeItemValue(strings.Replace(i.Link, "hsvcolor", "switch", 1), "ON")
	} else {
		changeItemValue(strings.Replace(i.Link, "hsvcolor", "switch", 1), "OFF")
	}
}
func (i *EnrichedItemDTO) SwitchDimmer(on bool) {
	if on == true {
		changeItemValue(i.Link, "100")
	} else {
		changeItemValue(i.Link, "0")
	}
}

func (i *EnrichedItemDTO) PreferGas(on bool) {
	if on {
		changeItemValue(i.Link, "GAS")
	}
}

func (i *EnrichedItemDTO) PreferHeatPump(on bool) {
	if on {
		changeItemValue(i.Link, "HEATPUMP")
	}
}

func (i *EnrichedItemDTO) ChangeDimmer(brightness int) {
	changeItemValue(i.Link, strconv.Itoa(brightness))
}

func (i *EnrichedItemDTO) ChangeHueValue(hue float64) {
	if hue < 0 {
		hue = 0
	}
	if hue > 255 {
		hue = 255
	}
	i.GetCurrentValue()
	hsv := strings.Split(i.State, ",")
	if len(hsv) != 3 {
		log.Printf("Invalid state received for HSV. Update failed.")
		return
	}
	hsv[0] = strconv.Itoa(int(hue))
	changeItemValue(i.Link, strings.Join(hsv, ","))
}

func (i *EnrichedItemDTO) ChangeSaturationValue(sat float64) {
	if sat < 0 {
		sat = 0
	}
	if sat > 100 {
		sat = 100
	}
	i.GetCurrentValue()
	hsv := strings.Split(i.State, ",")
	if len(hsv) != 3 {
		log.Printf("Invalid state received for HSV. Update failed.")
		return
	}
	hsv[1] = strconv.Itoa(int(sat))
	changeItemValue(i.Link, strings.Join(hsv, ","))
}

func (i *EnrichedItemDTO) ChangeBrightnessValue(brightness int) {
	if brightness < 0 {
		brightness = 0
	}
	if brightness > 100 {
		brightness = 100
	}
	i.GetCurrentValue()
	hsv := strings.Split(i.State, ",")
	if len(hsv) != 3 {
		log.Printf("Invalid state received for HSV. Update failed.")
		return
	}
	hsv[2] = strconv.Itoa(brightness)
	changeItemValue(i.Link, strings.Join(hsv, ","))
}

func (i *EnrichedItemDTO) SetHVACToMode(mode int) {
	switch mode {
	case 0:
		changeItemValue(i.Link, "OFF")
		break
	case 1:
		changeItemValue(i.Link, "HEAT")
		break
	case 2:
		changeItemValue(i.Link, "COOL")
		break
	case 3:
		changeItemValue(i.Link, "HEATCOOL")
	default:
		log.Printf("Invalid mode passed to HVAC. Got %v was expecting 0, 1, 2, or 3", mode)
	}
}

func (i *EnrichedItemDTO) SetTempValue(temp float64) {
	changeItemValue(i.Link, strconv.Itoa(int(temp)))
}

func changeItemValue(link string, value string) {
	httpClient := http.DefaultClient
	body := bytes.NewBuffer([]byte(value))
	req, err := http.NewRequest(http.MethodPost, link, body)
	if err != nil {
		log.Printf("Error creating request for things from OpenHAB: %s", err)
		return
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("Error making request for things from OpenHAB: %s", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("Invalid response from OpenHAB. Got %v expecting 202", resp.StatusCode)
		return
	}
	log.Printf("Successfully changed state for %s to %s", link, value)
}

func (i *EnrichedItemDTO) GetState() (string, error) {
	i.GetCurrentState()
	return i.State, nil
}

func (i *EnrichedItemDTO) SetState(state string) {
	i.SetItemState(state)
}

func (i *EnrichedItemDTO) InHPState() bool {
	i.GetCurrentValue()
	if i.State == "HEAT" {
		if source, ok := i.getCurrentSource(); ok && source == "GAS" {
			return false
		}
	}
	return true
}

func (i *EnrichedItemDTO) GetCurrentValue() {
	httpClient := http.DefaultClient
	req, err := http.NewRequest(http.MethodGet, i.Link, nil)
	if err != nil {
		log.Printf("Error creating request for things from OpenHAB: %s", err)
		return
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("Error making request for things from OpenHAB: %s", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("Invalid response from OpenHAB. Got %v expecting 202", resp.StatusCode)
		return
	}
	err = json.NewDecoder(resp.Body).Decode(i)
	if err != nil {
		log.Printf("Error getting latest values: %s", err)
		return
	}
}
func (i *EnrichedItemDTO) getCurrentSource() (string, bool) {
	if i.Label != "HVAC Mode" {
		return "", false
	}
	httpClient := http.DefaultClient
	link := strings.Replace(i.Link, "hvac_mode", "heat_source", 1)
	req, err := http.NewRequest(http.MethodGet, link, nil)
	if err != nil {
		log.Printf("Error creating request for things from OpenHAB: %s", err)
		return "", false
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("Error making request for things from OpenHAB: %s", err)
		return "", false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("Invalid response from OpenHAB. Got %v expecting 202", resp.StatusCode)
		return "", false
	}
	var heatSource EnrichedItemDTO
	err = json.NewDecoder(resp.Body).Decode(&heatSource)
	if err != nil {
		log.Printf("Error getting latest values: %s", err)
		return "", false
	}
	return heatSource.State, true
}

func (i *EnrichedItemDTO) SetItemState(value string) {
	changeItemValue(i.Link, value)
}

func (i *EnrichedItemDTO) GetCurrentState() bool {
	i.GetCurrentValue()
	switch i.State {
	case "RUNNING", "PRIMING", "ON":
		return true
	default:
		return false
	}
}
