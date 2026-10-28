package config

import (
	"testing"

	"gopkg.in/yaml.v2"
)

func TestLegacyConfigDecoding(t *testing.T) {
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

	var cfg Config
	if err := yaml.Unmarshal([]byte(legacyYAML), &cfg); err != nil {
		t.Fatalf("Unmarshal legacy config: %v", err)
	}

	if cfg.ClientID != "b827eb9928c0" {
		t.Errorf("expected ClientID b827eb9928c0, got %s", cfg.ClientID)
	}
	if cfg.MQTT.Host != "mqtt" {
		t.Errorf("expected MQTT.Host mqtt, got %s", cfg.MQTT.Host)
	}
	if cfg.MQTT.Port != 8883 {
		t.Errorf("expected MQTT.Port 8883, got %d", cfg.MQTT.Port)
	}
	if cfg.MQTT.CACert != "certs/ca.crt" {
		t.Errorf("expected MQTT.CACert certs/ca.crt, got %s", cfg.MQTT.CACert)
	}
	if cfg.API.URL != "https://auth.makeitlabs.com/authit/" {
		t.Errorf("expected API.URL https://auth.makeitlabs.com/authit/, got %s", cfg.API.URL)
	}
	if cfg.API.CAFile != "certs/auth-ca.crt" {
		t.Errorf("expected API.CAFile certs/auth-ca.crt, got %s", cfg.API.CAFile)
	}
	if cfg.Reader.Type != "serial" {
		t.Errorf("expected Reader.Type serial, got %s", cfg.Reader.Type)
	}
	if cfg.Reader.Device != "/dev/ttyUSB0" {
		t.Errorf("expected Reader.Device /dev/ttyUSB0, got %s", cfg.Reader.Device)
	}
	if cfg.TagFile != "validtags.txt" {
		t.Errorf("expected TagFile validtags.txt, got %s", cfg.TagFile)
	}
}

func TestModernConfigDecoding(t *testing.T) {
	modernYAML := `
client_id: "testnode"
tag_file: "tags.txt"
mqtt:
  host: "test.broker"
  port: 1883
reader:
  type: "wiegand"
  device: "/dev/serial0"
`

	var cfg Config
	if err := yaml.Unmarshal([]byte(modernYAML), &cfg); err != nil {
		t.Fatalf("Unmarshal modern config: %v", err)
	}

	if cfg.ClientID != "testnode" {
		t.Errorf("expected ClientID testnode, got %s", cfg.ClientID)
	}
	if cfg.MQTT.Host != "test.broker" {
		t.Errorf("expected MQTT.Host test.broker, got %s", cfg.MQTT.Host)
	}
	if cfg.Reader.Type != "wiegand" {
		t.Errorf("expected Reader.Type wiegand, got %s", cfg.Reader.Type)
	}
}

