package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"goratt/lib/acl"
	"goratt/lib/audio"
	"goratt/lib/config"
	"goratt/lib/door"
	"goratt/lib/eventpipe"
	"goratt/lib/indicator"
	"goratt/lib/mqtt"
	"goratt/lib/reader"
	"goratt/lib/rotary"
	"goratt/lib/video"
	"goratt/lib/video/screen"
	"goratt/lib/video/screen/screens"
)

// Handler defines the hooks an application must implement to customize logic.
type Handler interface {
	HandleTag(tagID uint64, record acl.ACLRecord, found bool)
	HandleExternalEvent(evt screen.Event)
	OnMQTTConnect()
	OnMQTTMessage(topic string, payload []byte)
}

// BaseApp provides the common framework for GoRATT applications.
type BaseApp struct {
	Cfg       *config.Config
  BuildID   string
	MQTT      *mqtt.Client
	Reader    reader.TagReader
	Door      door.DoorOpener
	Indicator indicator.Indicator
	Display   *video.Display
	Rotary    *rotary.Rotary
	EventPipe *eventpipe.EventPipe
	ACL       *acl.ACLManager
	Audio     *audio.AudioManager
	Ctx       context.Context
	Cancel    context.CancelFunc
	Handler   Handler

	// Common Screens
	IdleScreen     *screens.IdleScreen
	ShutdownScreen *screens.ShutdownScreen
}

// NewBaseApp creates and initializes the core application components.
func NewBaseApp(cfg *config.Config, audioParams *audio.Params, handler Handler, Build string) *BaseApp {
	ctx, cancel := context.WithCancel(context.Background())

	app := &BaseApp{
		Cfg:     cfg,
		Ctx:     ctx,
		Cancel:  cancel,
		Handler: handler,
    BuildID: Build,
	}

	var err error

	if audioParams != nil {
		app.Audio = audio.NewAudioManager(*audioParams, cfg.Audio.Device)
	}

	// Initialize MQTT
	app.MQTT, err = mqtt.New(cfg.MQTT, cfg.ClientID, mqtt.Handlers{
		OnConnect:    app.onMQTTConnect,
		OnDisconnect: app.onMQTTDisconnect,
		OnMessage:    app.onMQTTMessage,
	})
	if err != nil {
		log.Fatalf("Init MQTT: %v", err)
	}

	// Initialize indicator
	app.Indicator, err = indicator.New(cfg.Indicator)
	if err != nil {
		log.Fatalf("Init indicator: %v", err)
	}
	app.Indicator.ConnectionLost()

	// Initialize display if enabled
	if cfg.VideoEnabled {
		if !video.ScreenSupported() {
			log.Fatalf("Video enabled but screen support not compiled in")
		}
		app.Display, err = video.New(cfg.Video)
		if err != nil {
			log.Fatalf("Init display: %v", err)
		}

		mgr := app.Display.Manager()
		app.IdleScreen = screens.NewIdleScreen()
		app.ShutdownScreen = screens.NewShutdownScreen()

		mgr.Register(screen.ScreenIdle, app.IdleScreen)
		mgr.Register(screen.ScreenShutdown, app.ShutdownScreen)

		mgr.SwitchTo(screen.ScreenIdle)
		mgr.SetStopAudioFn(app.Audio.Stop)
		mgr.SetPlayAudioFn(app.Audio.PlayPCM)
		mgr.SetPlayAudioBytesFn(app.Audio.PlayBuffer)
	}

	// Initialize rotary encoder
	app.Rotary, err = rotary.New(cfg.Rotary, rotary.Handlers{
		OnTurn:      app.SendRotaryEvent,
		OnPress:     app.SendRotaryPressEvent,
		OnLongPress: app.SendRotaryLongPressEvent,
		OnButtonUp:  app.SendButtonUpEvent,
	})
	if err != nil {
		log.Fatalf("Init rotary: %v", err)
	}

	// Initialize door opener
	app.Door, err = door.New(cfg.Door)
	if err != nil {
		log.Fatalf("Init door: %v", err)
	}

	// Initialize tag reader
	app.Reader, err = reader.New(cfg.Reader)
	if err != nil {
		log.Fatalf("Init reader: %v", err)
	}

	// Initialize ACL manager
	app.ACL = acl.NewACLManager(cfg)
	app.ACL.SetUpdateCallback(func() {
		topic := fmt.Sprintf("ratt/status/node/%s/acl/update", cfg.ClientID)
		app.MQTT.Publish(topic, `{"status":"downloaded"}`)
	})
	if err := app.ACL.LoadFromFile(); err != nil {
		log.Printf("Warning: could not load tag file: %v", err)
	}
	// Initial fetch in background
	go func() {
		if err := app.ACL.FetchFromAPI(); err != nil {
			log.Printf("Warning: could not fetch ACL from API: %v", err)
		}
	}()

	// Initialize event pipe
	app.EventPipe, err = eventpipe.New(cfg.EventPipe, app.handleExternalEvent)
	if err != nil {
		log.Fatalf("Init event pipe: %v", err)
	}

	return app
}

// Run starts background goroutines and waits for termination signal.
func (app *BaseApp) Run() {
	go func() {
		if err := app.MQTT.Connect(); err != nil {
			log.Printf("MQTT connect: %v", err)
		}
	}()
	go app.tagListener()
	go app.pingSender()
	if app.EventPipe != nil {
		go app.EventPipe.Start()
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	fmt.Println("Shutting down...")
	app.Shutdown()
}

func (app *BaseApp) Shutdown() {
	app.Cancel()
	app.MQTT.Disconnect()
	app.Reader.Close()
	app.Door.Release()
	app.Indicator.Shutdown()
	app.Indicator.Release()
	if app.Display != nil {
		app.Display.Manager().SwitchTo(screen.ScreenShutdown)
		app.Display.Release()
	}
	if app.Rotary != nil {
		app.Rotary.Release()
	}
	if app.EventPipe != nil {
		app.EventPipe.Close()
	}
	fmt.Println("Shutdown complete")
}

var firstMessageSent = false

func (app *BaseApp) onMQTTConnect() {
  // Sent startup message
  if (!firstMessageSent) {
    topic := fmt.Sprintf("ratt/status/node/%s/system/boot", app.Cfg.ClientID)
    message := fmt.Sprintf(`{"fw_name":"goratt", "fw_version":"%s"}`, app.BuildID)
    app.MQTT.Publish(topic, message)
    firstMessageSent=true
  }
	app.Indicator.Idle()
	if app.Display != nil {
		app.Display.SetMQTTConnected(true)
	}
	
	// Subscribe to the global broadcast topic for ACL updates
	if err := app.MQTT.Subscribe("ratt/control/broadcast/acl/update"); err != nil {
		log.Printf("Subscribe to broadcast ACL update error: %v", err)
	}

	if app.Handler != nil {
		app.Handler.OnMQTTConnect()
	}
}

func (app *BaseApp) onMQTTDisconnect() {
	app.Indicator.ConnectionLost()
	if app.Display != nil {
		app.Display.SetMQTTConnected(false)
	}
}

func (app *BaseApp) onMQTTMessage(topic string, payload []byte) {
	if app.Display != nil {
		app.Display.SendEvent(screen.Event{
			Type: screen.EventMQTTMessage,
			Data: screen.MQTTData{Topic: topic, Payload: payload},
		})
	}

	// Core ACL update broadcast
	if topic == "ratt/control/broadcast/acl/update" {
		log.Println("Received ACL update message")
		go app.ACL.FetchFromAPI()
	}

	if app.Handler != nil {
		app.Handler.OnMQTTMessage(topic, payload)
	}
}

func (app *BaseApp) tagListener() {
	for {
		select {
		case <-app.Ctx.Done():
			return
		default:
		}

		tagID, err := app.Reader.Read(app.Ctx)
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

		log.Printf("Tag read: %d", tagID)
		record, found := app.ACL.Lookup(tagID)
		if app.Handler != nil {
			app.Handler.HandleTag(tagID, record, found)
		}
	}
}

func (app *BaseApp) pingSender() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-app.Ctx.Done():
			return
		case <-ticker.C:
			topic := fmt.Sprintf("ratt/status/node/%s/ping", app.Cfg.ClientID)
			app.MQTT.Publish(topic, `{"status":"ok"}`)
		}
	}
}

func (app *BaseApp) handleExternalEvent(evt screen.Event) {
	if evt.Type == screen.EventRFID {
		if rfid := evt.RFID(); rfid != nil {
			record, found := app.ACL.Lookup(rfid.TagID)
			if app.Handler != nil {
				app.Handler.HandleTag(rfid.TagID, record, found)
			}
		}
		return
	}

	// Forward interactive events to display
	if app.Display != nil {
		switch evt.Type {
		case screen.EventRotaryTurn, screen.EventRotaryPress, screen.EventPin:
			app.Display.SendEvent(evt)
		}
	}

	if app.Handler != nil {
		app.Handler.HandleExternalEvent(evt)
	}
}

// Convenience event senders

func (app *BaseApp) SendRotaryEvent(delta int) {
	if app.Display != nil {
		app.Display.SendEvent(screen.Event{
			Type: screen.EventRotaryTurn,
			Data: screen.RotaryData{ID: screen.RotaryMain, Delta: delta},
		})
	}
}

func (app *BaseApp) SendRotaryPressEvent() {
	if app.Display != nil {
		app.Display.SendEvent(screen.Event{
			Type: screen.EventRotaryPress,
			Data: screen.RotaryData{ID: screen.RotaryMain},
		})
	}
}

func (app *BaseApp) SendRotaryLongPressEvent() {
	if app.Display != nil {
		app.Display.SendEvent(screen.Event{
			Type: screen.EventRotaryLongPress,
			Data: screen.RotaryData{ID: screen.RotaryMain},
		})
	}
}

func (app *BaseApp) SendButtonUpEvent() {
	if app.Display != nil {
		app.Display.SendEvent(screen.Event{
			Type: screen.EventButtonUp,
		})
	}
}

func (app *BaseApp) SendPinEvent(pinID screen.PinID, pressed bool) {
	if app.Display != nil {
		app.Display.SendEvent(screen.Event{
			Type: screen.EventPin,
			Data: screen.PinData{ID: pinID, Pressed: pressed},
		})
	}
}

func (app *BaseApp) OpenDoor(info *indicator.AccessInfo, waitSecs int, openingScreen, grantedScreen screen.ScreenID) {
	app.Indicator.Opening(info)
	if app.Display != nil && openingScreen != 0 {
		app.Display.Manager().SwitchTo(openingScreen)
	}

	if err := app.Door.Open(); err != nil {
		log.Printf("Door open: %v", err)
	}

	app.Indicator.Granted(info)
	if app.Display != nil && grantedScreen != 0 {
		app.Display.Manager().SwitchTo(grantedScreen)
	}

	time.Sleep(time.Duration(waitSecs) * time.Second)

	if err := app.Door.Close(); err != nil {
		log.Printf("Door close: %v", err)
	}
	app.Indicator.Idle()
}

func (app *BaseApp) PublishAccess(member string, allowed bool) {
	allowedInt := 0
	if allowed {
		allowedInt = 1
	}
	topic := fmt.Sprintf("ratt/status/node/%s/personality/access", app.Cfg.ClientID)
	msg := fmt.Sprintf(`{"allowed":%d,"member":"%s"}`, allowedInt, member)
	app.MQTT.Publish(topic, msg)
}
