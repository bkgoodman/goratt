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
	gpiocdev "github.com/warthog618/go-gpiocdev"
	// "github.com/warthog618/gpiod/device/rpi"
   // "github.com/hjkoskel/govattu"
	"time"
  	// "strings" 

	"github.com/eclipse/paho.mqtt.golang"

	"encoding/base64"
	"net/http"

	"github.com/tarm/serial"
	"bytes"
    "github.com/kenshaw/evdev"
    "context"
)


var client mqtt.Client
var alertMessage string
var watchdog time.Time


const (
        Event_Update = iota
        Event_Alert
        Event_Encoderknob
)

var occupiedBy *string

var lastEventTime time.Time
var safelight = false

type UIEvent struct {
        Event int
        Name string
}

var uiEvent chan UIEvent


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

func FetchCalendarURL() {
        if nextCalFetch.After(time.Now()) {
                return
        }
        if cfg.CalendarURL == "" {
                return
        }

    nextCalFetch = time.Now().Add(60 * 15 * time.Second)
    // Make an HTTP GET request to the URL.
    response, err := http.Get(cfg.CalendarURL)
    if err != nil {
        log.Fatalf("Failed to fetch URL: %v", err)
    }
    defer response.Body.Close()

    // Check for a successful HTTP status code (e.g., 200 OK).
    if response.StatusCode != http.StatusOK {
        log.Fatalf("Received non-200 status code: %d", response.StatusCode)
    }

    // Create an instance of your struct to hold the data.
    var cal []CalEntry

    // Use a JSON decoder to unmarshal the response body into the struct.
    err = json.NewDecoder(response.Body).Decode(&cal)
    if err != nil {
        log.Fatalf("Failed to decode JSON: %v", err)
    }

    	// Iterate over the slice and print the data.
	for _, item := range cal {
		fmt.Printf("Summary: %s\n", item.SUMMARY)
		fmt.Printf("Organizer: %s\n", item.ORGANIZER)
		fmt.Printf("When: %s\n", item.WHEN)
		fmt.Println("--------------------")
	}

    if len(cal) != 0 {
            nextCalEntry = &cal[0]
    } else {
            nextCalEntry = nil
    }

}

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

func onConnectHandler(client mqtt.Client) {
	fmt.Println("MQTT Connection Established")
	// Subscribe to the topic
	var topic = "ratt/control/broadcast/acl/update"
	if token := client.Subscribe(topic, 0, nil); token.Wait() && token.Error() != nil {
		log.Fatal("MQTT Subscribe error: ",token.Error())
	}

    // Slow Blue Pulse
    LEDupdateIdleString(LEDnormalIdle)
    LEDwriteString(LEDnormalIdle)
    petWatchdog()

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

func onConnectionLost(client mqtt.Client, err error) {
	// Panic - because a restart will fix???
	// panic(fmt.Errorf("MQTT CONNECTION LOST: %s",err))
	log.Printf("MQTT CONNECTION LOST: %s",err)
    // Slow Yellow Wink
    LEDupdateIdleString(LEDconnectionLost)
    LEDwriteString(LEDconnectionLost)
}
func onMessageReceived(client mqtt.Client, message mqtt.Message) {
	//fmt.Printf("Received message on topic: %s\n", message.Topic())
	//fmt.Printf("Message: %s\n", message.Payload())

	// Is this aun update ACL message? If so - Update
    petWatchdog()
	if (message.Topic() == "ratt/control/broadcast/acl/update") {
		log.Println("Got ACL Update message")
		GetACLList()
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

// This tag number tried to badge in
func BadgeTag(id uint64) {
	var found bool = false
	for _,tag := range validTags {
		if id == tag.Tag {
			found = true
			access := "Denied"
			if (tag.Allowed) { access = "Allowed" }
			log.Printf("Tag %d Member %s Access %s",id,tag.Member,access)


			if (tag.Allowed) {
				//open_servo(cfg.ServoOpen, cfg.ServoClose, cfg.WaitSecs, cfg.Mode)
                Signin(tag.Member)
			    return
			}  else {
                Disallowed()
			    return
            }
		} 

	}

	if (found == false) {
		log.Println("Tag not found",id)
        TagNotFound()
	}
    /*
	hw, err := govattu.Open()
	if err != nil {
		panic(err)
	}
	defer  hw.Close()
	hw.PinSet(23)
  LEDwriteString(LEDaccessDenied)
	time.Sleep(time.Duration(3) * time.Second)
	hw.PinClear(23)
  LEDwriteString(LEDidleString)
  */
	return
}

// Read from KEYBOARD in simple 10h + cr format
func readkbd(devtype int) {
	log.Println("USB 10H Keyboard mode")
	device,err := evdev.OpenFile(cfg.NFCdevice)
	if (err != nil) {
		log.Fatal("Error Opening NFC device : ",err)
		return
	}
	defer device.Close()

    
    log.Printf("Opened keyboard device: %s\n", device.Name()) // Device name from evdev
	log.Printf("Vendor: 0x%04x, Product: 0x%04x\n", device.ID().Vendor, device.ID().Product)
	log.Println("Listening for keyboard events. Press keys to see output.")
	log.Println("Press Ctrl+C to exit.")
    ch := device.Poll(context.Background())
    strbuf := ""
    loop:
	for {
		select {
		case event := <-ch:
			// channel closed
			if event == nil {
				break loop
			}

			switch event.Type.(type) {
			case evdev.KeyType:
                if (event.Value == 1) {
                        //log.Printf("received key event: %+v TYPE:%+v/%T", event,event.Type,event.Type)
                        // We do this so we can map a GPIO as an escape key easily if we want
                        if (event.Type == evdev.KeyEscape) {
                                Signout()
                        } else if (event.Type == evdev.KeyEnter) {
                                var number uint64
                                if (devtype == 0) {
                                        number, err = strconv.ParseUint(strbuf,16,64)
                                } else if (devtype == 1) {
                                        number, err = strconv.ParseUint(strbuf,10,64)
                                }
                                number &= 0xffffffff
                                log.Printf("Got String %s BadgeId %d\n",strbuf,number)
                                if (err == nil) {
                                        BadgeTag(number)
                                } else {
                                        log.Printf("Bad hex badge line \"%s\"\n",strbuf)
                                }
                                strbuf = ""
                        } else {
                                //log.Printf("KEY (%d) \"%s\"\n",event.Type,event.Type)
                                s := evdev.KeyType(event.Code).String()
                                //log.Printf("ecode %+v keytype %T \"%v\"\n",event.Code,s,s)
                                strbuf += s
                                //log.Printf("strbuf now \"%s\"\n",strbuf)
                        }
                }

			}
		}
	}
}

// THis reads from the weird USB RFID Serial Protocol w/ Weird Encoding
func readrfid() uint64  {
      // Open the serial port
    //mode := &serial.Mode{
    //  BaudRate: 115200,
   // }
    //port, err := serial.Open(cfg.NFCdevice,mode)
		c := &serial.Config{Name: cfg.NFCdevice, Baud: 115200, ReadTimeout: time.Second}
    port, err := serial.OpenPort(c)
    if err != nil {
			panic(fmt.Errorf("Canot open tty %s: %v",cfg.NFCdevice,err))
    }
    buff := make([]byte, 9)
    for {
			//fmt.Println("READING")
    	n, err := port.Read(buff)
			//fmt.Println("READ EXIT")
      if err != nil {
			  //fmt.Printf("Fatalbreak %v\n",err)
        //log.Fatal(err)
        break
      }
      if n == 0 {
        //fmt.Println("\nEOF SLEEP")
				time.Sleep(time.Second * 5)
        //fmt.Println("\nENDSLEEP")
        continue
      }
      if n != 9 {
        //fmt.Println("\nPARTIAL")
       continue
      }
      //fmt.Printf("%x", string(buff[:n]))
      break
    }

			// fmt.Printf("\nGotdata\n")

        // Define the preambles and terminator
    preambles := []byte{0x02, 0x09}
    terminator := []byte{0x03}


    // Verify the preambles
    if !bytes.Equal(buff[0:2], preambles) {
      //panic(fmt.Errorf("invalid preambles: %v", buff[0:2]))
			return 0
    }

    // Verify the terminator
    if !bytes.Equal(buff[8:9], terminator) {
      //panic(fmt.Errorf("invalid terminator: %v", buff[8:9]))
			return 0
    }


    // Print the data
    //fmt.Println(buff)
    data := buff[1:7]
    // XOR all the bytes in the slice
    xor := data[0]
    for i := 1; i < len(data); i++ {
        xor ^= data[i]
        //fmt.Printf("Byte %d is %x\n",i,data[i])
    }

    var tagno uint64
    tagno= (uint64(data[2]) << 24) | (uint64(data[3])<<16) | (uint64(data[4]) <<8 ) | uint64(data[5])
    //fmt.Printf("XOR is %x should be %x Tagno %d\n",xor,buff[7],tagno)
    if xor!= buff[7] {
      return 0
    }

    return tagno

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

func Signin(user string) {
    if (occupiedBy == nil) {
            occupiedBy = &user
			var topic string = fmt.Sprintf("ratt/status/node/%s/personality/login",cfg.ClientID)
			var message string = fmt.Sprintf("{\"allowed\":true,\"member\":\"%s\"}",user)
			client.Publish(topic,0,false,message)
    } else {
			var topic string = fmt.Sprintf("ratt/status/node/%s/personality/logout",cfg.ClientID)
			var message string = fmt.Sprintf("{\"allowed\":true,\"member\":\"%s\"}",occupiedBy)
			client.Publish(topic,0,false,message)
            occupiedBy = nil
    }
    lastEventTime = time.Now()
    UpdateUser()

}

func Disallowed() {
   alertMessage = "Not Authorized"
   uiEvent <- UIEvent { Event: Event_Alert, }
}

func TagNotFound() {
   alertMessage = "Tag Not Found"
   uiEvent <- UIEvent { Event: Event_Alert, }
}

func AlreadyOccupied() {
   alertMessage = "Already Occupied"
   uiEvent <- UIEvent { Event: Event_Alert, }
}


func Signout() {
    lastEventTime = time.Now()
    occupiedBy = nil
    UpdateUser()
}

// Update user - deals with safelight state
func UpdateUser() {
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

// Do not call directy. Use uiEvent queue
func display_update() {
        if (safelight) {
                // Safelight on
          video_draw()
        } else if (occupiedBy == nil) {
                // Unoccupied
                video_available()
        } else  {
                // Occupied
                video_comein()
        }

        video_update()
}

// Any display updates must be done through
// uiEvent channel so they can be queued
func display() {
    FetchCalendarURL()
    display_update()
    for {
            FetchCalendarURL()
            select {
                    case evt := <- uiEvent :
                      switch (evt.Event) {
                        case Event_Alert:
                            video_alert()
                            video_update()
                            time.Sleep(3 * time.Second)
                            display_update()
                            break
                        case Event_Encoderknob:
                             video_updateknob(evt)
                             break
                        default:
                                display_update()
                    
            }

                    case <-time.After(60 * time.Second):
                            fmt.Printf("Minute Timeout")
                            display_update()
                    
            }
    }
}



func safelightCallback(evt gpiocdev.LineEvent) {
        if (evt.Type == 1) {
                SafelightOff()
        } else {
                SafelightOn()
        }
}

func badgeoutCallback(evt gpiocdev.LineEvent) {
        Signout()
}
func main() {
	openflag := flag.Bool("holdopen",false,"Hold door open indefinitley")
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


	chip := "gpiochip0"
	l, err := gpiocdev.RequestLine(chip, 18,
		gpiocdev.WithPullUp,
		gpiocdev.WithBothEdges,
        gpiocdev.WithDebounce(10* time.Millisecond),
		gpiocdev.WithEventHandler(safelightCallback))
	if err != nil {
		fmt.Printf("RequestLine returned error: %s\n", err)
		if err == syscall.Errno(22) {
			fmt.Println("Note that the WithPullUp option requires Linux 5.5 or later - check your kernel version.")
		}
		os.Exit(1)
	}
	defer l.Close()

	l, err = gpiocdev.RequestLine(chip, 23,
		gpiocdev.WithPullUp,
		gpiocdev.WithBothEdges,
        gpiocdev.WithDebounce(10* time.Millisecond),
		gpiocdev.WithEventHandler(badgeoutCallback))
	if err != nil {
		fmt.Printf("RequestLine returned error: %s\n", err)
		if err == syscall.Errno(22) {
			fmt.Println("Note that the WithPullUp option requires Linux 5.5 or later - check your kernel version.")
		}
		os.Exit(1)
	}
	defer l.Close()

    // If we are already on
    if ll,_ :=l.Value() ; ll  ==2  {
            SafelightOn()
    }
    /*
    hw, err := govattu.Open()
    if err != nil {
      panic(err)
    }
		hw.PinMode(23,govattu.ALToutput)
		hw.PinMode(24,govattu.ALToutput)
		//hw.PinMode(25,govattu.ALTinput)
		hw.PinSet(23)
		hw.PinSet(24)
		hw.ZeroPinEventDetectMask()
        //hw.SetPinEventDetectMask(25, govattu.PINE_FALL|govattu.PINE_RISE)

		if (*openflag) {
				open_servo(cfg.ServoOpen, cfg.ServoClose, cfg.WaitSecs, cfg.Mode)
		}
        */
        _ = openflag


	// MQTT broker address
	broker := fmt.Sprintf("ssl://%s:%d",cfg.MqttHost,cfg.MqttPort)

	// MQTT client ID
	clientID := cfg.ClientID

	// Load client key pair for TLS (replace with your own paths)
	cert, err := tls.LoadX509KeyPair(cfg.ClientCert, cfg.ClientKey)
	if err != nil {
		log.Fatal("Error loading X509 Keypair: ",err)
	}

		// Load your CA certificate (replace with your own path)
	caCert, err := ioutil.ReadFile(cfg.CACert)
	if err != nil {
		log.Fatal("Error reading CA file: ",cfg.CACert,err)
	}

	// Create a certificate pool and add your CA certificate
	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(caCert)

	// Create a TLS configuration
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs: caPool,
	}

	// Create an MQTT client options
	opts := mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID(clientID).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetKeepAlive(60 * time.Second).
		SetTLSConfig(tlsConfig).
		SetConnectionLostHandler(onConnectionLost).
		SetOnConnectHandler(onConnectHandler).
		SetDefaultPublishHandler(onMessageReceived)

	// Create an MQTT client
	client = mqtt.NewClient(opts)

	mqtt.ERROR = log.New(os.Stdout, "[ERROR] ", 0)
	mqtt.CRITICAL = log.New(os.Stdout, "[CRIT] ", 0)
	mqtt.WARN = log.New(os.Stdout, "[WARN]  ", 0)
	//mqtt.DEBUG = log.New(os.Stdout, "[DEBUG] ", 0)


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

