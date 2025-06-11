# GoRATT

Go-based RATT client 

# Build

1. Clone repo
2. `go build`

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
| Mode  | "Servo", "openhigh" or "openlow" |
| TagFile | Path to file to store allowed tags on local system |
| NFCdevice |  Device file of NFC reader for tags swiped in. /dev/tty for local keyboard, or /dev/ttyUSB0, etc |
| NFCmode |  Use "10h-kbd" for 10h (hex) keyboard device. `NFCdevce` must be a `/dev/input/event0" device for this |
| DoorPin |  Pin Number for Door open or servo (Defaults to 18) |
| LEDpipe | Filename for named pipe for LED commands |


# Neopixel Support

Neopixels are supported only through an external program to drive them. See [RPi Neopixel Tool](http://github.com/bkgoodman/rpi-neopixel-tool.git)

Coordination is necessary when starting neopixel and doorlock services, for example in systemd files, notice that one is dependent on the other:

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


# Troubleshooting

If ClientID is not unique, mqtt connections will be disrupted, and ACL update messages will get lost!
