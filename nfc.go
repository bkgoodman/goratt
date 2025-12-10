package main

import (
	"log"
	"strconv"
    "github.com/kenshaw/evdev"
    "context"
	"github.com/tarm/serial"
	"bytes"
    "time"
    "fmt"
    "github.com/hjkoskel/govattu"
)


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
			var message string = fmt.Sprintf("{\"allowed\":%d,\"member\":\"%s\"}",tag.Allowed,tag.Member)
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
  LEDwriteString(LEDaccessDenied)
	time.Sleep(time.Duration(3) * time.Second)
	hw.PinClear(23)
  LEDwriteString(LEDidleString)
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
                                //Signout()
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

