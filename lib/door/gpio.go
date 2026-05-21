package door

import (
	"fmt"
	"log"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
)

// GPIO implements DoorOpener using simple GPIO pin control.
type GPIO struct {
	pin      gpio.PinIO
	openHigh bool // true = set pin high to open, false = set pin low to open
}

// NewGPIO creates a new GPIO-based door opener.
func NewGPIO(pinNum int, openHigh bool) (*GPIO, error) {
	pinName := fmt.Sprintf("GPIO%d", pinNum)
	p := gpioreg.ByName(pinName)
	if p == nil {
		return nil, fmt.Errorf("GPIO pin %s not found", pinName)
	}

	g := &GPIO{
		pin:      p,
		openHigh: openHigh,
	}

	// Start in closed state
	if err := g.Close(); err != nil {
		log.Printf("Warning: failed to close GPIO pin: %v", err)
	}
	return g, nil
}

// Open implements DoorOpener.Open.
func (g *GPIO) Open() error {
	if g.openHigh {
		return g.pin.Out(gpio.High)
	}
	return g.pin.Out(gpio.Low)
}

// Close implements DoorOpener.Close.
func (g *GPIO) Close() error {
	if g.openHigh {
		return g.pin.Out(gpio.Low)
	} else {
		return g.pin.Out(gpio.High)
	}
}

// Release implements DoorOpener.Release.
func (g *GPIO) Release() error {
	return g.pin.Out(gpio.Low)
}
