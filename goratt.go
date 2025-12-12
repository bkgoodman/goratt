package main

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/hjkoskel/govattu"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
	// "strings"

	"github.com/eclipse/paho.mqtt.golang"

	"encoding/base64"
	"net/http"

	//"github.com/tarm/serial"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"github.com/kenshaw/evdev"
	"goratt/wiegland"
)

var client mqtt.Client
var myOpenTopic string

type RattConfig struct {
	CACert      string `yaml:"CACert"`
	ClientCert  string `yaml:"ClientCert"`
	ClientKey   string `yaml:"ClientKey"`
	ClientID    string `yaml:"ClientID"`
	MqttHost    string `yaml:"MqttHost"`
	MqttPort    int    `yaml:"MqttPort"`
	ApiURL      string `yaml:"ApiURL"`
	ApiCAFile   string `yaml:"ApiCAFile"`
	ApiUsername string `yaml:"ApiUsername"`
	ApiPassword string `yaml:"ApiPassword"`
	Resource    string `yaml:"Resource"`
	Mode        string `yaml:"Mode"`
	OpenSecret  string `yaml:"OpenSecret"`
	OpenToolName  string `yaml:"OpenToolName"`

	TagFile    string `yaml:"TagFile"`
	ServoClose int    `yaml:"ServoClose"`
	ServoOpen  int    `yaml:"ServoOpen"`
	WaitSecs   int    `yaml:"WaitSecs"`

	NFCdevice string `yaml:"NFCdevice"`
	NFCmode   string `yaml:"NFCmode"`

	DoorPin *int   `yaml:"DoorPin"`
	LEDpipe string `yaml:"LEDpipe"`

	GreenLED  *uint8 `yaml:"GreenLED"`
	YellowLED *uint8 `yaml:"YellowLED"`
	RedLED    *uint8 `yaml:"RedLED"`
}

// In-memory ACL list
type ACLlist struct {
	Tag     uint64
	Level   int
	Member  string
	Allowed bool
}

var validTags []ACLlist

var LEDfile *os.File
var LEDidleString string
var cfg RattConfig

const (
	LEDconnectionLost = "@2 !150000 001010"
	LEDnormalIdle     = "@3 !150000 400000"
	LEDaccessGranted  = "@1 !50000 8000"
	LEDaccessDenied   = "@2 !10000 ff"
	LEDterminated     = "@0 010101"
)

// From API - off the wire
type ACLentry struct {
	Tagid         string `json:"tagid"`
	Tag_ident     string `json:"tag_ident"`
	Allowed       string `json:"allowed"`
	Warning       string `json:"warning"`
	Member        string `json:"member"`
	Nickname      string `json:"nickname"`
	Plan          string `json:"plan"`
	Last_accessed string `json:"last_accessed"`
	Level         int    `json:"level"`
	Raw_tag_id    string `json:"raw_tag_id"`
}

type OpenRequest struct {
	Member    string `json:"member"`
	ToolName  string `json:"tool"`
	Timestamp uint64 `json:"timestamp"`
	Signature string `json:"signature"`
}

var aclfileMutex sync.Mutex

func LEDupdateIdleString(str string) {
	LEDidleString = str
}

func LEDwriteString(str string) {
	if LEDfile != nil {
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
	url := fmt.Sprintf("%s/api/v1/resources/%s/acl", cfg.ApiURL, cfg.Resource)

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
	file, err := os.Create(cfg.TagFile + ".tmp")

	validTags = validTags[:0]
	// Print the desired field (e.g., "name") from each dictionary
	for index, item := range items {
		_ = index
		//fmt.Println("ID:", index,item.Tagid,item.Raw_tag_id)

		number, err := strconv.ParseUint(item.Raw_tag_id, 10, 64)
		if err == nil {
			validTags = append(validTags, ACLlist{
				Tag:     number,
				Level:   item.Level,
				Member:  item.Member,
				Allowed: (item.Allowed == "allowed"),
			})
		}
		access := "denied"
		if item.Allowed == "allowed" {
			access = "allowed"
		}
		_, err = file.WriteString(fmt.Sprintf("%d %s %d %s\n", number, access, item.Level, item.Member))
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

	var topic string = fmt.Sprintf("ratt/status/node/%s/acl/update", cfg.ClientID)
	var message string = "{\"status\":\"downloaded\"}"
	client.Publish(topic, 0, false, message)

}

func ReadTagFile() {
	aclfileMutex.Lock()
	defer aclfileMutex.Unlock()

	file, err := os.Open(cfg.TagFile)
	if err != nil {
		log.Fatal("Error Reading Tag File: ", err)
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
		_, err := fmt.Sscanf(line, "%d %s %d %s", &tag, &access, &level, &member)
		if err == nil {
			validTags = append(validTags, ACLlist{
				Tag:     tag,
				Level:   level,
				Member:  member,
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
		log.Fatal("MQTT Subscribe error: ", token.Error())
	}

	if token := client.Subscribe(myOpenTopic, 0, nil); token.Wait() && token.Error() != nil {
		log.Fatal("MQTT Subscribe error: ", token.Error())
	}
	// Slow Blue Pulse
	LEDupdateIdleString(LEDnormalIdle)
	LEDwriteString(LEDnormalIdle)

}

func onConnectionLost(client mqtt.Client, err error) {
	// Panic - because a restart will fix???
	// panic(fmt.Errorf("MQTT CONNECTION LOST: %s",err))
	fmt.Printf("MQTT CONNECTION LOST: %s", err)
	// Slow Yellow Wink
	LEDupdateIdleString(LEDconnectionLost)
	LEDwriteString(LEDconnectionLost)
}

// SignRequest computes an HMAC-SHA256 over (member || timestampBE)
// using a base64-encoded shared secret.
func SignOpenRequest(base64Secret string, member string, tool string, ts uint64) (sigHex string, sigBase64 string, err error) {
	// 1) Decode base64 secret
	secret, err := base64.StdEncoding.DecodeString(base64Secret)
	if err != nil {
		return "", "", fmt.Errorf("invalid base64 secret: %w", err)
	}
	if len(secret) == 0 {
		return "", "", fmt.Errorf("secret cannot be empty")
	}

	// 2) Prepare message: member (bytes) + timestamp (uint64 big-endian)
	msg := make([]byte, 0, len(member)+len(tool)+8)
	msg = append(msg, []byte(member)...) // UTF-8 bytes of member
	msg = append(msg, []byte(tool)...) // UTF-8 bytes of member

	var tsBuf [8]byte
	binary.BigEndian.PutUint64(tsBuf[:], ts)
	msg = append(msg, tsBuf[:]...) // Append timestamp

	// 3) Compute HMAC-SHA256
	mac := hmac.New(sha256.New, secret)
	mac.Write(msg)
	sum := mac.Sum(nil)

	// 4) Return both hex and base64 encodings (choose whichever you prefer)
	return hex.EncodeToString(sum), base64.StdEncoding.EncodeToString(sum), nil
}

func VerifyOpenRequestSignature(base64Secret string, member string, tool string, ts uint64, providedSig string) error {
	sigHex, sigBase64, err := SignOpenRequest(base64Secret, member, tool, ts)
	if err != nil {
		return err
	}

	// Try hex first
	if decodedHex, errHex := hex.DecodeString(providedSig); errHex == nil {
		expectedHex, _ := hex.DecodeString(sigHex) // should not fail
		if subtle.ConstantTimeCompare(decodedHex, expectedHex) == 1 {
			return nil
		}
	}

	// Try base64 next
	if decodedB64, errB64 := base64.StdEncoding.DecodeString(providedSig); errB64 == nil {
		expectedB64, _ := base64.StdEncoding.DecodeString(sigBase64)
		if subtle.ConstantTimeCompare(decodedB64, expectedB64) == 1 {
			return nil
		}
	}

	return fmt.Errorf("Signature verification failed")
}

func onMessageReceived(client mqtt.Client, message mqtt.Message) {
	//fmt.Printf("Received message on topic: %s\n", message.Topic())
	//fmt.Printf("Message: %s\n", message.Payload())
	var topic = fmt.Sprintf("ratt/control/node/%s/open", cfg.ClientID)

	// Is this aun update ACL message? If so - Update
	if message.Topic() == "ratt/control/broadcast/acl/update" {
		fmt.Println("Got ACL Update message")
		GetACLList()
	} else if message.Topic() == topic {
		fmt.Println("Got OPEN request")
		if cfg.OpenSecret == "" {
			fmt.Printf("No OpenSecret configured - remote open disabled")
			return
		}
		if cfg.OpenToolName == "" {
			fmt.Printf("No OpenToolName configured - remote open disabled")
			return
        }
		var request OpenRequest
		err := json.Unmarshal(message.Payload(), request)
		if err != nil {
			fmt.Println("Error decoding JSON:", err)
			return
		}
        if (cfg.OpenToolName != request.ToolName) {
			fmt.Printf("Wrong toolname \"%s\" - expected \"%s\"\n",request.ToolName,cfg.OpenToolName)
			return
        }
		fmt.Printf("Open request member \"%s\" door \"%s\" Timestamp \"%d\" Signature \"%s\"\n", request.Member, request.ToolName, request.Timestamp, request.Signature)

		timestamp := time.Unix(int64(request.Timestamp), 0) // seconds + 0 nanos
		windowStart := timestamp.Add(-5 * time.Minute)
		windowEnd := timestamp.Add(5 * time.Minute)
		now := time.Now()

		if now.Before(windowStart) || now.After(windowEnd) {
			fmt.Println("Open request timeout")
			return
		}

		err = VerifyOpenRequestSignature(cfg.OpenSecret, request.Member, request.ToolName, request.Timestamp, request.Signature)
		if err != nil {
			fmt.Printf("Open request verification failed: %s\n", err)
			return
		}

		var topic string = fmt.Sprintf("ratt/status/node/%s/personality/access", cfg.ClientID)
		var message string = fmt.Sprintf("{\"allowed\":1,\"member\":\"%s\"}", request.Member)
		client.Publish(topic, 0, false, message)
		open_servo(cfg.ServoOpen, cfg.ServoClose, cfg.WaitSecs, cfg.Mode)
	}
}

func PingSender() {

	for {
		var topic string = fmt.Sprintf("ratt/status/node/%s/ping", cfg.ClientID)
		var message string = "{\"status\":\"ok\"}"
		client.Publish(topic, 0, false, message)
		time.Sleep(120 * time.Second)
	}
}

// Read from KEYBOARD in simple 10h + cr format
func readkdb_10h() {
	fmt.Println("USB 10H Keyboard mode")
	device, err := evdev.OpenFile(cfg.NFCdevice)
	if err != nil {
		log.Fatal("Error Opening NFC device : ", err)
		return
	}
	defer device.Close()

	fmt.Printf("Opened keyboard device: %s\n", device.Name()) // Device name from evdev
	fmt.Printf("Vendor: 0x%04x, Product: 0x%04x\n", device.ID().Vendor, device.ID().Product)
	fmt.Println("Listening for keyboard events. Press keys to see output.")
	fmt.Println("Press Ctrl+C to exit.")
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
				if event.Value == 1 {
					//log.Printf("received key event: %+v TYPE:%+v/%T", event,event.Type,event.Type)
					if event.Type == evdev.KeyEnter {
						number, err := strconv.ParseUint(strbuf, 16, 64)
						number &= 0xffffffff
						log.Printf("Got 10h String %s BadgeId %d\n", strbuf, number)
						if err == nil {
							BadgeTag(number)
						} else {
							log.Printf("Bad hex badge line \"%s\"\n", strbuf)
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

func NFClistener() {
	if cfg.NFCmode == "wiegland" {
		reader := &wiegland.RFIDReader{}
		if err := reader.Initialize(cfg.NFCdevice, 9600); err != nil {
			log.Fatalf("Wiegland init failed: %v", err)
		}
		defer reader.Close()

		for {
			tag, err := reader.GetCard()
			if err != nil {
				fmt.Println("Weigland error", err)
			} else {
				fmt.Println("Got RFID", tag)
				if tag != 0 {
					BadgeTag(tag)
				}
			}

		}
	} else if cfg.NFCmode == "10h-kbd" {
		// 10 hex digits - USB Keyboard device
		readkdb_10h()
	} else {
		for {
			// Default - Serial device w/ weird protocol
			tag := readrfid()
			//var tag uint64
			//tag = 0
			//time.Sleep(time.Second * 3)
			if tag != 0 {
				fmt.Println("Got RFID", tag)
				BadgeTag(tag)
			}
		}
	}
}

// This reads regular numbers from the device
func OLD_NFClistener() {

	file, err := os.Open(cfg.NFCdevice)
	if err != nil {
		log.Fatal("Error Opening NFC device : ", err)
		return
	}
	defer file.Close()

	// Create a bufio.Scanner to read lines from the file
	scanner := bufio.NewScanner(file)

	// Loop through each line
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println("Got NFC Tag: " + line)
		// Convert the line to an integer
		number, err := strconv.ParseUint(line, 10, 64)
		if err != nil {
			fmt.Println("Error converting to integer:", err)
		} else {
			fmt.Println("Got tag number", number)
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
		log.Fatal("MQTT Connect error: ", token.Error())
	}
	fmt.Println("MQTT Connected")
}

func main() {
	openflag := flag.Bool("holdopen", false, "Hold door open indefinitley")
	cfgfile := flag.String("cfg", "goratt.cfg", "Config file")
	flag.Parse()

	f, err := os.Open(*cfgfile)
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		log.Fatal("Config Decode error: ", err)
	}

	if cfg.ClientID == "" {
		panic("ClientID missing in Config file")
	}

	myOpenTopic = fmt.Sprintf("ratt/control/node/%s/open", cfg.ClientID)
	if cfg.LEDpipe != "" {
		LEDfile, err = os.OpenFile(cfg.LEDpipe, os.O_RDWR, 0644)
		if LEDfile == nil {
			log.Fatal("Error opening LED pipe: ", err)
		}
		defer LEDfile.Close()
	}
	hw, err := govattu.Open()
	if err != nil {
		panic(err)
	}
	if cfg.RedLED != nil {
		hw.PinMode(*cfg.RedLED, govattu.ALToutput)
		hw.PinSet(*cfg.RedLED)
	}
	if cfg.GreenLED != nil {
		hw.PinMode(*cfg.GreenLED, govattu.ALToutput)
		hw.PinSet(*cfg.GreenLED)
	}
	if cfg.YellowLED != nil {
		hw.PinMode(*cfg.YellowLED, govattu.ALToutput)
		hw.PinSet(*cfg.YellowLED)
	}
	hw.ZeroPinEventDetectMask()

	if *openflag {
		open_servo(cfg.ServoOpen, cfg.ServoClose, cfg.WaitSecs, cfg.Mode)
	}

	// MQTT broker address
	broker := fmt.Sprintf("ssl://%s:%d", cfg.MqttHost, cfg.MqttPort)

	// MQTT client ID
	clientID := cfg.ClientID

	// Load client key pair for TLS (replace with your own paths)
	cert, err := tls.LoadX509KeyPair(cfg.ClientCert, cfg.ClientKey)
	if err != nil {
		log.Fatal("Error loading X509 Keypair: ", err)
	}

	// Load your CA certificate (replace with your own path)
	caCert, err := ioutil.ReadFile(cfg.CACert)
	if err != nil {
		log.Fatal("Error reading CA file: ", cfg.CACert, err)
	}

	// Create a certificate pool and add your CA certificate
	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(caCert)

	// Create a TLS configuration
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caPool,
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

	ReadTagFile()
	GetACLList()

	if cfg.RedLED != nil {
		hw.PinClear(*cfg.RedLED)
	}
	if cfg.GreenLED != nil {
		hw.PinClear(*cfg.GreenLED)
	}
	if cfg.YellowLED != nil {
		hw.PinClear(*cfg.YellowLED)
	}
	hw.Close()

	LEDupdateIdleString(LEDconnectionLost)
	LEDwriteString(LEDconnectionLost)

	go mqttconnect()
	go NFClistener()
	go PingSender()

	//dymo_label("- Ready -")
	// Wait for a signal to exit
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, os.Kill)
	signal.Notify(c, syscall.SIGTERM)
	//fmt.Println("Waitsignal")
	<-c

	fmt.Println("Got Terminate Signal")
	// Disconnect from the MQTT broker
	client.Disconnect(250)
	fmt.Println("Disconnected from the MQTT broker")
	LEDwriteString(LEDterminated)
}
