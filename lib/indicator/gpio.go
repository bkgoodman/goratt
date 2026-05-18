package indicator

import (
	"fmt"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
)

// GPIO implements Indicator using discrete GPIO LED pins.
type GPIO struct {
	greenPin  gpio.PinIO
	yellowPin gpio.PinIO
	redPin    gpio.PinIO
}

// NewGPIO creates a new GPIO-based indicator.
func NewGPIO(greenPinNum, yellowPinNum, redPinNum *uint8) (*GPIO, error) {
	g := &GPIO{}

	var err error
	if greenPinNum != nil {
		g.greenPin, err = lookupPin(*greenPinNum)
		if err != nil {
			return nil, err
		}
		g.greenPin.Out(gpio.Low)
	}
	if yellowPinNum != nil {
		g.yellowPin, err = lookupPin(*yellowPinNum)
		if err != nil {
			return nil, err
		}
		g.yellowPin.Out(gpio.Low)
	}
	if redPinNum != nil {
		g.redPin, err = lookupPin(*redPinNum)
		if err != nil {
			return nil, err
		}
		g.redPin.Out(gpio.Low)
	}

	return g, nil
}

// lookupPin resolves a GPIO pin number to a periph.io PinIO.
func lookupPin(num uint8) (gpio.PinIO, error) {
	pinName := fmt.Sprintf("GPIO%d", num)
	p := gpioreg.ByName(pinName)
	if p == nil {
		return nil, fmt.Errorf("GPIO pin %s not found", pinName)
	}
	return p, nil
}

// Idle implements Indicator.Idle.
func (g *GPIO) Idle() {
	g.allOff()
}

// Granted implements Indicator.Granted.
func (g *GPIO) Granted(info *AccessInfo) {
	g.allOff()
	if g.greenPin != nil {
		g.greenPin.Out(gpio.High)
	}
}

// Denied implements Indicator.Denied.
func (g *GPIO) Denied(info *AccessInfo) {
	g.allOff()
	if g.redPin != nil {
		g.redPin.Out(gpio.High)
	}
}

// Opening implements Indicator.Opening.
func (g *GPIO) Opening(info *AccessInfo) {
	g.allOff()
	if g.yellowPin != nil {
		g.yellowPin.Out(gpio.High)
	}
}

// ConnectionLost implements Indicator.ConnectionLost.
func (g *GPIO) ConnectionLost() {
	g.allOff()
	// Blink yellow and red together for connection lost
	if g.yellowPin != nil {
		g.yellowPin.Out(gpio.High)
	}
	if g.redPin != nil {
		g.redPin.Out(gpio.High)
	}
}

// Shutdown implements Indicator.Shutdown.
func (g *GPIO) Shutdown() {
	g.allOff()
}

// Release implements Indicator.Release.
func (g *GPIO) Release() error {
	g.allOff()
	// Halt all pins to release resources
	if g.greenPin != nil {
		g.greenPin.Halt()
	}
	if g.yellowPin != nil {
		g.yellowPin.Halt()
	}
	if g.redPin != nil {
		g.redPin.Halt()
	}
	return nil
}

func (g *GPIO) allOff() {
	if g.greenPin != nil {
		g.greenPin.Out(gpio.Low)
	}
	if g.yellowPin != nil {
		g.yellowPin.Out(gpio.Low)
	}
	if g.redPin != nil {
		g.redPin.Out(gpio.Low)
	}
}
