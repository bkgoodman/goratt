//go:build linux

package rotary

import (
	"fmt"
	"log"
	"sync"
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
	stopCh            chan struct{}
	lastTurnEventTime time.Time

	// Button state - protected by btnMu.
	// Accessed by both pollButton goroutine and longPressTimer goroutine.
	btnMu          sync.Mutex
	btnPressTime   time.Time
	longPressTimer *time.Timer
	longPressFired bool
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
				if time.Since(r.lastTurnEventTime) < 5*time.Millisecond {
					continue
				}
				r.lastTurnEventTime = time.Now()

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

// pollButton uses level-change detection with debounce to reliably detect
// button presses and releases.
//
// Why not just use WaitForEdge + Read?
// WaitForEdge returns when a kernel-queued edge event is consumed, but Read()
// returns the CURRENT pin level, not the level at the time of the edge. With
// BothEdges + mechanical bounce, multiple edges queue up and Read() can return
// the same level for different edges, making transitions invisible. By tracking
// the last known stable level and requiring a confirmed level change after a
// debounce delay, we are immune to edge-level desynchronization and bounce.
func (r *Rotary) pollButton() {
	// Read the initial stable level before entering the loop
	lastStableLevel := r.btnPin.Read()
	log.Printf("Button poll started, initial level: %v", lastStableLevel)

	const debounceDelay = 15 * time.Millisecond

	for {
		select {
		case <-r.stopCh:
			return
		default:
		}

		// Wait for an edge event or timeout. The edge wakes us up quickly
		// when the pin changes; the timeout ensures we don't miss anything
		// if an edge event is lost by the kernel.
		r.btnPin.WaitForEdge(100 * time.Millisecond)

		// Read the current pin level
		level := r.btnPin.Read()

		// Only process when we see a level that differs from last stable state.
		// This filters out stale/duplicate edge events that don't represent
		// an actual state change.
		if level == lastStableLevel {
			continue
		}

		// We see a different level. Wait for bounce to settle.
		time.Sleep(debounceDelay)

		// Re-read to confirm the level is stable after debounce
		confirmedLevel := r.btnPin.Read()

		// If it bounced back to the previous stable level, ignore it
		if confirmedLevel == lastStableLevel {
			continue
		}

		// Confirmed level change
		lastStableLevel = confirmedLevel

		if confirmedLevel == gpio.Low {
			// Button pressed (active low with pull-up)
			r.btnMu.Lock()
			r.btnPressTime = time.Now()
			r.longPressFired = false

			if r.longPressTimer != nil {
				r.longPressTimer.Stop()
			}
			r.longPressTimer = time.AfterFunc(1*time.Second, func() {
				r.btnMu.Lock()
				defer r.btnMu.Unlock()
				if !r.btnPressTime.IsZero() && !r.longPressFired {
					r.longPressFired = true
					if r.onLongPress != nil {
						fmt.Println("Button long press")
						r.onLongPress()
					}
				}
			})
			r.btnMu.Unlock()
		} else {
			// Button released
			r.btnMu.Lock()
			if r.longPressTimer != nil {
				r.longPressTimer.Stop()
			}

			pressTime := r.btnPressTime
			longPressFired := r.longPressFired

			r.btnPressTime = time.Time{}
			r.longPressFired = false
			r.btnMu.Unlock()

			if !pressTime.IsZero() && !longPressFired {
				// Short press - confirmed by debounce, no duration filter needed
				if r.onPress != nil {
					fmt.Println("Button short press")
					r.onPress(pressTime)
				}
			} else if longPressFired {
				// Released after long press
				fmt.Println("Button long press released")
				if r.onButtonUp != nil {
					r.onButtonUp()
				}
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
	r.btnMu.Lock()
	if r.longPressTimer != nil {
		r.longPressTimer.Stop()
	}
	r.btnMu.Unlock()

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
