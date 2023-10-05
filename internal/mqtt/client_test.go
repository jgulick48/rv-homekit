package mqtt

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/jgulick48/rv-homekit/internal/models"
)

type MQTTTest struct {
	suite.Suite
	mqtt Client
}

func (s *MQTTTest) SetupTest() {
	config := models.MQTTConfiguration{
		Host:     "192.168.3.86",
		Port:     1883,
		DeviceID: "d41243b4f71d",
	}
	s.mqtt = NewClient(config, models.CurrentLimitConfiguration{}, models.CurrentLimitConfiguration{}, models.ShoreDetection{}, false)
}

func (s *MQTTTest) Test_shouldShutOff_SOC() {
}

func TestAutomateGeneratorStart(t *testing.T) {
	suite.Run(t, new(MQTTTest))
}
