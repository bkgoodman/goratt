//go:build linux

package rotary

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/warthog618/go-gpiocdev"
)

// Rotary handles a rotary encoder input device.
type Rotary struct {
	dtLine         *gpiocdev.Line
	clkLine        *gpiocdev.Line
	btnLine        *gpiocdev.Line
	lastCLK        int
	lastDT         int
	pos            int64
	onTurn         func(delta int)
	onPress        func()
	onLongPress    func()
	btnPressTime   time.Time
	longPressTimer *time.Timer
	longPressFired bool
}

// Config holds configuration for a rotary encoder.
type Config struct {
	Chip      string `yaml:"chip"`
	CLKPin    int    `yaml:"clk_pin"`
	DTPin     int    `yaml:"dt_pin"`
	ButtonPin int    `yaml:"button_pin"`
}

// Handlers holds callback functions for rotary events.
type Handlers struct {
	OnTurn      func(delta int) // Called with +1 (CW) or -1 (CCW)
	OnPress     func()          // Called when button pressed (short press)
	OnLongPress func()          // Called when button held >1s
}

// New creates a new rotary encoder handler.
// Returns nil if config has no pins specified (CLKPin and DTPin both 0).
func New(cfg Config, handlers Handlers) (*Rotary, error) {
	// If no pins configured, return nil (rotary disabled)
	if cfg.CLKPin == 0 && cfg.DTPin == 0 {
		return nil, nil
	}

	if cfg.Chip == "" {
		cfg.Chip = "gpiochip0"
	}

	debounceRotary := 250 * time.Microsecond
	debounceButton := 2 * time.Millisecond

	r := &Rotary{
		onTurn:      handlers.OnTurn,
		onPress:     handlers.OnPress,
		onLongPress: handlers.OnLongPress,
	}

	var err error

	// Request DT line
	r.dtLine, err = gpiocdev.RequestLine(cfg.Chip, cfg.DTPin,
		gpiocdev.WithPullUp,
		gpiocdev.WithBothEdges,
		gpiocdev.WithDebounce(debounceRotary),
		gpiocdev.WithEventHandler(r.handleEvent))
	if err != nil {
		return nil, err
	}

	// Request CLK line
	r.clkLine, err = gpiocdev.RequestLine(cfg.Chip, cfg.CLKPin,
		gpiocdev.WithPullUp,
		gpiocdev.WithBothEdges,
		gpiocdev.WithDebounce(debounceRotary),
		gpiocdev.WithEventHandler(r.handleEvent))
	if err != nil {
		r.dtLine.Close()
		return nil, err
	}

	// Request button line if specified (both edges to detect press and release)
	if cfg.ButtonPin > 0 {
		r.btnLine, err = gpiocdev.RequestLine(cfg.Chip, cfg.ButtonPin,
			gpiocdev.WithPullUp,
			gpiocdev.WithBothEdges,
			gpiocdev.WithDebounce(debounceButton),
			gpiocdev.WithEventHandler(r.handleButton))
		if err != nil {
			r.dtLine.Close()
			r.clkLine.Close()
			return nil, err
		}
	}

	return r, nil
}

func (r *Rotary) handleEvent(evt gpiocdev.LineEvent) {
	var newState int
	if evt.Type == gpiocdev.LineEventRisingEdge {
		newState = 1
	} else if evt.Type == gpiocdev.LineEventFallingEdge {
		newState = 0
	} else {
		return
	}

	switch evt.Offset {
	case r.clkLine.Offset():
		r.lastCLK = newState
	case r.dtLine.Offset():
		r.lastDT = newState
	}

	// Decode direction on CLK rising edge
	if evt.Offset == r.clkLine.Offset() && evt.Type == gpiocdev.LineEventRisingEdge {
		//fmt.Printf("Rotary %d\n", r.lastDT)
		if r.lastDT == 0 {
			atomic.AddInt64(&r.pos, 1)
			if r.onTurn != nil {
				r.onTurn(1)
			}
		} else {
			atomic.AddInt64(&r.pos, -1)
			if r.onTurn != nil {
				r.onTurn(-1)
			}
		}
	}
}

func (r *Rotary) handleButton(evt gpiocdev.LineEvent) {
	if evt.Type == gpiocdev.LineEventFallingEdge {
		// Button pressed - start long-press timer
		r.btnPressTime = time.Now()
		r.longPressFired = false
		//fmt.Println("Button pressed")

		// Start timer for long press
		if r.longPressTimer != nil {
			r.longPressTimer.Stop()
		}
		r.longPressTimer = time.AfterFunc(1*time.Second, func() {
			// Long press triggered after 1s hold
			if !r.btnPressTime.IsZero() && !r.longPressFired {
				r.longPressFired = true
				if r.onLongPress != nil {
					fmt.Println("Button long press")
					r.onLongPress()
				}
			}
		})
	} else if evt.Type == gpiocdev.LineEventRisingEdge {
		// Button released
		if r.longPressTimer != nil {
			r.longPressTimer.Stop()
		}

		if !r.btnPressTime.IsZero() && !r.longPressFired {
			// Short press (released before 1s)
			if r.onPress != nil {
				//fmt.Println("Button short press")
				r.onPress()
			}
		}

		r.btnPressTime = time.Time{}
		r.longPressFired = false
	}
}

// Position returns the current encoder position.
func (r *Rotary) Position() int64 {
	return atomic.LoadInt64(&r.pos)
}

// Release releases GPIO resources.
func (r *Rotary) Release() error {
	if r.longPressTimer != nil {
		r.longPressTimer.Stop()
	}
	if r.dtLine != nil {
		r.dtLine.Close()
	}
	if r.clkLine != nil {
		r.clkLine.Close()
	}
	if r.btnLine != nil {
		r.btnLine.Close()
	}
	return nil
}
