package main

import (
	"goratt/door"
	"goratt/indicator"
	"goratt/reader"
)

// Config is the main configuration structure for GoRATT.
type Config struct {
	// MQTT connection settings
	MQTT MQTTConfig `yaml:"mqtt"`

	// API settings for ACL backend
	API APIConfig `yaml:"api"`

	// Reader configuration
	Reader reader.Config `yaml:"reader"`

	// Door opener configuration
	Door door.Config `yaml:"door"`

	// Indicator configuration
	Indicator indicator.Config `yaml:"indicator"`

	// General settings
	ClientID     string `yaml:"client_id"`
	Resource     string `yaml:"resource"`
	TagFile      string `yaml:"tag_file"`
	WaitSecs     int    `yaml:"wait_secs"`
	OpenSecret   string `yaml:"open_secret"`
	OpenToolName string `yaml:"open_tool_name"`
}

// MQTTConfig holds MQTT broker connection settings.
type MQTTConfig struct {
	Host       string `yaml:"host"`
	Port       int    `yaml:"port"`
	CACert     string `yaml:"ca_cert"`
	ClientCert string `yaml:"client_cert"`
	ClientKey  string `yaml:"client_key"`
}

// APIConfig holds API backend settings.
type APIConfig struct {
	URL      string `yaml:"url"`
	CAFile   string `yaml:"ca_file"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}
