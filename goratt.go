package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"os"
    "os/exec"
	"strconv"
	"bufio"
    "strings"
	"os/signal"
    "io"
	"crypto/x509"
	"io/ioutil"
    "syscall"
	"sync"
	"gopkg.in/yaml.v2"
	"flag"
	"encoding/json"
	// "github.com/warthog618/gpiod/device/rpi"
   // "github.com/hjkoskel/govattu"
	"time"
  	// "strings" 

	"github.com/eclipse/paho.mqtt.golang"

	"encoding/base64"
	"net/http"

)


var client mqtt.Client
var alertMessage string
var watchdog time.Time


const (
        Button_Knob = iota
        Button_Cancel
)

const (
        Event_Update = iota
        Event_Alert
        Event_Encoderknob
        Event_NFC
        Event_Button
        Event_Timer
        Event_Callback
)

var occupiedBy *string

var lastEventTime time.Time
var safelight = false

type UIEvent struct {
        Event int
        Name string
        Param int    // Button Number
        Data *any    // Usually used for Timer Callback Data
        Callback *any    // Usually used for Timer Callback Function
}

var uiEvent chan UIEvent

var currentScreen Screen

type RattConfig struct {
   CACert string `yaml:"CACert"`
   ClientCert string `yaml:"ClientCert"`
   ClientKey string `yaml:"ClientKey"`
   ClientID string `yaml:"ClientID"`
   MqttHost string `yaml:"MqttHost"`
   MqttPort int `yaml:"MqttPort"`
   ApiURL string `yaml:"ApiURL"`
   ApiCAFile string `yaml:"ApiCAFile"`
   ApiUsername string `yaml:"ApiUsername"`
   ApiPassword string `yaml:"ApiPassword"`
   Resource string `yaml:"Resource"`
   Mode string `yaml:"Mode"`
   CalendarURL string `yaml:"CalendarURL"`

   TagFile string `yaml:"TagFile"`
   ServoClose int `yaml:"ServoClose"`
   ServoOpen int `yaml:"ServoOpen"`
   WaitSecs int `yaml:"WaitSecs"`

   NFCdevice string `yaml:"NFCdevice"`
   NFCmode string `yaml:"NFCmode"`

   DoorPin *int `yaml:"DoorPin"`
   LEDpipe string `yaml:"LEDpipe"`
}

// In-memory ACL list
type ACLlist struct {
	Tag uint64
	Level  int
	Member string
	Allowed bool
}

var validTags []ACLlist
 
var LEDfile *os.File
var LEDidleString string
var cfg RattConfig

const (
        LEDconnectionLost = "@2 !150000 001010"
        LEDnormalIdle = "@3 !150000 400000"
        LEDaccessGranted = "@1 !50000 8000"
        LEDaccessDenied = "@2 !10000 ff"
        LEDterminated = "@0 010101"

)
// From API - off the wire
type ACLentry struct {
	 Tagid string `json:"tagid"`
	 Tag_ident string `json:"tag_ident"`
	 Allowed string `json:"allowed"`
	 Warning string `json:"warning"`
	 Member string `json:"member"`
	 Nickname string `json:"nickname"`
	 Plan string `json:"plan"`
	 Last_accessed string `json:"last_accessed"`
	 Level int `json:"level"`
	 Raw_tag_id  string `json:"raw_tag_id"`
}


var aclfileMutex sync.Mutex

func LEDupdateIdleString (str string) {
  LEDidleString = str
}

func LEDwriteString (str string) {
  if (LEDfile != nil) {
    LEDfile.Write([]byte(str))
  }
}

func GetACLList() {
	// Lock the mutex before entering the critical section
	aclfileMutex.Lock()
	defer aclfileMutex.Unlock()

		// Create a custom transport with your CA certificate
	caCert, err := ioutil.ReadFile(cfg.ApiCAFile)
	if err != nil {
		fmt.Println("Error reading CA certificate: ", err)
		return
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: caCertPool,
		},
	}

	// Create an HTTP client with the custom transport
	httpClient := &http.Client{Transport: transport}

	// Specify the URL you want to make a request to
	url := fmt.Sprintf("%s/api/v1/resources/%s/acl",cfg.ApiURL,cfg.Resource)

	// Create a new GET request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Error creating request: ", err)
		return
	}

	// Add custom credentials to the request header
	auth := base64.StdEncoding.EncodeToString([]byte(cfg.ApiUsername + ":" + cfg.ApiPassword))
	req.Header.Add("Authorization", "Basic "+auth)

	// Make the request
	response, err := httpClient.Do(req)
	if err != nil {
		fmt.Println("Error making request: ", err)
		return
	}
	defer response.Body.Close()

	// Process the response
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println("Error reading response body: ", err)
		return
	}

	//fmt.Printf("Response:\n%s\n", body)
		// Unmarshal JSON array into a slice of structs
	var items []ACLentry
	err = json.Unmarshal([]byte(body), &items)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		return
	}

	// Open temporary version of tagfile to write
	file, err := os.Create(cfg.TagFile+".tmp")

	validTags = validTags[:0]
	// Print the desired field (e.g., "name") from each dictionary
	for index, item := range items {
		_ = index
		//fmt.Println("ID:", index,item.Tagid,item.Raw_tag_id)

			number, err := strconv.ParseUint(item.Raw_tag_id,10,64)
			if err == nil {
				validTags = append(validTags, ACLlist{
					Tag: number,
					Level: item.Level,
					Member: item.Member,
					Allowed: (item.Allowed == "allowed"),
				})
			}
			access := "denied"
			if item.Allowed == "allowed" {
				access = "allowed"
			} 
			_, err = file.WriteString(fmt.Sprintf("%d %s %d %s\n",number,access,item.Level,item.Member))
			if err != nil {
			    fmt.Println("Error writing to tag file: ", err)
			    file.Close()
			    return
			}
	}

	file.Close()

	// Rename or move the file
	err = os.Rename(cfg.TagFile+".tmp", cfg.TagFile)
	if err != nil {
		fmt.Println("Error moving tag file :", err)
		return
	}


	// Signal we were updated

	var topic string = fmt.Sprintf("ratt/status/node/%s/acl/update",cfg.ClientID)
	var message string = "{\"status\":\"downloaded\"}"
	client.Publish(topic,0,false,message)

}

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

var nextCalEntry *CalEntry
var nextCalFetch time.Time


func ReadTagFile() {
	aclfileMutex.Lock()
	defer aclfileMutex.Unlock()

	file,err := os.Open(cfg.TagFile)
	if (err != nil) {
		log.Fatal("Error Reading Tag File: ",err)
		return
	}
	defer file.Close()

	// Create a bufio.Scanner to read lines from the file
	scanner := bufio.NewScanner(file)

	// Loop through each line
	var tag uint64
	var level int
	var member string
	var access string

	validTags = validTags[:0]
	for scanner.Scan() {
		line := scanner.Text()
		_, err := fmt.Sscanf(line, "%d %s %d %s",&tag,&access,&level,&member)
		if err == nil {
				validTags = append(validTags, ACLlist{
					Tag: tag,
					Level: level,
					Member: member,
					Allowed: (access == "allowed"),
			})
		}
	}

}

func watchdogHandler() {
    petWatchdog()
    for {
        fmt.Println("WatchdogCheck")
        if time.Now().After(watchdog) {
            log.Println("WATCHOG EXPIRED -- REBOOTING!!!")
            c := exec.Command("/usr/sbin/reboot")
            c.Run()
        }
        time.Sleep(time.Minute)
    }
}

func petWatchdog() {
    watchdog = time.Now().Add(3 * time.Hour)
}

func PingSender() {

	for {
		var topic string = fmt.Sprintf("ratt/status/node/%s/ping",cfg.ClientID)
		var message string = "{\"status\":\"ok\"}"
		client.Publish(topic,0,false,message)
		time.Sleep(120 * time.Second)
	}
}


func SafelightOn() {
   safelight = true
   lastEventTime = time.Now()
   uiEvent <- UIEvent { Event: Event_Update, }
}

func SafelightOff() {
   safelight = false
   lastEventTime = time.Now()
   
   uiEvent <- UIEvent { Event: Event_Update, }
}

func UpdateUI() {
        // TODO check Safelight state!
}

// This is used mainly for debugging.
// Allows tags to be written to a named pipe
func NamedPipeListener() {
	pipeName := "/tmp/nfctag"

	// Create the named pipe (FIFO) if it doesn't exist.
	// 0666 is the permission mode for the pipe.
	err := syscall.Mkfifo(pipeName, 0666)
	if err != nil && !os.IsExist(err) {
		fmt.Fprintf(os.Stderr, "Error creating named pipe: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Listening for events on named pipe: %s\n", pipeName)

	for {
		// Open the named pipe for reading.
		// This call will block until a writer opens the pipe.
		file, err := os.OpenFile(pipeName, os.O_RDONLY, 0666)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening named pipe: %v\n", err)
			continue
		}

		// Create a new reader to read the lines from the pipe.
		reader := bufio.NewReader(file)

		for {
			// Read a line from the pipe.
			line, err := reader.ReadString('\n')
			if err != nil {
				// If the error is EOF, it means the writer has closed the pipe.
				// We break the inner loop and the outer loop will re-open the pipe,
				// ready for the next writer.
				if err == io.EOF {
					break
				}

				// Handle other potential errors.
				fmt.Fprintf(os.Stderr, "Error reading from pipe: %v\n", err)
				break
			}

			// Process the received event.
			// The `line` includes the newline character, so we trim it.
            line = strings.TrimSuffix(line, "\n")
			fmt.Println( "Pipe Received event: " + line)
            switch (line) {
            case "safe":
                   fmt.Println("Pipe got SAFE") 
                   SafelightOn()
            case "nosafe":
                   fmt.Println("Pipe got NOSAFE") 
                   SafelightOff()
            case "signout":
                   fmt.Println("Pipe got SIGNOUT") 
                   Signout()
            default:
               number, err := strconv.ParseUint(line,10,64)
                if err != nil {
                    fmt.Println("Error converting to integer:", err)
                } else {
                    fmt.Println("Got tag number",number)
                        BadgeTag(number)
                }
                }
        }

		// Close the file handle for the pipe.
		file.Close()
	}
}

func NFClistener() {
  if cfg.NFCmode=="pipe" {
          NamedPipeListener()
  } else if cfg.NFCmode=="10d-kbd" {
      // 10 decimal digits - USB Keyboard device
      readkbd(1) 
  } else if cfg.NFCmode=="10h-kbd" {
      // 10 hex digits - USB Keyboard device
      readkbd(0) 
  } else {
    for {
      // Default - Serial device w/ weird protocol
      tag := readrfid()
      //var tag uint64 
      //tag = 0
      //time.Sleep(time.Second * 3)
      if (tag !=  0) {
        fmt.Println("Got RFID",tag)
        BadgeTag(tag)
      }
    }
  }
}

// This reads regular numbers from the device
func OLD_NFClistener() {

	file,err := os.Open(cfg.NFCdevice)
	if (err != nil) {
		log.Fatal("Error Opening NFC device : ",err)
		return
	}
	defer file.Close()

	// Create a bufio.Scanner to read lines from the file
	scanner := bufio.NewScanner(file)

	// Loop through each line
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println("Got NFC Tag: "+line)
			// Convert the line to an integer
		number, err := strconv.ParseUint(line,10,64)
		if err != nil {
			fmt.Println("Error converting to integer:", err)
		} else {
			fmt.Println("Got tag number",number)
		}
		BadgeTag(number)
	}

	// Check for errors from scanner
	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading file:", err)
	}

}

func mqttconnect() {
	// Connect to the MQTT broker
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal("MQTT Connect error: ",token.Error())
	}
	fmt.Println("MQTT Connected")
}

func changeScreen(newScreen Screen) {
        err := currentScreen.Close()
        if (err != nil) {
            fmt.Errorf("Error closing screen %v\n",err)
        }
        currentScreen = newScreen
        err = currentScreen.Init()
        if (err != nil) {
            fmt.Errorf("Error closing screen %v\n",err)
        }
        currentScreen.Draw()
}

// Do not call directy. Use uiEvent queue
func display_update() {
        currentScreen.Draw()
        video_update()
}

// Any display updates must be done through
// uiEvent channel so they can be queued
// All events should just to currentScreen --- ???
// TODO DEPRICATE THIS
func display() {
    display_update()
    for {
            evt := <- uiEvent
            currentScreen.HandleEvent(evt)
            /*
            select {
                    case evt := <- uiEvent :
                      switch (evt.Event) {
                        case Event_Encoderknob:
                             if (evt.Name == "button") {
                                     knobpos=0
                             }
                             video_updateknob(evt)
                             break
                        default:
                                display_update()
                    
            }

                    case <-time.After(60 * time.Second):
                            fmt.Printf("Minute Timeout")
                            display_update()
                    
            }
            */
    }
}

func main() {
	cfgfile := flag.String("cfg","goratt.cfg","Config file")
	flag.Parse()

    uiEvent = make(chan UIEvent,10)
	f, err := os.Open(*cfgfile)
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if (err != nil) {
	    log.Fatal("Config Decode error: ",err)
	}

    if (cfg.DoorPin == nil) {
            doorpin_18 := 18
            cfg.DoorPin = &doorpin_18
    }

	switch (cfg.Mode) {
		case "servo":
			servo_reset(cfg.ServoClose)
		case "openhigh":
			door_reset(false)
		case "openlow":
			door_reset(true)
		case "":
		case "none":
			break
		default:
			panic("Mode in configfile must be \"none\", \"servo\", \"openhigh\" or \"openlow\"")
	}

  if (cfg.LEDpipe != "") {
    LEDfile,err = os.OpenFile(cfg.LEDpipe, os.O_RDWR, 0644)
    if (LEDfile == nil) {
      log.Fatal("Error opening LED pipe: ",err)
    }
    defer  LEDfile.Close()
  }

  gpio_init()


    mqtt_init()

    // Init video
    lastEventTime = time.Now()
    video_init()

	ReadTagFile()
	GetACLList()
    rotary_init();

	//hw.PinClear(23)
	//hw.PinClear(24)
	//hw.Close()

    LEDupdateIdleString(LEDconnectionLost)
    LEDwriteString(LEDconnectionLost)

    go watchdogHandler()
    go mqttconnect()
	go NFClistener()
	go PingSender()

    s := idleScreen{}
    s.Init()
    currentScreen = &s

    go display()

	// Wait for a signal to exit
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, os.Kill)
	signal.Notify(c, syscall.SIGTERM)
	//fmt.Println("Waitsignal")
	<-c
	fmt.Println("Got Terminate Signal")
	// Disconnect from the MQTT broker
    video_clear()
	client.Disconnect(250)
	fmt.Println("Disconnected from the MQTT broker")
    LEDwriteString(LEDterminated)
}

