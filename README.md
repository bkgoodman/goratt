# GoRATT

Go-based RATT client 

# Build

1. Clone repo
2. `make all` (or `go build`)

For ARM (Raspberry Pi) builds:
- 32-bit: `make goratt_arm`
- 64-bit: `make goratt_arm64`

## Screen Support

To build with framebuffer video display support, use the screen targets:

```
make all-screen        # All platforms with screen support
make goratt_x86_screen # x86 with screen
make goratt_arm_screen # ARM32 with screen
make goratt_arm64_screen # ARM64 with screen
```

Or manually: `go build -tags=screen`

# Configure

1. Copy `example.cfg` to `goratt.cfg`
2. Edit as needed in your environment

See configuration section below for details.

# Run

`./goratt`

Options:
- `-cfg <file>` - Specify config file (default: `goratt.cfg`)
- `-holdopen` - Hold door open indefinitely (for testing)

# Configuration

The configuration file uses YAML format with nested sections.

## General Settings

| Parameter | Description |
| --------- | ----------- |
| `client_id` | *Unique* Client ID for Auth backend. MAC address of machine, no separators |
| `resource` | Resource name - which resource users are granted permissions for |
| `tag_file` | Path to file to store allowed tags on local system |
| `wait_secs` | How long to keep door open after access granted |
| `open_secret` | Base64 encoded SHA256 shared secret for remote open. Leave empty to disable |
| `open_tool_name` | Tool name for remote open. Leave empty to disable |

## MQTT Settings (`mqtt:`)

MQTT is **optional**. If `host` is omitted or empty, MQTT is disabled and the system operates in standalone mode (local ACL only, no remote open commands).

TLS is also **optional**. If no TLS certificates are provided, the connection uses plain TCP (default port 1883). If any TLS option is provided, the connection uses SSL (default port 8883).

| Parameter | Description |
| --------- | ----------- |
| `host` | Hostname of MQTT server. **Leave empty to disable MQTT.** |
| `port` | Port number (default: 1883 for non-TLS, 8883 for TLS) |
| `ca_cert` | Path to Root CA certificate (optional, enables TLS) |
| `client_cert` | Path to client TLS certificate (optional) |
| `client_key` | Path to client TLS key (optional) |

## API Settings (`api:`)

| Parameter | Description |
| --------- | ----------- |
| `url` | Base URL for Auth backend |
| `ca_file` | CA certificate for Auth backend |
| `username` | Username for Auth backend API access |
| `password` | Password for Auth backend API access |

## Reader Settings (`reader:`)

| Parameter | Description |
| --------- | ----------- |
| `type` | Reader type: `wiegand`, `keyboard`, or `serial` |
| `device` | Device path (e.g., `/dev/serial0`, `/dev/input/event0`) |
| `baud` | Baud rate for serial devices (default: 9600 for wiegand, 115200 for serial) |

### Reader Types

| Type | Description |
| ---- | ----------- |
| `wiegand` | Serial Wiegand protocol readers. Device usually `/dev/serial0` |
| `keyboard` | USB keyboard-style readers outputting hex digits. Device is `/dev/input/eventX` |
| `serial` | Custom serial protocol readers at 115200 baud |

## Door Settings (`door:`)

| Parameter | Description |
| --------- | ----------- |
| `type` | Door type: `servo`, `gpio_high`, `gpio_low`, or `none` |
| `pin` | GPIO pin number (usually 18) |
| `servo_open` | PWM value for open position (servo mode only) |
| `servo_close` | PWM value for closed position (servo mode only) |

### Door Types

| Type | Description |
| ---- | ----------- |
| `servo` | PWM servo control on specified pin |
| `gpio_high` | Set pin HIGH to open, LOW to close |
| `gpio_low` | Set pin LOW to open, HIGH to close |
| `none` | No door control |

## Indicator Settings (`indicator:`)

| Parameter | Description |
| --------- | ----------- |
| `green_pin` | GPIO pin for "Access Granted" LED (usually 24) |
| `yellow_pin` | GPIO pin for "Opening" LED (usually 25) |
| `red_pin` | GPIO pin for "Access Denied" LED (usually 23) |
| `neopixel_pipe` | Path to named pipe for neopixel commands |
| `video_enabled` | Enable framebuffer video display (requires screen build) |

All indicator settings are optional. Omit or set to null to disable.

**Note:** `video_enabled` requires building with `-tags=screen`. If enabled in config but not compiled in, the program will fail with an error.

# Neopixel Support

Neopixels are supported through an external program. See [RPi Neopixel Tool](http://github.com/bkgoodman/rpi-neopixel-tool.git)

When using both neopixels and the doorlock service, ensure proper startup ordering in systemd:

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
