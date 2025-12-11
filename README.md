# GoRATT

Go-based RATT client 

# Build

1. Clone repo
2. `go build`

For ARM (Raspberry Pi) builds are: `GOARCH=arm go build`
(I think this does 32 bit. `arm64` for 64 bit builds)

# Configure

1. Copy `example.cfg` to `goratt.cfg`
2. Edit as needed in your environment

See section below for parameters

# Run

`./goratt`


# Configuration 

All fields are manditory

| Parameter | Description |
| ---------- | ------------- |
| CACert | Path of file for the Root CA of your MQTT server |
| ClientCert  | Path of file for your GoRATT's TLS client cert |
| ClientKey | Path for TLS client key |
| ClientID | *Unique* Client ID for Auth backend. MAC address of machine, no seperators |
| MqttHost | Hostname of MQTT server |
| MqttPort | Port number of MQTT server |
| ApiCAFile | CA for Auth backend (Web site) |
| ApiURL | Base URL for Auth backend |
| ApiUsername | Username for Auth backend API access |
| ApiPassword | Password for Auth backend API access |
| Resource | Resource name - which resource users are granted permissions for |
| Mode  | "Servo", "openhigh" or "openlow"  - No door open if unset. Must set `DoorPin`|
| TagFile | Path to file to store allowed tags on local system |
| NFCdevice |  Device file of NFC reader for tags swiped in. /dev/tty for local keyboard, or /dev/ttyUSB0, etc |
| NFCmode |  Type of NFC device - see NFCmode table below |
| DoorPin |  Pin Number for Door open or servo (Usually 18). No door open if unset |
| RedLED |  "Access Deined" LED pin. (Usually 23 - No LED if Unset) |
| YellowLED |  "Servo Opening" LED pin. (Usually 25 - No LED if Unset) |
| GreenLED |  "Access Granted" LED pin. (Usually 24 - No LED if Unset) |
| LEDpipe | Filename for named pipe for LED commands |


# Neopixel Support

Neopixels are supported only through an external program to drive them. See [RPi Neopixel Tool](http://github.com/bkgoodman/rpi-neopixel-tool.git)

Coordination is necessary when starting neopixel and doorlock services, for example in systemd files, notice that one is dependent on the other:

# NFCmode
Different modes for different devices
| `10h-kbd` | for 10h (hex) keyboard device. `NFCdevce` must be a `/dev/input/event0" device for this |
| `wiegland` | External RFIDs like for doorbot. Serial Wegland protocol. Device usually `/dev/serial0` |

## Doorlock
```
[Unit]
Description=Doorlock RATT
Requires=neopixel.service

[Service]
WorkingDirectory=/home/bkg
Type=idle
User=root
Restart=always
ExecStart=/home/bkg/goratt
RestartSec=15s

[Install]
WantedBy=multi-user.target
```

## Neopixel
```
[Unit]
Description=Doorlock NeoPixel Service
After=multi-user.target

[Service]
WorkingDirectory=/home/bkg
Type=idle
User=root
Restart=always
ExecStart=/home/bkg/rpi-neopixel-tool/neotool -p /home/bkg/ledpipe -x 7 -c
RestartSec=15s

[Install]
WantedBy=multi-user.target
```

### Note
Note: If you are using a keyboard-based RFID reader, it is recommended to stop and disable the getty@tty1.service with the enclosed kbdstop service.

# Troubleshooting

If ClientID is not unique, mqtt connections will be disrupted, and ACL update messages will get lost!
