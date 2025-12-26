package door

import (
	"fmt"

	"github.com/hjkoskel/govattu"
)

// DoorOpener is the interface for all door control implementations.
type DoorOpener interface {
	// Open activates the door opener (unlocks/opens the door).
	Open() error

	// Close deactivates the door opener (locks/closes the door).
	Close() error

	// Release releases any hardware resources.
	Release() error
}

// Config holds configuration for door opener implementations.
type Config struct {
	Type       string `yaml:"type"`        // "servo", "gpio_high", "gpio_low", "none"
	Pin        *int   `yaml:"pin"`         // GPIO pin number
	ServoOpen  int    `yaml:"servo_open"`  // PWM value for open position
	ServoClose int    `yaml:"servo_close"` // PWM value for closed position
}

// New creates a DoorOpener based on the provided configuration.
func New(cfg Config) (DoorOpener, error) {
	if cfg.Pin == nil {
		return &Noop{}, nil
	}

	hw, err := govattu.Open()
	if err != nil {
		return nil, fmt.Errorf("open gpio: %w", err)
	}

	switch cfg.Type {
	case "servo":
		return NewServo(hw, uint8(*cfg.Pin), cfg.ServoOpen, cfg.ServoClose)
	case "gpio_high", "openhigh":
		return NewGPIO(hw, uint8(*cfg.Pin), true)
	case "gpio_low", "openlow":
		return NewGPIO(hw, uint8(*cfg.Pin), false)
	default:
		hw.Close()
		return &Noop{}, nil
	}
}
