package main
 
import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
 
	"gopkg.in/yaml.v2"
 
	darkroomscreens "goratt/cmd/darkroom/screens"
	"goratt/lib/acl"
	"goratt/lib/app"
	"goratt/lib/auth"
	"goratt/lib/config"
	"goratt/lib/indicator"
	"goratt/lib/video/screen"
)
 
var myBuild string
 
type CalEntry struct {
	SUMMARY   string `json:"SUMMARY"`
	START     string `json:"START"`
	END       string `json:"END"`
	ORGANIZER string `json:"ORGANIZER"`
	CODE      int64  `json:"CODE"`
	DOW       string `json:"DOW"`
	DEVICE    string `json:"DEVICE"`
	TIME      string `json:"TIME"`
	WHEN      string `json:"WHEN"`
}
 
type DarkroomApp struct {
	Base            *app.BaseApp
	openingScreen   *darkroomscreens.DarkroomOpeningScreen
	roomInUseScreen *darkroomscreens.RoomInUseScreen
	safeLightScreen *darkroomscreens.SafeLightScreen
	deniedScreen    *darkroomscreens.DarkroomDeniedScreen
	grantedScreen   *darkroomscreens.DarkroomGrantedScreen
	idleScreen      *darkroomscreens.DarkroomIdleScreen
 
	inUse      bool
	lastMember string
 
	nextCalEntry *CalEntry
	nextCalFetch time.Time
}
 
type OpenRequest struct {
	Member    string `json:"member"`
	ToolName  string `json:"tool"`
	Timestamp uint64 `json:"timestamp"`
	Signature string `json:"signature"`
}
 
func main() {
	fmt.Printf("GoRATT Darkroom build %s\n", myBuild)
 
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
 
	darkroomApp := &DarkroomApp{}
	darkroomApp.Base = app.NewBaseApp(&cfg, darkroomApp)
 
	if darkroomApp.Base.Display != nil {
		mgr := darkroomApp.Base.Display.Manager()
		darkroomApp.Base.IdleScreen.SetBuildID(myBuild)
 
		// Local custom screens
		darkroomApp.idleScreen = darkroomscreens.NewDarkroomIdleScreen()
		darkroomApp.grantedScreen = darkroomscreens.NewDarkroomGrantedScreen()
		darkroomApp.deniedScreen = darkroomscreens.NewDarkroomDeniedScreen()
		darkroomApp.openingScreen = darkroomscreens.NewDarkroomOpeningScreen()
		darkroomApp.roomInUseScreen = darkroomscreens.NewRoomInUseScreen()
		darkroomApp.safeLightScreen = darkroomscreens.NewSafeLightScreen()
 
		mgr.Register(screen.ScreenIdle, darkroomApp.idleScreen)
		mgr.Register(screen.ScreenGranted, darkroomApp.grantedScreen)
		mgr.Register(screen.ScreenDenied, darkroomApp.deniedScreen)
		mgr.Register(screen.ScreenOpening, darkroomApp.openingScreen)
		mgr.Register(screen.ScreenRoomInUse, darkroomApp.roomInUseScreen)
		mgr.Register(screen.ScreenSafeLight, darkroomApp.safeLightScreen)
	}
 
	// Start calendar fetch loop
	darkroomApp.fetchCalendar()
	go darkroomApp.calendarLoop()
 
	if *openflag {
		darkroomApp.Base.OpenDoor(&indicator.AccessInfo{Member: "holdopen"}, cfg.WaitSecs, screen.ScreenOpening, screen.ScreenGranted)
		select {} // Block forever
	}
 
	darkroomApp.Base.Run()
}
 
func (app *DarkroomApp) calendarLoop() {
	ticker := time.NewTicker(15 * time.Minute)
	for range ticker.C {
		app.fetchCalendar()
	}
}
 
func (app *DarkroomApp) fetchCalendar() {
	if app.Base.Cfg.CalendarURL == "" {
		return
	}
 
	log.Println("Fetching calendar...")
	resp, err := http.Get(app.Base.Cfg.CalendarURL)
	if err != nil {
		log.Printf("Failed to fetch calendar: %v", err)
		return
	}
	defer resp.Body.Close()
 
	var cal []CalEntry
	if err := json.NewDecoder(resp.Body).Decode(&cal); err != nil {
		log.Printf("Failed to decode calendar: %v", err)
		return
	}
 
	if len(cal) > 0 {
		app.nextCalEntry = &cal[0]
		log.Printf("Next reservation: %s by %s at %s", cal[0].SUMMARY, cal[0].ORGANIZER, cal[0].WHEN)
		if app.idleScreen != nil {
			app.idleScreen.SetNextReservation(cal[0].SUMMARY, cal[0].ORGANIZER, cal[0].WHEN)
		}
	} else {
		app.nextCalEntry = nil
		if app.idleScreen != nil {
			app.idleScreen.SetNextReservation("", "", "")
		}
	}
}
 
func (app *DarkroomApp) OnMQTTConnect() {
	openTopic := fmt.Sprintf("ratt/control/node/%s/open", app.Base.Cfg.ClientID)
	if err := app.Base.MQTT.Subscribe(openTopic); err != nil {
		log.Printf("Subscribe error: %v", err)
	}
}
 
func (app *DarkroomApp) OnMQTTMessage(topic string, payload []byte) {
	openTopic := fmt.Sprintf("ratt/control/node/%s/open", app.Base.Cfg.ClientID)
	if topic == openTopic {
		app.handleOpenRequest(payload)
	}
}
 
func (app *DarkroomApp) handleOpenRequest(payload []byte) {
	if app.Base.Cfg.OpenSecret == "" || app.Base.Cfg.OpenToolName == "" {
		log.Println("Remote open disabled")
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
 
	ts := time.Unix(int64(req.Timestamp), 0)
	if time.Since(ts) > 5*time.Minute || time.Until(ts) > 5*time.Minute {
		log.Println("Open request timestamp out of range")
		return
	}
 
	log.Printf("Remote open request from %s", req.Member)
	app.Base.PublishAccess(req.Member, true)
	app.Base.OpenDoor(&indicator.AccessInfo{Member: req.Member, Allowed: true}, app.Base.Cfg.WaitSecs, screen.ScreenOpening, screen.ScreenGranted)
}
 
func (app *DarkroomApp) HandleTag(tagID uint64, record acl.ACLRecord, found bool) {
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
 
	if app.Base.Display != nil {
		if app.Base.Display.SendEvent(evt) {
			return
		}
	}
 
	if !authorized {
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
		}
		return
	}
 
	log.Printf("Tag %d: member=%s allowed in darkroom", tagID, record.Member)
 
	app.Base.PublishAccess(record.Member, true)
 
	if app.inUse && (app.lastMember == record.Member || strings.ReplaceAll(record.Member, ".", " ") == app.lastMember) {
		// Badge out
		app.inUse = false
		app.lastMember = ""
		if app.Base.Display != nil {
			app.Base.Display.Manager().SwitchTo(screen.ScreenIdle)
		}
		return
	}
 
	// Badge in
	app.inUse = true
	app.lastMember = record.Member
 
	go func() {
		app.Base.OpenDoor(&indicator.AccessInfo{
			Member:   record.Member,
			Nickname: record.Nickname,
			Warning:  record.Warning,
			Allowed:  record.Allowed,
		}, app.Base.Cfg.WaitSecs, screen.ScreenOpening, screen.ScreenGranted)
 
		// After door closes, switch to Room In Use
		if app.Base.Display != nil && app.inUse {
			app.roomInUseScreen.SetInfo(record.Member, record.Nickname)
			app.Base.Display.Manager().SwitchTo(screen.ScreenRoomInUse)
		}
	}()
}
 
func (app *DarkroomApp) HandleExternalEvent(evt screen.Event) {
	// Custom logic for physical safe-light switch could go here
	if evt.Type == screen.EventPin {
		if pin := evt.Pin(); pin != nil {
			if pin.ID == screen.PinSafelight {
				if pin.Pressed {
					log.Println("Safelight sensor active!")
					if app.Base.Display != nil {
						app.Base.Display.Manager().SwitchTo(screen.ScreenSafeLight)
					}
				} else {
					log.Println("Safelight sensor inactive.")
					if app.Base.Display != nil {
						app.Base.Display.Manager().SwitchTo(screen.ScreenIdle)
					}
				}
			}
		}
	}
}
