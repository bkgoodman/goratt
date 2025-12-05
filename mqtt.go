package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"crypto/x509"
	"io/ioutil"
	"time"
	"github.com/eclipse/paho.mqtt.golang"
)

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

func mqtt_init() {
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
