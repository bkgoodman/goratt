package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gopkg.in/yaml.v2"

	"goratt/door"
	"goratt/indicator"
	"goratt/mqtt"
	"goratt/reader"
	"goratt/rotary"
	"goratt/video"
	"goratt/video/screen"
)

var myBuild string

// App holds the application state and dependencies.
type App struct {
	cfg       *Config
	mqtt      *mqtt.Client
	reader    reader.TagReader
	door      door.DoorOpener
	indicator indicator.Indicator
	display   *video.Display
	rotary    *rotary.Rotary
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

	// Initialize indicator (LEDs, neopixels)
	app.indicator, err = indicator.New(cfg.Indicator)
	if err != nil {
		log.Fatalf("Init indicator: %v", err)
	}
	app.indicator.ConnectionLost() // Start with connection lost state

	// Initialize display if enabled
	if cfg.VideoEnabled {
		if !video.ScreenSupported() {
			log.Fatalf("Video enabled but screen support not compiled in")
		}
		app.display, err = video.New()
		if err != nil {
			log.Fatalf("Init display: %v", err)
		}
		app.display.ConnectionLost()
	}

	// Initialize rotary encoder if configured
	app.rotary, err = rotary.New(cfg.Rotary, rotary.Handlers{
		OnTurn:  app.SendRotaryEvent,
		OnPress: app.SendRotaryPressEvent,
	})
	if err != nil {
		log.Fatalf("Init rotary: %v", err)
	}
	if app.rotary != nil {
		log.Printf("Rotary encoder initialized (CLK=%d, DT=%d, BTN=%d)",
			cfg.Rotary.CLKPin, cfg.Rotary.DTPin, cfg.Rotary.ButtonPin)
	}

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
		app.mqtt.Publish(topic, `{"status":"downloaded"}`)
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
		app.openDoor(&indicator.AccessInfo{Member: "holdopen"})
		select {} // Block forever
	}

	// Initialize MQTT
	app.mqtt, err = mqtt.New(cfg.MQTT, cfg.ClientID, mqtt.Handlers{
		OnConnect:    app.onMQTTConnect,
		OnDisconnect: app.onMQTTDisconnect,
		OnMessage:    app.onMQTTMessage,
	})
	if err != nil {
		log.Fatalf("Init MQTT: %v", err)
	}

	// Start background goroutines
	go func() {
		if err := app.mqtt.Connect(); err != nil {
			log.Printf("MQTT connect: %v", err)
		}
	}()
	go app.tagListener()
	go app.pingSender()

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	fmt.Println("Shutting down...")
	cancel()

	// Cleanup
	app.mqtt.Disconnect()
	app.reader.Close()
	app.door.Release()
	app.indicator.Shutdown()
	app.indicator.Release()
	if app.display != nil {
		app.display.Shutdown()
		app.display.Release()
	}
	if app.rotary != nil {
		app.rotary.Release()
	}

	fmt.Println("Shutdown complete")
}

func (app *App) onMQTTConnect() {
	// Subscribe to broadcast ACL update
	if err := app.mqtt.Subscribe("ratt/control/broadcast/acl/update"); err != nil {
		log.Printf("Subscribe error: %v", err)
	}

	// Subscribe to node-specific open command
	openTopic := fmt.Sprintf("ratt/control/node/%s/open", app.cfg.ClientID)
	if err := app.mqtt.Subscribe(openTopic); err != nil {
		log.Printf("Subscribe error: %v", err)
	}

	app.indicator.Idle()
	if app.display != nil {
		app.display.Idle()
	}
}

func (app *App) onMQTTDisconnect() {
	app.indicator.ConnectionLost()
	if app.display != nil {
		app.display.ConnectionLost()
	}
}

func (app *App) onMQTTMessage(topic string, payload []byte) {
	switch topic {
	case "ratt/control/broadcast/acl/update":
		fmt.Println("Received ACL update message")
		if err := app.acl.FetchFromAPI(); err != nil {
			log.Printf("Fetch ACL: %v", err)
		}

	default:
		// Check if it's an open command for this node
		openTopic := fmt.Sprintf("ratt/control/node/%s/open", app.cfg.ClientID)
		if topic == openTopic {
			app.handleOpenRequest(payload)
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
	app.openDoor(&indicator.AccessInfo{Member: req.Member, Allowed: true})
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

	// Build event with ACL lookup result
	evt := screen.Event{
		TagID: tagID,
		Found: found,
	}
	if found {
		evt.Member = record.Member
		evt.Nickname = record.Nickname
		evt.Warning = record.Warning
		evt.Allowed = record.Allowed
	}

	// Determine if authorized or denied
	authorized := found && record.Allowed

	if authorized {
		evt.Type = screen.EventAuthorized
	} else {
		evt.Type = screen.EventDenied
	}

	// Send event to current screen - if handled, skip default processing
	if app.display != nil {
		if app.display.SendEvent(evt) {
			return
		}
	}

	// Default handling (access control)
	var info *indicator.AccessInfo
	if found {
		info = &indicator.AccessInfo{
			Member:   record.Member,
			Nickname: record.Nickname,
			Warning:  record.Warning,
			Allowed:  record.Allowed,
		}
	}

	if !authorized {
		if !found {
			fmt.Printf("Tag %d not found in ACL\n", tagID)
		} else {
			fmt.Printf("Tag %d: member=%s denied\n", tagID, record.Member)
		}
		app.indicator.Denied(info)
		if app.display != nil {
			warning := "Unknown Tag"
			if found {
				warning = record.Warning
			}
			app.display.Denied(evt.Member, evt.Nickname, warning)
		}
		time.Sleep(3 * time.Second)
		app.indicator.Idle()
		if app.display != nil {
			app.display.Idle()
		}
		return
	}

	fmt.Printf("Tag %d: member=%s allowed\n", tagID, record.Member)
	app.publishAccess(record.Member, true)
	app.openDoor(info)
}

func (app *App) openDoor(info *indicator.AccessInfo) {
	app.indicator.Opening(info)
	if app.display != nil && info != nil {
		app.display.Opening(info.Member, info.Nickname, info.Warning)
	}

	if err := app.door.Open(); err != nil {
		log.Printf("Door open: %v", err)
	}

	app.indicator.Granted(info)
	if app.display != nil && info != nil {
		app.display.Granted(info.Member, info.Nickname, info.Warning)
	}
	time.Sleep(time.Duration(app.cfg.WaitSecs) * time.Second)

	if err := app.door.Close(); err != nil {
		log.Printf("Door close: %v", err)
	}

	app.indicator.Idle()
	if app.display != nil {
		app.display.Idle()
	}
}

func (app *App) publishAccess(member string, allowed bool) {
	allowedInt := 0
	if allowed {
		allowedInt = 1
	}
	topic := fmt.Sprintf("ratt/status/node/%s/personality/access", app.cfg.ClientID)
	msg := fmt.Sprintf(`{"allowed":%d,"member":"%s"}`, allowedInt, member)
	app.mqtt.Publish(topic, msg)
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
			app.mqtt.Publish(topic, `{"status":"ok"}`)
		}
	}
}

// SendRotaryEvent sends a rotary turn event to the current screen.
func (app *App) SendRotaryEvent(delta int) {
	if app.display == nil {
		return
	}
	app.display.SendEvent(screen.Event{
		Type:  screen.EventRotaryTurn,
		Delta: delta,
	})
}

// SendRotaryPressEvent sends a rotary button press event to the current screen.
func (app *App) SendRotaryPressEvent() {
	if app.display == nil {
		return
	}
	app.display.SendEvent(screen.Event{
		Type: screen.EventRotaryPress,
	})
}

// SendButtonEvent sends a button press event to the current screen.
func (app *App) SendButtonEvent() {
	if app.display == nil {
		return
	}
	app.display.SendEvent(screen.Event{
		Type: screen.EventButton,
	})
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
