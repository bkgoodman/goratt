package mqtt

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

// Client wraps the MQTT client with application-specific functionality.
type Client struct {
	client       paho.Client
	clientID     string
	enabled      bool
	onConnect    func()
	onDisconnect func()
	onMessage    func(topic string, payload []byte)
}

// Config holds MQTT connection settings.
type Config struct {
	Host       string `yaml:"host"`
	Port       int    `yaml:"port"`
	CACert     string `yaml:"ca_cert"`
	ClientCert string `yaml:"client_cert"`
	ClientKey  string `yaml:"client_key"`
}

// Handlers holds callback functions for MQTT events.
type Handlers struct {
	OnConnect    func()
	OnDisconnect func()
	OnMessage    func(topic string, payload []byte)
}

// New creates a new MQTT client. Returns a disabled no-op client if host is empty.
func New(cfg Config, clientID string, handlers Handlers) (*Client, error) {
	c := &Client{
		clientID:     clientID,
		onConnect:    handlers.OnConnect,
		onDisconnect: handlers.OnDisconnect,
		onMessage:    handlers.OnMessage,
	}

	// If no host configured, return disabled client
	if cfg.Host == "" {
		c.enabled = false
		log.Println("MQTT disabled (no host configured)")
		return c, nil
	}

	c.enabled = true

	// Determine broker URL and TLS config
	var broker string
	var tlsConfig *tls.Config

	hasTLS := cfg.CACert != "" || cfg.ClientCert != ""

	if hasTLS {
		broker = fmt.Sprintf("ssl://%s:%d", cfg.Host, cfg.Port)

		var err error
		tlsConfig, err = buildTLSConfig(cfg)
		if err != nil {
			return nil, fmt.Errorf("build TLS config: %w", err)
		}
	} else {
		// Non-TLS connection
		if cfg.Port == 0 {
			cfg.Port = 1883 // Default non-TLS MQTT port
		}
		broker = fmt.Sprintf("tcp://%s:%d", cfg.Host, cfg.Port)
		log.Println("MQTT using non-TLS connection")
	}

	opts := paho.NewClientOptions().
		AddBroker(broker).
		SetClientID(clientID).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetKeepAlive(60 * time.Second).
		SetConnectionLostHandler(c.handleConnectionLost).
		SetOnConnectHandler(c.handleConnect).
		SetDefaultPublishHandler(c.handleMessage)

	if tlsConfig != nil {
		opts.SetTLSConfig(tlsConfig)
	}

	c.client = paho.NewClient(opts)

	// Set up logging
	paho.ERROR = log.New(os.Stdout, "[MQTT ERROR] ", 0)
	paho.CRITICAL = log.New(os.Stdout, "[MQTT CRIT] ", 0)
	paho.WARN = log.New(os.Stdout, "[MQTT WARN] ", 0)

	return c, nil
}

func buildTLSConfig(cfg Config) (*tls.Config, error) {
	tlsConfig := &tls.Config{}

	// Load CA cert if provided
	if cfg.CACert != "" {
		caCert, err := ioutil.ReadFile(cfg.CACert)
		if err != nil {
			return nil, fmt.Errorf("read CA cert: %w", err)
		}
		caPool := x509.NewCertPool()
		caPool.AppendCertsFromPEM(caCert)
		tlsConfig.RootCAs = caPool
	}

	// Load client cert if provided
	if cfg.ClientCert != "" && cfg.ClientKey != "" {
		cert, err := tls.LoadX509KeyPair(cfg.ClientCert, cfg.ClientKey)
		if err != nil {
			return nil, fmt.Errorf("load client cert: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}

// Connect connects to the MQTT broker. If disabled, calls onConnect immediately.
func (c *Client) Connect() error {
	if !c.enabled {
		// When MQTT is disabled, simulate successful connection
		// so indicator goes to Idle state instead of ConnectionLost
		if c.onConnect != nil {
			c.onConnect()
		}
		return nil
	}

	if token := c.client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("connect: %w", token.Error())
	}
	log.Println("MQTT connected")
	return nil
}

// Disconnect disconnects from the MQTT broker. No-op if disabled.
func (c *Client) Disconnect() {
	if !c.enabled || c.client == nil {
		return
	}
	c.client.Disconnect(250)
}

// Subscribe subscribes to a topic. No-op if disabled.
func (c *Client) Subscribe(topic string) error {
	if !c.enabled {
		return nil
	}

	if token := c.client.Subscribe(topic, 0, nil); token.Wait() && token.Error() != nil {
		return fmt.Errorf("subscribe %s: %w", topic, token.Error())
	}
	return nil
}

// Publish publishes a message to a topic. No-op if disabled.
func (c *Client) Publish(topic string, payload string) {
	if !c.enabled {
		return
	}
	c.client.Publish(topic, 0, false, payload)
}

// IsEnabled returns whether MQTT is enabled.
func (c *Client) IsEnabled() bool {
	return c.enabled
}

func (c *Client) handleConnect(client paho.Client) {
	log.Println("MQTT connection established")
	if c.onConnect != nil {
		c.onConnect()
	}
}

func (c *Client) handleConnectionLost(client paho.Client, err error) {
	log.Printf("MQTT connection lost: %v", err)
	if c.onDisconnect != nil {
		c.onDisconnect()
	}
}

func (c *Client) handleMessage(client paho.Client, msg paho.Message) {
	if c.onMessage != nil {
		c.onMessage(msg.Topic(), msg.Payload())
	}
}
