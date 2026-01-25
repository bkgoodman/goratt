package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v2"

	"goratt/lib/acl"
	"goratt/lib/app"
	"goratt/lib/auth"
	"goratt/lib/config"
	"goratt/lib/indicator"
	"goratt/lib/video/screen"

	doorlockscreens "goratt/cmd/doorlock/screens"
)

var myBuild string

type DoorlockApp struct {
	Base          *app.BaseApp
	grantedScreen *doorlockscreens.DoorlockGrantedScreen
	deniedScreen  *doorlockscreens.DoorlockDeniedScreen
	openingScreen *doorlockscreens.DoorlockOpeningScreen
}

type OpenRequest struct {
	Member    string `json:"member"`
	ToolName  string `json:"tool"`
	Timestamp uint64 `json:"timestamp"`
	Signature string `json:"signature"`
}

func main() {
	fmt.Printf("GoRATT Doorlock build %s\n", myBuild)

	openflag := flag.Bool("holdopen", false, "Hold door open indefinitely")
	cfgfile := flag.String("cfg", "goratt.cfg", "Config file")
	flag.Parse()

	f, err := os.Open(*cfgfile)
	if err != nil {
		log.Fatalf("Open config: %v", err)
	}
	defer f.Close()

	var cfg config.Config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		log.Fatalf("Decode config: %v", err)
	}

	doorlockApp := &DoorlockApp{}
	doorlockApp.Base = app.NewBaseApp(&cfg, doorlockApp)

	if doorlockApp.Base.Display != nil {
		mgr := doorlockApp.Base.Display.Manager()
		doorlockApp.Base.IdleScreen.SetBuildID(myBuild)

		// Local custom screens
		idle := doorlockscreens.NewDoorlockIdleScreen()
		doorlockApp.grantedScreen = doorlockscreens.NewDoorlockGrantedScreen()
		doorlockApp.deniedScreen = doorlockscreens.NewDoorlockDeniedScreen()
		doorlockApp.openingScreen = doorlockscreens.NewDoorlockOpeningScreen()

		mgr.Register(screen.ScreenIdle, idle)
		mgr.Register(screen.ScreenGranted, doorlockApp.grantedScreen)
		mgr.Register(screen.ScreenDenied, doorlockApp.deniedScreen)
		mgr.Register(screen.ScreenOpening, doorlockApp.openingScreen)
	}

	if *openflag {
		doorlockApp.Base.OpenDoor(&indicator.AccessInfo{Member: "holdopen"}, cfg.WaitSecs, screen.ScreenOpening, screen.ScreenGranted)
		select {} // Block forever
	}

	doorlockApp.Base.Run()
}

func (app *DoorlockApp) OnMQTTConnect() {
	// Subscribe to node-specific open command
	openTopic := fmt.Sprintf("ratt/control/node/%s/open", app.Base.Cfg.ClientID)
	if err := app.Base.MQTT.Subscribe(openTopic); err != nil {
		log.Printf("Subscribe error: %v", err)
	}
}

func (app *DoorlockApp) OnMQTTMessage(topic string, payload []byte) {
	openTopic := fmt.Sprintf("ratt/control/node/%s/open", app.Base.Cfg.ClientID)
	if topic == openTopic {
		app.handleOpenRequest(payload)
	}
}

func (app *DoorlockApp) handleOpenRequest(payload []byte) {
	if app.Base.Cfg.OpenSecret == "" || app.Base.Cfg.OpenToolName == "" {
		log.Println("Remote open disabled (no secret or tool name configured)")
		return
	}

	var req OpenRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		log.Printf("Decode open request: %v", err)
		return
	}

	if err := auth.VerifySignature(app.Base.Cfg.OpenSecret, req.Member, req.ToolName, req.Timestamp, req.Signature); err != nil {
		log.Printf("Signature verification failed: %v", err)
		return
	}

	if req.ToolName != app.Base.Cfg.OpenToolName {
		log.Printf("Wrong tool name %q, expected %q", req.ToolName, app.Base.Cfg.OpenToolName)
		return
	}

	// Check timestamp is within 5 minute window
	ts := time.Unix(int64(req.Timestamp), 0)
	if time.Since(ts) > 5*time.Minute || time.Until(ts) > 5*time.Minute {
		log.Println("Open request timestamp out of range")
		return
	}

	log.Printf("Remote open request from %s", req.Member)
	app.Base.PublishAccess(req.Member, true)
	app.Base.OpenDoor(&indicator.AccessInfo{Member: req.Member, Allowed: true}, app.Base.Cfg.WaitSecs, screen.ScreenOpening, screen.ScreenGranted)
}

func (app *DoorlockApp) HandleTag(tagID uint64, record acl.ACLRecord, found bool) {
	rfidData := screen.RFIDData{
		TagID:    tagID,
		Found:    found,
		Member:   record.Member,
		Nickname: record.Nickname,
		Warning:  record.Warning,
		Allowed:  record.Allowed,
	}

	authorized := found && record.Allowed
	var evt screen.Event
	if authorized {
		evt = screen.Event{Type: screen.EventAuthorized, Data: rfidData}
	} else {
		evt = screen.Event{Type: screen.EventDenied, Data: rfidData}
	}

	// Send event to current screen - if handled, skip default processing
	if app.Base.Display != nil {
		if app.Base.Display.SendEvent(evt) {
			return
		}
	}

	if !authorized {
		if !found {
			log.Printf("Tag %d not found in ACL", tagID)
		} else {
			log.Printf("Tag %d: member=%s denied", tagID, record.Member)
		}
		app.Base.Indicator.Denied(&indicator.AccessInfo{
			Member:   record.Member,
			Nickname: record.Nickname,
			Warning:  record.Warning,
			Allowed:  record.Allowed,
		})

		if app.Base.Display != nil {
			warning := "Unknown Tag"
			if found {
				warning = record.Warning
			}
			app.deniedScreen.SetInfo(rfidData.Member, rfidData.Nickname, warning)
			app.Base.Display.Manager().SwitchTo(screen.ScreenDenied)
		} else {
			go func() {
				time.Sleep(3 * time.Second)
				app.Base.Indicator.Idle()
			}()
		}
		return
	}

	log.Printf("Tag %d: member=%s allowed", tagID, record.Member)

	// Basic door opener behavior
	app.Base.PublishAccess(record.Member, true)
	app.Base.OpenDoor(&indicator.AccessInfo{
		Member:   record.Member,
		Nickname: record.Nickname,
		Warning:  record.Warning,
		Allowed:  record.Allowed,
	}, app.Base.Cfg.WaitSecs, screen.ScreenOpening, screen.ScreenGranted)
}

func (app *DoorlockApp) HandleExternalEvent(evt screen.Event) {
}
