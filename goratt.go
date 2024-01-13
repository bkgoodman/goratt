package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"strconv"
	"bufio"
	 "os/signal"
	"crypto/x509"
	"io/ioutil"
	"sync"
	"gopkg.in/yaml.v2"
	"flag"
	"encoding/json"
  "github.com/hjkoskel/govattu"
	"time"
  	// "strings" 

	"github.com/eclipse/paho.mqtt.golang"

	"encoding/base64"
	"net/http"

	"github.com/tarm/serial"
	"bytes"
)


var client mqtt.Client

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

   TagFile string `yaml:"TagFile"`
   ServoClose int `yaml:"ServoClose"`
   ServoOpen int `yaml:"ServoOpen"`
   WaitSecs int `yaml:"WaitSecs"`

   NFCdevice string `yaml:"NFCdevice"`
}

// In-memory ACL list
type ACLlist struct {
	Tag uint64
	Level  int
	Member string
	Allowed bool
}

var validTags []ACLlist

var cfg RattConfig
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

func onMessageReceived(client mqtt.Client, message mqtt.Message) {
	//fmt.Printf("Received message on topic: %s\n", message.Topic())
	//fmt.Printf("Message: %s\n", message.Payload())

	// Is this aun update ACL message? If so - Update
	if (message.Topic() == "ratt/control/broadcast/acl/update") {
		fmt.Println("Got ACL Update message")
		GetACLList()
	}
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
			fmt.Printf("Tag %d Member %s Access %s",id,tag.Member,access)

			var topic string = fmt.Sprintf("ratt/status/node/%s/personality/access",cfg.ClientID)
			var message string = fmt.Sprintf("{\"allowed\":tag.Allowed,\"member\":\"%s\"}",tag.Member)
			client.Publish(topic,0,false,message)

			if (tag.Allowed) {
				open_servo(cfg.ServoOpen, cfg.ServoClose, cfg.WaitSecs, cfg.Mode)
			  return
			}
		} 

	}

	if (found == false) {
		fmt.Println("Tag not found",id)
	}
	hw, err := govattu.Open()
	if err != nil {
		panic(err)
	}
	defer  hw.Close()
	hw.PinSet(23)
	time.Sleep(time.Duration(3) * time.Second)
	hw.PinClear(23)
	return
}

// THis reads from the weird USB RFID Serial Protocol
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

func NFClistener() {
	for {
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

func main() {
	openflag := flag.Bool("holdopen",false,"Hold door open indefinitley")
	cfgfile := flag.String("cfg","goratt.cfg","Config file")
	flag.Parse()

	f, err := os.Open(*cfgfile)
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if (err != nil) {
	    log.Fatal("Config Decode error: ",err)
	}


	switch (cfg.Mode) {
		case "servo":
			servo_reset(cfg.ServoClose)
		case "openhigh":
			door_reset(false)
		case "openlow":
			door_reset(true)
		default:
			panic("Mode in configfile must be \"servo\", \"openhigh\" or \"openlow\"")
	}
  // REMOVE
  //NFClistener()

    hw, err := govattu.Open()
    if err != nil {
      panic(err)
    }
		// 18 is SERVO!
		hw.PinMode(23,govattu.ALToutput)
		hw.PinMode(24,govattu.ALToutput)
		hw.PinMode(25,govattu.ALToutput)
		hw.PinSet(23)
		hw.PinSet(24)
		hw.PinSet(25)
		hw.ZeroPinEventDetectMask()

		if (*openflag) {
				open_servo(cfg.ServoOpen, cfg.ServoClose, cfg.WaitSecs, cfg.Mode)
		}


	// MQTT broker address
	broker := fmt.Sprintf("ssl://%s:%d",cfg.MqttHost,cfg.MqttPort)

	// MQTT client ID
	clientID := cfg.ClientID

	// MQTT topic to subscribe to
	topic := "#"

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
		SetTLSConfig(tlsConfig).
		SetDefaultPublishHandler(onMessageReceived)

	// Create an MQTT client
	client = mqtt.NewClient(opts)

	// Connect to the MQTT broker
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal("MQTT Connect error: ",token.Error())
	}

	// Subscribe to the topic
	if token := client.Subscribe(topic, 0, nil); token.Wait() && token.Error() != nil {
		log.Fatal("MQTT Subscribe error: ",token.Error())
	}

	ReadTagFile()
	GetACLList()

	hw.PinClear(23)
	hw.PinClear(24)
	hw.PinClear(25)
	hw.Close()

	go NFClistener()
	go PingSender()
	fmt.Printf("Connected to %s\n", broker)

	// Wait for a signal to exit
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	//fmt.Println("Waitsignal")
	<-c

	fmt.Println("Got Terminate Signal")
	// Disconnect from the MQTT broker
	client.Disconnect(250)
	fmt.Println("Disconnected from the MQTT broker")
}

