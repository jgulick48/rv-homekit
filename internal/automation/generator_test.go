package automation

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/jgulick48/rv-homekit/internal/bmv"
	"github.com/jgulick48/rv-homekit/internal/models"
)

type GeneratorTest struct {
	suite.Suite
	bmvClient *bmv.MockClient
}

var paramaters = models.Automation{
	HighValue:        99,
	LowValue:         10,
	OffDelay:         models.Duration{Duration: 5 * time.Minute},
	CoolDown:         models.Duration{Duration: time.Hour},
	MinOn:            models.Duration{Duration: 30 * time.Minute},
	MaxOn:            models.Duration{Duration: 3 * time.Hour},
	MinChargeCurrent: 1,
}

func (s *GeneratorTest) SetupTest() {
	s.bmvClient = &bmv.MockClient{}
}

func (s *GeneratorTest) Test_shouldShutOff_SOC() {
	s.bmvClient.On("GetBatteryStateOfCharge").Return(99.1, true).Once()
	turnOff := shouldShutOff(paramaters, time.Now().Add(time.Hour*-2), s.bmvClient)
	s.Assert().True(turnOff)
}

func (s *GeneratorTest) Test_shouldShutOff_MinRunTime() {
	turnOff := shouldShutOff(paramaters, time.Now().Add(time.Minute*-2), s.bmvClient)
	s.Assert().False(turnOff)
}

func (s *GeneratorTest) Test_shouldShutOff_MaxOn() {
	turnOff := shouldShutOff(paramaters, time.Now().Add(time.Hour*-4), s.bmvClient)
	s.Assert().True(turnOff)
}

func (s *GeneratorTest) Test_shouldShutOff_MinCurrent() {
	s.bmvClient.On("GetBatteryStateOfCharge").Return(97.1, true).Once()
	s.bmvClient.On("GetBatteryCurrent").Return(-2.1, true).Once()
	turnOff := shouldShutOff(paramaters, time.Now().Add(time.Hour*-2), s.bmvClient)
	s.Assert().True(turnOff)
}

func TestAutomateGeneratorStart(t *testing.T) {
	suite.Run(t, new(GeneratorTest))
}
