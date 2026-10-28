package main

import (
	"testing"

	"gopkg.in/yaml.v2"
)

func TestAppConfigLegacyDecoding(t *testing.T) {
	legacyYAML := `
CACert: "certs/ca.crt"
ClientCert: "certs/client_goratt_test.crt"
ClientKey: "certs/client_goratt_test.key"
ClientID: "b827eb9928c0"
MqttHost: "mqtt"
MqttPort: 8883
Mode: servo

ApiCAFile: "certs/auth-ca.crt"
ApiURL: "https://auth.makeitlabs.com/authit/"
ApiUsername: "goratt_test"
ApiPassword: "secretpassword"
Resource: "frontdoor"

ServoOpen: 800
ServoClose: 1700
WaitSecs: 5
NFCdevice: /dev/ttyUSB0

TagFile: "validtags.txt"
`

	var cfg AppConfig
	if err := yaml.Unmarshal([]byte(legacyYAML), &cfg); err != nil {
		t.Fatalf("Unmarshal legacy config into AppConfig: %v", err)
	}
	cfg.LabelPrinter.SetDefaults()

	if cfg.ClientID != "b827eb9928c0" {
		t.Errorf("expected ClientID b827eb9928c0, got %s", cfg.ClientID)
	}
	if cfg.MQTT.Host != "mqtt" {
		t.Errorf("expected MQTT.Host mqtt, got %s", cfg.MQTT.Host)
	}
	if cfg.Reader.Type != "serial" || cfg.Reader.Device != "/dev/ttyUSB0" {
		t.Errorf("expected Reader serial /dev/ttyUSB0, got %s %s", cfg.Reader.Type, cfg.Reader.Device)
	}
	if cfg.LabelPrinter.Button1Pin != 18 {
		t.Errorf("expected Button1Pin 18, got %d", cfg.LabelPrinter.Button1Pin)
	}
	if cfg.LabelPrinter.Button2Pin != 25 {
		t.Errorf("expected Button2Pin 25, got %d", cfg.LabelPrinter.Button2Pin)
	}
	if cfg.LabelPrinter.PrintOnBoot == nil || !*cfg.LabelPrinter.PrintOnBoot {
		t.Errorf("expected PrintOnBoot true by default, got %v", cfg.LabelPrinter.PrintOnBoot)
	}
}

