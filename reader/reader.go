package reader

import (
	"context"
	"fmt"
)

// TagReader is the interface for all tag/card reader implementations.
// Implementations should block until a tag is read or context is cancelled.
type TagReader interface {
	// Read blocks until a tag is read or context is cancelled.
	// Returns the tag ID (as uint64) or an error.
	// A return of (0, nil) indicates no tag was read (e.g., timeout).
	Read(ctx context.Context) (uint64, error)

	// Close releases any resources held by the reader.
	Close() error
}

// Config holds common configuration for reader implementations.
type Config struct {
	Type   string `yaml:"type"`   // "wiegand", "keyboard", "serial"
	Device string `yaml:"device"` // e.g., "/dev/serial0", "/dev/input/event0"
	Baud   int    `yaml:"baud"`   // baud rate for serial devices
	Format string `yaml:"format"` // for keyboard: "10h", "8h", "10d", "8d" (digits + h=hex/d=decimal)
}

// New creates a TagReader based on the provided configuration.
func New(cfg Config) (TagReader, error) {
	switch cfg.Type {
	case "wiegand":
		return NewWiegand(cfg.Device, cfg.Baud)
	case "keyboard", "10h-kbd", "10d-kbd", "8h-kbd", "8d-kbd":
		return NewKeyboard(cfg.Type, cfg.Device)
	case "serial":
		return NewSerial(cfg.Device)
	default:
		// Default to serial for backwards compatibility
		return nil, fmt.Errorf("bad reader type \"%s\"", cfg.Type)
	}
}
