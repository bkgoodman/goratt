//go:build !linux

package rotary

import "errors"

var ErrNotSupported = errors.New("rotary encoder not supported on this platform")

// Rotary is a stub for non-linux platforms.
type Rotary struct{}

// Config holds configuration for a rotary encoder.
type Config struct {
	Chip      string `yaml:"chip"`
	CLKPin    int    `yaml:"clk_pin"`
	DTPin     int    `yaml:"dt_pin"`
	ButtonPin int    `yaml:"button_pin"`
}

// Handlers holds callback functions for rotary events.
type Handlers struct {
	OnTurn      func(delta int)
	OnPress     func()
	OnLongPress func()
}

// New returns an error on non-linux platforms.
func New(cfg Config, handlers Handlers) (*Rotary, error) {
	if cfg.CLKPin == 0 && cfg.DTPin == 0 {
		return nil, nil
	}
	return nil, ErrNotSupported
}

func (r *Rotary) Position() int64 { return 0 }
func (r *Rotary) Release() error  { return nil }
