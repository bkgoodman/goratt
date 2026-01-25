package config

import (
	"goratt/lib/door"
	"goratt/lib/eventpipe"
	"goratt/lib/indicator"
	"goratt/lib/mqtt"
	"goratt/lib/reader"
	"goratt/lib/rotary"
	"goratt/lib/video"
)

// Config is the main configuration structure for GoRATT.
type Config struct {
	// MQTT connection settings
	MQTT mqtt.Config `yaml:"mqtt"`

	// API settings for ACL backend
	API APIConfig `yaml:"api"`

	// Reader configuration
	Reader reader.Config `yaml:"reader"`

	// Door opener configuration
	Door door.Config `yaml:"door"`

	// Indicator configuration (LEDs, neopixels)
	Indicator indicator.Config `yaml:"indicator"`

	// Video display configuration
	VideoEnabled bool         `yaml:"video_enabled"`
	Video        video.Config `yaml:"video"`

	// Rotary encoder configuration
	Rotary rotary.Config `yaml:"rotary"`

	// Event pipe for external event injection
	EventPipe eventpipe.Config `yaml:"event_pipe"`

	// Audio configuration for PCM playback
	Audio AudioConfig `yaml:"audio"`

	// General settings
	ClientID     string `yaml:"client_id"`
	Resource     string `yaml:"resource"`
	TagFile      string `yaml:"tag_file"`
	WaitSecs     int    `yaml:"wait_secs"`
	OpenSecret   string `yaml:"open_secret"`
	OpenToolName string `yaml:"open_tool_name"`
	CalendarURL  string `yaml:"calendar_url"`
}

// APIConfig holds API backend settings.
type APIConfig struct {
	URL      string `yaml:"url"`
	CAFile   string `yaml:"ca_file"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// AudioConfig holds audio playback settings for PCM files.
type AudioConfig struct {
	Device string `yaml:"device"` // Optional ALSA device name, defaults to "default"
}
