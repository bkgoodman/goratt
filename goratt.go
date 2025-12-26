package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"gopkg.in/yaml.v2"

	"goratt/door"
	"goratt/indicator"
	"goratt/reader"
)

var myBuild string

// App holds the application state and dependencies.
type App struct {
	cfg       *Config
	client    mqtt.Client
	reader    reader.TagReader
	door      door.DoorOpener
	indicator indicator.Indicator
	acl       *ACLManager
	ctx       context.Context
	cancel    context.CancelFunc
}

// OpenRequest represents a remote open request.
type OpenRequest struct {
	Member    string `json:"member"`
	ToolName  string `json:"tool"`
	Timestamp uint64 `json:"timestamp"`
	Signature string `json:"signature"`
}

func main() {
	fmt.Printf("goratt build %s\n", myBuild)

	openflag := flag.Bool("holdopen", false, "Hold door open indefinitely")
	cfgfile := flag.String("cfg", "goratt.cfg", "Config file")
	flag.Parse()

	// Load configuration
	f, err := os.Open(*cfgfile)
	if err != nil {
		log.Fatalf("Open config: %v", err)
	}
	defer f.Close()

	var cfg Config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		log.Fatalf("Decode config: %v", err)
	}

	if cfg.ClientID == "" {
		log.Fatal("client_id missing in config file")
	}

	// Create application context
	ctx, cancel := context.WithCancel(context.Background())

	app := &App{
		cfg:    &cfg,
		ctx:    ctx,
		cancel: cancel,
	}

	// Initialize indicator
	app.indicator, err = indicator.New(cfg.Indicator)
	if err != nil {
		log.Fatalf("Init indicator: %v", err)
	}
	app.indicator.ConnectionLost() // Start with connection lost state

	// Initialize door opener
	app.door, err = door.New(cfg.Door)
	if err != nil {
		log.Fatalf("Init door: %v", err)
	}

	// Initialize tag reader
	app.reader, err = reader.New(cfg.Reader)
	if err != nil {
		log.Fatalf("Init reader: %v", err)
	}

	// Initialize ACL manager
	app.acl = NewACLManager(&cfg)
	app.acl.SetUpdateCallback(func() {
		topic := fmt.Sprintf("ratt/status/node/%s/acl/update", cfg.ClientID)
		app.client.Publish(topic, 0, false, `{"status":"downloaded"}`)
	})

	// Load existing ACL from file, then fetch from API
	if err := app.acl.LoadFromFile(); err != nil {
		log.Printf("Warning: could not load tag file: %v", err)
	}
	if err := app.acl.FetchFromAPI(); err != nil {
		log.Printf("Warning: could not fetch ACL from API: %v", err)
	}

	// Handle holdopen flag
	if *openflag {
		app.openDoor("holdopen")
		select {} // Block forever
	}

	// Initialize MQTT
	app.initMQTT()

	// Start background goroutines
	go app.mqttConnect()
	go app.tagListener()
	go app.pingSender()

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	fmt.Println("Shutting down...")
	cancel()

	// Cleanup
	app.client.Disconnect(250)
	app.reader.Close()
	app.door.Release()
	app.indicator.Shutdown()
	app.indicator.Release()

	fmt.Println("Shutdown complete")
}

func (app *App) initMQTT() {
	broker := fmt.Sprintf("ssl://%s:%d", app.cfg.MQTT.Host, app.cfg.MQTT.Port)

	cert, err := tls.LoadX509KeyPair(app.cfg.MQTT.ClientCert, app.cfg.MQTT.ClientKey)
	if err != nil {
		log.Fatalf("Load X509 keypair: %v", err)
	}

	caCert, err := ioutil.ReadFile(app.cfg.MQTT.CACert)
	if err != nil {
		log.Fatalf("Read CA cert: %v", err)
	}

	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caPool,
	}

	opts := mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID(app.cfg.ClientID).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetKeepAlive(60 * time.Second).
		SetTLSConfig(tlsConfig).
		SetConnectionLostHandler(app.onConnectionLost).
		SetOnConnectHandler(app.onConnect).
		SetDefaultPublishHandler(app.onMessage)

	app.client = mqtt.NewClient(opts)

	mqtt.ERROR = log.New(os.Stdout, "[ERROR] ", 0)
	mqtt.CRITICAL = log.New(os.Stdout, "[CRIT] ", 0)
	mqtt.WARN = log.New(os.Stdout, "[WARN]  ", 0)
}

func (app *App) mqttConnect() {
	if token := app.client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("MQTT connect: %v", token.Error())
	}
	fmt.Println("MQTT connected")
}

func (app *App) onConnect(client mqtt.Client) {
	fmt.Println("MQTT connection established")

	// Subscribe to broadcast ACL update
	topic := "ratt/control/broadcast/acl/update"
	if token := client.Subscribe(topic, 0, nil); token.Wait() && token.Error() != nil {
		log.Printf("Subscribe error: %v", token.Error())
	}

	// Subscribe to node-specific open command
	openTopic := fmt.Sprintf("ratt/control/node/%s/open", app.cfg.ClientID)
	if token := client.Subscribe(openTopic, 0, nil); token.Wait() && token.Error() != nil {
		log.Printf("Subscribe error: %v", token.Error())
	}

	app.indicator.Idle()
}

func (app *App) onConnectionLost(client mqtt.Client, err error) {
	fmt.Printf("MQTT connection lost: %v\n", err)
	app.indicator.ConnectionLost()
}

func (app *App) onMessage(client mqtt.Client, msg mqtt.Message) {
	switch msg.Topic() {
	case "ratt/control/broadcast/acl/update":
		fmt.Println("Received ACL update message")
		if err := app.acl.FetchFromAPI(); err != nil {
			log.Printf("Fetch ACL: %v", err)
		}

	default:
		// Check if it's an open command for this node
		openTopic := fmt.Sprintf("ratt/control/node/%s/open", app.cfg.ClientID)
		if msg.Topic() == openTopic {
			app.handleOpenRequest(msg.Payload())
		}
	}
}

func (app *App) handleOpenRequest(payload []byte) {
	if app.cfg.OpenSecret == "" || app.cfg.OpenToolName == "" {
		fmt.Println("Remote open disabled (no secret or tool name configured)")
		return
	}

	var req OpenRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		log.Printf("Decode open request: %v", err)
		return
	}

	if err := verifySignature(app.cfg.OpenSecret, req.Member, req.ToolName, req.Timestamp, req.Signature); err != nil {
		log.Printf("Signature verification failed: %v", err)
		return
	}

	if req.ToolName != app.cfg.OpenToolName {
		log.Printf("Wrong tool name %q, expected %q", req.ToolName, app.cfg.OpenToolName)
		return
	}

	// Check timestamp is within 5 minute window
	ts := time.Unix(int64(req.Timestamp), 0)
	now := time.Now()
	if now.Before(ts.Add(-5*time.Minute)) || now.After(ts.Add(5*time.Minute)) {
		log.Println("Open request timestamp out of range")
		return
	}

	fmt.Printf("Remote open request from %s\n", req.Member)
	app.publishAccess(req.Member, true)
	app.openDoor(req.Member)
}

func (app *App) tagListener() {
	for {
		select {
		case <-app.ctx.Done():
			return
		default:
		}

		tagID, err := app.reader.Read(app.ctx)
		if err != nil {
			if err == context.Canceled {
				return
			}
			log.Printf("Read tag: %v", err)
			time.Sleep(time.Second)
			continue
		}

		if tagID == 0 {
			continue
		}

		fmt.Printf("Tag read: %d\n", tagID)
		app.handleTag(tagID)
	}
}

func (app *App) handleTag(tagID uint64) {
	record, found := app.acl.Lookup(tagID)

	if !found {
		fmt.Printf("Tag %d not found in ACL\n", tagID)
		app.indicator.Denied()
		time.Sleep(3 * time.Second)
		app.indicator.Idle()
		return
	}

	fmt.Printf("Tag %d: member=%s allowed=%v\n", tagID, record.Member, record.Allowed)
	app.publishAccess(record.Member, record.Allowed)

	if record.Allowed {
		app.openDoor(record.Member)
	} else {
		app.indicator.Denied()
		time.Sleep(3 * time.Second)
		app.indicator.Idle()
	}
}

func (app *App) openDoor(member string) {
	app.indicator.Opening()

	if err := app.door.Open(); err != nil {
		log.Printf("Door open: %v", err)
	}

	app.indicator.Granted()
	time.Sleep(time.Duration(app.cfg.WaitSecs) * time.Second)

	if err := app.door.Close(); err != nil {
		log.Printf("Door close: %v", err)
	}

	app.indicator.Idle()
}

func (app *App) publishAccess(member string, allowed bool) {
	allowedInt := 0
	if allowed {
		allowedInt = 1
	}
	topic := fmt.Sprintf("ratt/status/node/%s/personality/access", app.cfg.ClientID)
	msg := fmt.Sprintf(`{"allowed":%d,"member":"%s"}`, allowedInt, member)
	app.client.Publish(topic, 0, false, msg)
}

func (app *App) pingSender() {
	ticker := time.NewTicker(120 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-app.ctx.Done():
			return
		case <-ticker.C:
			topic := fmt.Sprintf("ratt/status/node/%s/ping", app.cfg.ClientID)
			app.client.Publish(topic, 0, false, `{"status":"ok"}`)
		}
	}
}

// Signature verification helpers

func signOpenRequest(base64Secret, member, tool string, ts uint64) (string, string, error) {
	secret, err := base64.StdEncoding.DecodeString(base64Secret)
	if err != nil {
		return "", "", fmt.Errorf("invalid base64 secret: %w", err)
	}
	if len(secret) == 0 {
		return "", "", fmt.Errorf("secret cannot be empty")
	}

	msg := make([]byte, 0, len(member)+len(tool)+8)
	msg = append(msg, []byte(member)...)
	msg = append(msg, []byte(tool)...)

	var tsBuf [8]byte
	binary.BigEndian.PutUint64(tsBuf[:], ts)
	msg = append(msg, tsBuf[:]...)

	mac := hmac.New(sha256.New, secret)
	mac.Write(msg)
	sum := mac.Sum(nil)

	return hex.EncodeToString(sum), base64.StdEncoding.EncodeToString(sum), nil
}

func verifySignature(base64Secret, member, tool string, ts uint64, providedSig string) error {
	sigHex, sigBase64, err := signOpenRequest(base64Secret, member, tool, ts)
	if err != nil {
		return err
	}

	// Try hex
	if decoded, err := hex.DecodeString(providedSig); err == nil {
		expected, _ := hex.DecodeString(sigHex)
		if subtle.ConstantTimeCompare(decoded, expected) == 1 {
			return nil
		}
	}

	// Try base64
	if decoded, err := base64.StdEncoding.DecodeString(providedSig); err == nil {
		expected, _ := base64.StdEncoding.DecodeString(sigBase64)
		if subtle.ConstantTimeCompare(decoded, expected) == 1 {
			return nil
		}
	}

	return fmt.Errorf("signature verification failed")
}
