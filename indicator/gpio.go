package indicator

import (
	"fmt"

	"github.com/hjkoskel/govattu"
)

// GPIO implements Indicator using discrete GPIO LED pins.
type GPIO struct {
	hw        govattu.Vattu
	greenPin  *uint8
	yellowPin *uint8
	redPin    *uint8
}

// NewGPIO creates a new GPIO-based indicator.
func NewGPIO(greenPin, yellowPin, redPin *uint8) (*GPIO, error) {
	hw, err := govattu.Open()
	if err != nil {
		return nil, fmt.Errorf("open gpio: %w", err)
	}

	g := &GPIO{
		hw:        hw,
		greenPin:  greenPin,
		yellowPin: yellowPin,
		redPin:    redPin,
	}

	// Initialize all pins as outputs, start off
	if greenPin != nil {
		hw.PinMode(*greenPin, govattu.ALToutput)
		hw.PinClear(*greenPin)
	}
	if yellowPin != nil {
		hw.PinMode(*yellowPin, govattu.ALToutput)
		hw.PinClear(*yellowPin)
	}
	if redPin != nil {
		hw.PinMode(*redPin, govattu.ALToutput)
		hw.PinClear(*redPin)
	}

	return g, nil
}

// Idle implements Indicator.Idle.
func (g *GPIO) Idle() {
	g.allOff()
}

// Granted implements Indicator.Granted.
func (g *GPIO) Granted() {
	g.allOff()
	if g.greenPin != nil {
		g.hw.PinSet(*g.greenPin)
	}
}

// Denied implements Indicator.Denied.
func (g *GPIO) Denied() {
	g.allOff()
	if g.redPin != nil {
		g.hw.PinSet(*g.redPin)
	}
}

// Opening implements Indicator.Opening.
func (g *GPIO) Opening() {
	g.allOff()
	if g.yellowPin != nil {
		g.hw.PinSet(*g.yellowPin)
	}
}

// ConnectionLost implements Indicator.ConnectionLost.
func (g *GPIO) ConnectionLost() {
	g.allOff()
	// Blink yellow and red together for connection lost
	if g.yellowPin != nil {
		g.hw.PinSet(*g.yellowPin)
	}
	if g.redPin != nil {
		g.hw.PinSet(*g.redPin)
	}
}

// Shutdown implements Indicator.Shutdown.
func (g *GPIO) Shutdown() {
	g.allOff()
}

// Release implements Indicator.Release.
func (g *GPIO) Release() error {
	g.allOff()
	return g.hw.Close()
}

func (g *GPIO) allOff() {
	if g.greenPin != nil {
		g.hw.PinClear(*g.greenPin)
	}
	if g.yellowPin != nil {
		g.hw.PinClear(*g.yellowPin)
	}
	if g.redPin != nil {
		g.hw.PinClear(*g.redPin)
	}
}
