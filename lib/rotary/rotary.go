//go:build linux

package rotary

import (
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"periph.io/x/conn/v3/gpio"
	"periph.io/x/conn/v3/gpio/gpioreg"
)

// Rotary handles a rotary encoder input device.
type Rotary struct {
	dtPin          gpio.PinIO
	clkPin         gpio.PinIO
	btnPin         gpio.PinIO
	lastCLK        int
	lastDT         int
	pos            int64
	onTurn         func(delta int)
	onPress        func(pressedAt time.Time)
	onLongPress    func()
	onButtonUp     func()
	btnPressTime   time.Time
	longPressTimer *time.Timer
	longPressFired bool
	stopCh         chan struct{}
}

// Config holds configuration for a rotary encoder.
type Config struct {
	Chip      string `yaml:"chip"` // unused with periph.io, kept for config compat
	CLKPin    int    `yaml:"clk_pin"`
	DTPin     int    `yaml:"dt_pin"`
	ButtonPin int    `yaml:"button_pin"`
}

// Handlers holds callback functions for rotary events.
type Handlers struct {
	OnTurn      func(delta int) // Called with +1 (CW) or -1 (CCW)
	OnPress     func(pressedAt time.Time)          // Called when button pressed (short press)
	OnLongPress func()          // Called when button held >1s
	OnButtonUp  func()          // Called when button released (after long press)
}

// New creates a new rotary encoder handler.
// Returns nil if config has no pins specified (CLKPin and DTPin both 0).
func New(cfg Config, handlers Handlers) (*Rotary, error) {
	// If no pins configured, return nil (rotary disabled)
	if cfg.CLKPin == 0 && cfg.DTPin == 0 {
		return nil, nil
	}

	r := &Rotary{
		onTurn:      handlers.OnTurn,
		onPress:     handlers.OnPress,
		onLongPress: handlers.OnLongPress,
		onButtonUp:  handlers.OnButtonUp,
		stopCh:      make(chan struct{}),
	}

	// Lookup DT pin
	dtName := fmt.Sprintf("GPIO%d", cfg.DTPin)
	r.dtPin = gpioreg.ByName(dtName)
	if r.dtPin == nil {
		return nil, fmt.Errorf("GPIO pin %s not found", dtName)
	}
	if err := r.dtPin.In(gpio.PullUp, gpio.BothEdges); err != nil {
		return nil, fmt.Errorf("configure DT pin: %w", err)
	}

	// Lookup CLK pin
	clkName := fmt.Sprintf("GPIO%d", cfg.CLKPin)
	r.clkPin = gpioreg.ByName(clkName)
	if r.clkPin == nil {
		return nil, fmt.Errorf("GPIO pin %s not found", clkName)
	}
	if err := r.clkPin.In(gpio.PullUp, gpio.BothEdges); err != nil {
		r.dtPin.Halt()
		return nil, fmt.Errorf("configure CLK pin: %w", err)
	}

	// Lookup button pin if specified
	if cfg.ButtonPin > 0 {
		btnName := fmt.Sprintf("GPIO%d", cfg.ButtonPin)
		r.btnPin = gpioreg.ByName(btnName)
		if r.btnPin == nil {
			r.dtPin.Halt()
			r.clkPin.Halt()
			return nil, fmt.Errorf("GPIO pin %s not found", btnName)
		}
		if err := r.btnPin.In(gpio.PullUp, gpio.BothEdges); err != nil {
			r.dtPin.Halt()
			r.clkPin.Halt()
			return nil, fmt.Errorf("configure button pin: %w", err)
		}
	}

	// Start polling goroutines for edge detection
	go r.pollRotary()
	if r.btnPin != nil {
		go r.pollButton()
	}

	return r, nil
}

// pollRotary waits for edges on CLK and DT pins and decodes rotation.
func (r *Rotary) pollRotary() {
	for {
		select {
		case <-r.stopCh:
			return
		default:
		}

		// Wait for edge on CLK pin (with timeout to allow shutdown)
		if r.clkPin.WaitForEdge(100 * time.Millisecond) {
			clkLevel := r.clkPin.Read()
			dtLevel := r.dtPin.Read()

			newCLK := 0
			if clkLevel == gpio.High {
				newCLK = 1
			}
			newDT := 0
			if dtLevel == gpio.High {
				newDT = 1
			}

			r.lastCLK = newCLK
			r.lastDT = newDT

			// Decode direction on CLK rising edge
			if clkLevel == gpio.High {
				if dtLevel == gpio.Low {
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
	}
}

// pollButton waits for edges on the button pin and handles press/release.
func (r *Rotary) pollButton() {
	for {
		select {
		case <-r.stopCh:
			return
		default:
		}

		if r.btnPin.WaitForEdge(100 * time.Millisecond) {
			level := r.btnPin.Read()

			if level == gpio.Low {
				// Button pressed (active low with pull-up)
				r.btnPressTime = time.Now()
				r.longPressFired = false

				if r.longPressTimer != nil {
					r.longPressTimer.Stop()
				}
				r.longPressTimer = time.AfterFunc(1*time.Second, func() {
					if !r.btnPressTime.IsZero() && !r.longPressFired {
						r.longPressFired = true
						if r.onLongPress != nil {
							fmt.Println("Button long press")
							r.onLongPress()
						}
					}
				})
			} else {
				// Button released
				if r.longPressTimer != nil {
					r.longPressTimer.Stop()
				}

				if !r.btnPressTime.IsZero() && !r.longPressFired {
					// Short press - require at least 50ms duration to filter out mechanical bounce
					if time.Since(r.btnPressTime) > 50*time.Millisecond {
						if r.onPress != nil {
							r.onPress(r.btnPressTime)
						}
					}
				} else if r.longPressFired {
					// Released after long press
					if r.onButtonUp != nil {
						r.onButtonUp()
					}
				}

				r.btnPressTime = time.Time{}
				r.longPressFired = false
			}
		}
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
	close(r.stopCh)
	if r.dtPin != nil {
		if err := r.dtPin.Halt(); err != nil {
			log.Printf("Warning: failed to halt DT pin: %v", err)
		}
	}
	if r.clkPin != nil {
		if err := r.clkPin.Halt(); err != nil {
			log.Printf("Warning: failed to halt CLK pin: %v", err)
		}
	}
	if r.btnPin != nil {
		if err := r.btnPin.Halt(); err != nil {
			log.Printf("Warning: failed to halt button pin: %v", err)
		}
	}
	return nil
}
