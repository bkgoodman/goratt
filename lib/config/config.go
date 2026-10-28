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

// UnmarshalYAML implements custom unmarshaling to support both the modern
// nested GoRATT configuration format and the legacy flat format.
func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type rawConfig Config
	var raw rawConfig
	if err := unmarshal(&raw); err != nil {
		return err
	}
	*c = Config(raw)

	// Legacy config fields from OLD goratt
	var legacy struct {
		CACert      string `yaml:"CACert"`
		ClientCert  string `yaml:"ClientCert"`
		ClientKey   string `yaml:"ClientKey"`
		ClientID    string `yaml:"ClientID"`
		MqttHost    string `yaml:"MqttHost"`
		MqttPort    int    `yaml:"MqttPort"`
		ApiURL      string `yaml:"ApiURL"`
		ApiCAFile   string `yaml:"ApiCAFile"`
		ApiUsername string `yaml:"ApiUsername"`
		ApiPassword string `yaml:"ApiPassword"`
		Resource    string `yaml:"Resource"`
		TagFile     string `yaml:"TagFile"`
		NFCdevice   string `yaml:"NFCdevice"`
		WaitSecs    int    `yaml:"WaitSecs"`
		Mode        string `yaml:"Mode"`
		ServoOpen   int    `yaml:"ServoOpen"`
		ServoClose  int    `yaml:"ServoClose"`
	}
	if err := unmarshal(&legacy); err == nil {
		if c.MQTT.Host == "" && legacy.MqttHost != "" {
			c.MQTT.Host = legacy.MqttHost
			c.MQTT.Port = legacy.MqttPort
			c.MQTT.CACert = legacy.CACert
			c.MQTT.ClientCert = legacy.ClientCert
			c.MQTT.ClientKey = legacy.ClientKey
		}
		if c.ClientID == "" && legacy.ClientID != "" {
			c.ClientID = legacy.ClientID
		}
		if c.Resource == "" && legacy.Resource != "" {
			c.Resource = legacy.Resource
		}
		if c.TagFile == "" && legacy.TagFile != "" {
			c.TagFile = legacy.TagFile
		}
		if c.WaitSecs == 0 && legacy.WaitSecs != 0 {
			c.WaitSecs = legacy.WaitSecs
		}
		if c.API.URL == "" && legacy.ApiURL != "" {
			c.API.URL = legacy.ApiURL
			c.API.CAFile = legacy.ApiCAFile
			c.API.Username = legacy.ApiUsername
			c.API.Password = legacy.ApiPassword
		}
		if c.Reader.Type == "" && legacy.NFCdevice != "" {
			c.Reader.Type = "serial"
			c.Reader.Device = legacy.NFCdevice
			c.Reader.Baud = 115200
		}
		if c.Door.Type == "" && legacy.Mode != "" {
			c.Door.Type = legacy.Mode
			c.Door.ServoOpen = legacy.ServoOpen
			c.Door.ServoClose = legacy.ServoClose
		}
	}

	return nil
}

