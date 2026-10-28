package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v2"

	"goratt/lib/acl"
	"goratt/lib/app"
	"goratt/lib/indicator"
	"goratt/lib/video/screen"
)

var myBuild string

type LabelPrinterApp struct {
	Base       *app.BaseApp
	Cfg        *AppConfig
	Printer    *Printer
	ControlPad *ControlPad
	HTTPServer *HTTPServer
}

func main() {
	fmt.Printf("GoRATT Label Printer build %s\n", myBuild)

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
	cfg.LabelPrinter.SetDefaults()

	printer, err := NewPrinter(cfg.LabelPrinter.Device)
	if err != nil {
		log.Fatalf("Init printer: %v", err)
	}

	labelApp := &LabelPrinterApp{
		Cfg:     &cfg,
		Printer: printer,
	}

	// Initialize GoRATT shared framework (MQTT, ACL, RFID reader, Indicator)
	labelApp.Base = app.NewBaseApp(&cfg.Config, nil, labelApp, myBuild)

	// Set up personality-specific MQTT publisher
	publishPersonality := func(personality, payload string) {
		topic := fmt.Sprintf("ratt/status/node/%s/personality/%s", labelApp.Base.Cfg.ClientID, personality)
		labelApp.Base.MQTT.Publish(topic, payload)
	}

	// Start button/LED controller
	labelApp.ControlPad = NewControlPad(cfg.LabelPrinter, printer, publishPersonality)
	labelApp.ControlPad.Start()
	defer labelApp.ControlPad.Stop()

	// Start HTTP server for backend ArUco label print requests
	labelApp.HTTPServer = StartHTTPServer(cfg.LabelPrinter.HTTPPort, printer)
	defer labelApp.HTTPServer.Shutdown()

	if cfg.LabelPrinter.PrintOnBoot != nil && *cfg.LabelPrinter.PrintOnBoot {
		log.Println("PrintOnBoot is enabled; printing ready label")
		go func() {
			if err := printer.PrintStoragePass("- Ready -"); err != nil {
				log.Printf("PrintOnBoot error: %v", err)
			}
		}()
	}

	log.Printf("Label printer running (Printer: %s, HTTP: :%d)", cfg.LabelPrinter.Device, cfg.LabelPrinter.HTTPPort)
	labelApp.Base.Run()
}

// HandleTag processes RFID badge swipes from the reader.
func (a *LabelPrinterApp) HandleTag(tagID uint64, record acl.ACLRecord, found bool) {
	if !found || !record.Allowed {
		if !found {
			log.Printf("Tag %d not found in ACL", tagID)
		} else {
			log.Printf("Tag %d: member=%s denied", tagID, record.Member)
		}
		a.Base.Indicator.Denied(&indicator.AccessInfo{
			Member:   record.Member,
			Nickname: record.Nickname,
			Warning:  record.Warning,
			Allowed:  record.Allowed,
		})
		return
	}

	log.Printf("Tag %d: member=%s allowed", tagID, record.Member)
	a.Base.PublishAccess(record.Member, true)
	a.Base.Indicator.Granted(&indicator.AccessInfo{
		Member:   record.Member,
		Nickname: record.Nickname,
		Warning:  record.Warning,
		Allowed:  record.Allowed,
	})

	// Notify control pad of authorized badge scan
	a.ControlPad.HandleBadgeScan(record.Member)
}

func (a *LabelPrinterApp) HandleExternalEvent(evt screen.Event) {
}

func (a *LabelPrinterApp) OnMQTTConnect() {
}

func (a *LabelPrinterApp) OnMQTTMessage(topic string, payload []byte) {
}
