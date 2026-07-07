package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v2"

	vendingscreens "goratt/cmd/vending/screens"
	"goratt/cmd/vending/vending"
	"goratt/lib/acl"
	"goratt/lib/app"
	"goratt/lib/audio"
	"goratt/lib/auth"
	"goratt/lib/indicator"
	"goratt/lib/video/screen"
)

var myBuild string

type VendingApp struct {
	Base                  *app.BaseApp
	vendingClient         *vending.Client
	currentVendingSession *VendingSessionState
	// Vending specific screens
	deniedScreen            *vendingscreens.VendingDeniedScreen
	selectAmountScreen      *vendingscreens.SelectAmountScreen
	confirmScreen           *vendingscreens.ConfirmScreen
	abortedScreen           *vendingscreens.AbortedScreen
	insufficientFundsScreen *vendingscreens.InsufficientFundsScreen
	processingScreen        *vendingscreens.ProcessingScreen
	successScreen           *vendingscreens.SuccessScreen
	paymentFailedScreen     *vendingscreens.PaymentFailedScreen
}

type OpenRequest struct {
	Member    string `json:"member"`
	ToolName  string `json:"tool"`
	Timestamp uint64 `json:"timestamp"`
	Signature string `json:"signature"`
}

func main() {
	fmt.Printf("GoRATT Vending build %s\n", myBuild)

	openflag := flag.Bool("holdopen", false, "Hold door open indefinitely")
	cfgfile := flag.String("cfg", "goratt.cfg", "Config file")
	flag.Parse()

	f, err := os.Open(*cfgfile)
	if err != nil {
		log.Fatalf("Open config: %v", err)
	}
	defer f.Close()

	var cfg AppConfig
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		log.Fatalf("Decode config: %v", err)
	}

	vendingApp := &VendingApp{}
	audioParams := audio.Params{
		Format:   "S16_LE",
		Rate:     16000,
		Type:     "raw",
		Channels: 1,
	}
	vendingApp.Base = app.NewBaseApp(&cfg.Config, &audioParams, vendingApp, myBuild)
	if vendingApp.Base.IdleScreen != nil {
		vendingApp.Base.IdleScreen.SetBuildID(myBuild)
	}

	if vendingApp.Base.Display != nil {
		mgr := vendingApp.Base.Display.Manager()

		// Local custom screens
		idle := vendingscreens.NewVendingIdleScreen()
		idle.SetBuildID(myBuild)
		vendingApp.deniedScreen = vendingscreens.NewVendingDeniedScreen()
		vendingApp.selectAmountScreen = vendingscreens.NewSelectAmountScreen()
		vendingApp.confirmScreen = vendingscreens.NewConfirmScreen()
		vendingApp.abortedScreen = vendingscreens.NewAbortedScreen()
		vendingApp.insufficientFundsScreen = vendingscreens.NewInsufficientFundsScreen()
		vendingApp.processingScreen = vendingscreens.NewProcessingScreen()
		vendingApp.successScreen = vendingscreens.NewSuccessScreen()
		vendingApp.paymentFailedScreen = vendingscreens.NewPaymentFailedScreen()

		mgr.Register(screen.ScreenIdle, idle)
		mgr.Register(screen.ScreenDenied, vendingApp.deniedScreen)
		mgr.Register(screen.ScreenSelectAmount, vendingApp.selectAmountScreen)
		mgr.Register(screen.ScreenConfirm, vendingApp.confirmScreen)
		mgr.Register(screen.ScreenAborted, vendingApp.abortedScreen)
		mgr.Register(screen.ScreenInsufficientFunds, vendingApp.insufficientFundsScreen)
		mgr.Register(screen.ScreenProcessing, vendingApp.processingScreen)
		mgr.Register(screen.ScreenSuccess, vendingApp.successScreen)
		mgr.Register(screen.ScreenPaymentFailed, vendingApp.paymentFailedScreen)

		// Ensure our custom idle screen replaces the default one
		mgr.SwitchTo(screen.ScreenIdle)
	}

	// Initialize vending API client
	var mockMode bool
	if cfg.API.URL != "" {
		product := cfg.Vending.Product
		vendingApp.vendingClient = vending.NewClient(cfg.API.URL, cfg.API.Username, cfg.API.Password, product)
		log.Printf("Vending API client initialized: %s (product: %s)", cfg.API.URL, product)
		vending.SetGlobalProcessor(vendingApp)
		mockMode = false
	} else {
		log.Printf("Warning: no API URL configured, using mock mode")
		vending.SetGlobalProcessor(&vending.MockProcessor{ShouldFail: false})
		mockMode = true
	}

	if vendingApp.Base.Display != nil {
		vendingApp.Base.Display.Manager().SetMockMode(mockMode)
	}

	if *openflag {
		vendingApp.Base.OpenDoor(&indicator.AccessInfo{Member: "holdopen"}, cfg.WaitSecs, 0, 0)
		select {} // Block forever
	}

	vendingApp.Base.Run()
}

func (app *VendingApp) OnMQTTConnect() {
	// Subscribe to node-specific open command
	openTopic := fmt.Sprintf("ratt/control/node/%s/open", app.Base.Cfg.ClientID)
	if err := app.Base.MQTT.Subscribe(openTopic); err != nil {
		log.Printf("Subscribe error: %v", err)
	}
}

func (app *VendingApp) OnMQTTMessage(topic string, payload []byte) {
	openTopic := fmt.Sprintf("ratt/control/node/%s/open", app.Base.Cfg.ClientID)
	if topic == openTopic {
		app.handleOpenRequest(payload)
	}
}

func (app *VendingApp) handleOpenRequest(payload []byte) {
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
	app.Base.OpenDoor(&indicator.AccessInfo{Member: req.Member, Allowed: true}, app.Base.Cfg.WaitSecs, 0, 0)
}

func (app *VendingApp) HandleTag(tagID uint64, record acl.ACLRecord, found bool) {
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

		// Ignore tag if we are not in the idle screen
		current := app.Base.Display.Manager().Current()
		if current != nil && current.Name() != "VendingIdle" {
			log.Printf("Tag read ignored (not in idle screen)")
			return
		}
	}

	if !authorized {
		if !found {
			log.Printf("Tag %d not found in ACL", tagID)
			if app.vendingClient != nil {
				go app.vendingClient.ReportBadTag(tagID)
			}
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

	// Start vending session instead of opening door
	if app.Base.Display != nil {
		balance := 0.0
		lastLog := 0
		if app.vendingClient != nil {
			if resp, err := app.vendingClient.QueryBalance(record.Member); err == nil {
				balance = resp.Balance
				lastLog = resp.LastLog
				log.Printf("Queried balance for %s: $%.2f (lastLog: %d)", record.Member, balance, lastLog)
			} else {
				log.Printf("Failed to query balance for %s: %v", record.Member, err)
			}
		}

		app.Base.Display.Manager().SetVendingSession(record.Member, record.Nickname, 1.00)
		app.Base.Display.Manager().SetVendingAddAmount(0)
		app.Base.Display.Manager().SetVendingBalance(balance)
		app.startVendingSession(record.Member, record.Nickname, balance, lastLog)
		app.Base.Display.Manager().SwitchTo(screen.ScreenSelectAmount)
	} else {
		app.Base.PublishAccess(record.Member, true)
		app.Base.OpenDoor(&indicator.AccessInfo{
			Member:   record.Member,
			Nickname: record.Nickname,
			Warning:  record.Warning,
			Allowed:  record.Allowed,
		}, app.Base.Cfg.WaitSecs, 0, 0)
	}
}

func (app *VendingApp) HandleExternalEvent(evt screen.Event) {
}
