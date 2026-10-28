package main

import "goratt/lib/config"

// AppConfig extends the shared GoRATT config with labelprinter-specific options.
type AppConfig struct {
	config.Config `yaml:",inline"`

	LabelPrinter LabelPrinterConfig `yaml:"labelprinter"`
}

func (c *AppConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := unmarshal(&c.Config); err != nil {
		return err
	}
	type rawAppConfig struct {
		LabelPrinter LabelPrinterConfig `yaml:"labelprinter"`
	}
	var raw rawAppConfig
	if err := unmarshal(&raw); err != nil {
		return err
	}
	c.LabelPrinter = raw.LabelPrinter
	return nil
}

// LabelPrinterConfig contains hardware and network configuration for the label printer.
type LabelPrinterConfig struct {
	Device      string `yaml:"device"`        // Path to USB printer device (default: "/dev/usb/lp0")
	HTTPPort    int    `yaml:"http_port"`     // Port for incoming HTTP ArUco print requests (default: 8080)
	Button1Pin  int    `yaml:"button1_pin"`   // GPIO pin for Button 1 / Free Table (default: 18)
	Button2Pin  int    `yaml:"button2_pin"`   // GPIO pin for Button 2 / Storage Pass (default: 25)
	LED1Pin     int    `yaml:"led1_pin"`      // GPIO pin for LED 1 (default: 23)
	LED2Pin     int    `yaml:"led2_pin"`      // GPIO pin for LED 2 (default: 24)
	PrintOnBoot *bool  `yaml:"print_on_boot"` // Whether to print a ready label on boot (default: true)
	Personality string `yaml:"personality"`   // Personality name for MQTT reporting (default: "storagepass")
}

// SetDefaults applies default settings for any unspecified options.
func (c *LabelPrinterConfig) SetDefaults() {
	if c.Device == "" {
		c.Device = "/dev/usb/lp0"
	}
	if c.HTTPPort == 0 {
		c.HTTPPort = 8080
	}
	if c.Button1Pin == 0 {
		c.Button1Pin = 18
	}
	if c.Button2Pin == 0 {
		c.Button2Pin = 25
	}
	if c.LED1Pin == 0 {
		c.LED1Pin = 23
	}
	if c.LED2Pin == 0 {
		c.LED2Pin = 24
	}
	if c.PrintOnBoot == nil {
		defaultPrint := true
		c.PrintOnBoot = &defaultPrint
	}
	if c.Personality == "" {
		c.Personality = "storagepass"
	}
}
