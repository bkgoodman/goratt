package reader

import "context"

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
}

// New creates a TagReader based on the provided configuration.
func New(cfg Config) (TagReader, error) {
	switch cfg.Type {
	case "wiegand":
		return NewWiegand(cfg.Device, cfg.Baud)
	case "keyboard", "10h-kbd":
		return NewKeyboard(cfg.Device)
	case "serial":
		return NewSerial(cfg.Device)
	default:
		// Default to serial for backwards compatibility
		return NewSerial(cfg.Device)
	}
}
