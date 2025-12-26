package door

import (
	"github.com/hjkoskel/govattu"
)

// GPIO implements DoorOpener using simple GPIO pin control.
type GPIO struct {
	hw       govattu.Vattu
	pin      uint8
	openHigh bool // true = set pin high to open, false = set pin low to open
}

// NewGPIO creates a new GPIO-based door opener.
func NewGPIO(hw govattu.Vattu, pin uint8, openHigh bool) (*GPIO, error) {
	hw.PinMode(pin, govattu.ALToutput)

	g := &GPIO{
		hw:       hw,
		pin:      pin,
		openHigh: openHigh,
	}

	// Start in closed state
	g.Close()
	return g, nil
}

// Open implements DoorOpener.Open.
func (g *GPIO) Open() error {
	if g.openHigh {
		g.hw.PinSet(g.pin)
	} else {
		g.hw.PinClear(g.pin)
	}
	return nil
}

// Close implements DoorOpener.Close.
func (g *GPIO) Close() error {
	if g.openHigh {
		g.hw.PinClear(g.pin)
	} else {
		g.hw.PinSet(g.pin)
	}
	return nil
}

// Release implements DoorOpener.Release.
func (g *GPIO) Release() error {
	return g.hw.Close()
}
