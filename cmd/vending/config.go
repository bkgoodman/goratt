package main

import "goratt/lib/config"

// AppConfig extends the shared config with vending-specific options.
type AppConfig struct {
	config.Config `yaml:",inline"`

	Vending struct {
		Product string `yaml:"product"`
	} `yaml:"vending"`
}
