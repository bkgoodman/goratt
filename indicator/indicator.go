package indicator

import "goratt/video"

// Indicator is the interface for status indicator implementations (LEDs, neopixels, etc).
type Indicator interface {
	// Idle sets the indicator to idle/ready state.
	Idle()

	// Granted sets the indicator to access granted state.
	// info may be nil if no ACL information is available.
	Granted(info *AccessInfo)

	// Denied sets the indicator to access denied state.
	// info may be nil if no ACL information is available.
	Denied(info *AccessInfo)

	// Opening sets the indicator to door opening state.
	// info may be nil if no ACL information is available.
	Opening(info *AccessInfo)

	// ConnectionLost sets the indicator to connection lost state.
	ConnectionLost()

	// Shutdown sets the indicator to shutdown state.
	Shutdown()

	// Release releases any hardware resources.
	Release() error
}

// Config holds configuration for indicator implementations.
type Config struct {
	// GPIO LED pins (nil = not configured)
	GreenPin  *uint8 `yaml:"green_pin"`
	YellowPin *uint8 `yaml:"yellow_pin"`
	RedPin    *uint8 `yaml:"red_pin"`

	// Neopixel pipe path (empty = not configured)
	NeopixelPipe string `yaml:"neopixel_pipe"`

	// Video framebuffer display (true = enabled)
	VideoEnabled bool `yaml:"video_enabled"`
}

// New creates an Indicator based on the provided configuration.
// Returns a Multi indicator if both GPIO and Neopixel are configured.
func New(cfg Config) (Indicator, error) {
	var indicators []Indicator

	// Add GPIO indicator if any pins configured
	if cfg.GreenPin != nil || cfg.YellowPin != nil || cfg.RedPin != nil {
		gpio, err := NewGPIO(cfg.GreenPin, cfg.YellowPin, cfg.RedPin)
		if err != nil {
			return nil, err
		}
		indicators = append(indicators, gpio)
	}

	// Add Neopixel indicator if pipe configured
	if cfg.NeopixelPipe != "" {
		neo, err := NewNeopixel(cfg.NeopixelPipe)
		if err != nil {
			return nil, err
		}
		indicators = append(indicators, neo)
	}

	// Add Video indicator if enabled
	if cfg.VideoEnabled {
		if !video.ScreenSupported() {
			return nil, video.ErrScreenNotCompiled
		}
		vid, err := NewVideo()
		if err != nil {
			return nil, err
		}
		indicators = append(indicators, vid)
	}

	if len(indicators) == 0 {
		return &Noop{}, nil
	}
	if len(indicators) == 1 {
		return indicators[0], nil
	}
	return &Multi{indicators: indicators}, nil
}
