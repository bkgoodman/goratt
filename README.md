# GoRATT

Go-based RATT client suite supporting access control, vending, darkroom management, and label printing.

## Applications

- **Doorlock** (`cmd/doorlock`): Door access control with RFID badge reader, servo/GPIO strike, optional status LEDs, and framebuffer video display.
- **Label Printer** (`cmd/labelprinter`): Dymo thermal label printer controller for member temporary storage passes, free table passes, and backend ArUco bin labels.
- **Vending** (`cmd/vending`): Interactive vending machine controller with screen display and audio prompts.
- **Darkroom** (`cmd/darkroom`): Darkroom timer and occupancy management with screen display.

# Build

1. Clone repo
2. Run `make` for target platform:

```bash
# Build everything
make all

# Architecture bundles
make x86     # Build all binaries for x86_64
make arm     # Build all binaries for ARM32 (Raspberry Pi OS 32-bit)
make arm64   # Build all binaries for ARM64 (Raspberry Pi OS 64-bit)

# Label Printer targets (headless)
make labelprinter_x86
make labelprinter_arm
make labelprinter_arm64

# Doorlock targets
make doorlock_x86          # Headless doorlock
make doorlock_arm
make doorlock_arm64
make doorlock_x86_screen   # Doorlock with framebuffer video display
make doorlock_arm_screen
make doorlock_arm64_screen

# Vending & Darkroom targets (require screen support)
make vending_x86_screen
make vending_arm_screen
make vending_arm64_screen
make darkroom_x86_screen
make darkroom_arm_screen
make darkroom_arm64_screen
```

## Screen Support

Vending and Darkroom always require screen support (`-tags=screen`). Doorlock can be built either headless or with screen support (`*_screen` targets). The Label Printer does not require screen support.

# Configure

1. Copy `example.cfg` to `goratt.cfg`
2. Edit as needed in your environment

See configuration section below for details.

# Run

### Doorlock
```bash
./doorlock_arm -cfg goratt.cfg
```

Options:
- `-cfg <file>` - Specify config file (default: `goratt.cfg`)
- `-holdopen` - Hold door open indefinitely (for testing)

### Label Printer
```bash
./labelprinter_arm -cfg goratt.cfg
```

Options:
- `-cfg <file>` - Specify config file (default: `goratt.cfg`)

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
| `device` | Device path (e.g., `/dev/serial0` for serial, `/dev/input/event0` for keyboard) |
| `baud` | Baud rate for serial devices (default: 9600 for wiegand, 115200 for serial) |
| `format` | For keyboard readers: digit format (see below) |

### Reader Types

| Type | Description |
| ---- | ----------- |
| `wiegand` | Serial Wiegand protocol readers. Device usually `/dev/serial0` |
| `keyboard` | USB keyboard-style readers outputting digits. Device is `/dev/input/eventX` |
| `serial` | Custom serial protocol readers at 115200 baud |

### Keyboard Format

The `format` option specifies how keyboard input is parsed. Format is `<digits><base>` where:
- `<digits>` is the expected number of digits (0 for any length)
- `<base>` is `h` for hexadecimal or `d` for decimal

| Format | Description |
| ------ | ----------- |
| `10h` | 10 hex digits (default) |
| `10d` | 10 decimal digits |
| `8h` | 8 hex digits |
| `8d` | 8 decimal digits |

Example for a reader outputting 10 decimal digits:
```yaml
reader:
  type: keyboard
  device: /dev/input/event0
  format: 10d
```

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

All indicator settings are optional. Omit or set to null to disable.

## Label Printer Settings (`labelprinter:`)

The label printer service (`labelprinter`) interfaces with a Dymo USB label printer (e.g., LabelWriter 450) and a 2-button / 2-LED control pad to issue member labels. It also hosts an HTTP API for backend ArUco label print requests.

### Configuration Options

| Parameter | Description | Default |
| --------- | ----------- | ------- |
| `device` | Path to USB printer device | `/dev/usb/lp0` |
| `http_port` | HTTP port for incoming backend print requests | `8080` |
| `button1_pin` | GPIO pin for Button 1 (Free Table label, 8 days) | `18` |
| `button2_pin` | GPIO pin for Button 2 (Storage Pass label, 3 days) | `25` |
| `led1_pin` | GPIO pin for Button 1 LED | `23` |
| `led2_pin` | GPIO pin for Button 2 LED | `24` |
| `print_on_boot` | Print ready label (`- Ready -`) on startup | `true` |
| `personality` | Personality topic name for MQTT status reporting | `storagepass` |

Example configuration:

```yaml
labelprinter:
  device: "/dev/usb/lp0"
  http_port: 8080
  button1_pin: 18
  button2_pin: 25
  led1_pin: 23
  led2_pin: 24
  print_on_boot: false
  personality: "storagepass"
```

### Hardware Buttons & Flow

The control pad uses two physical buttons (configured with internal pull-ups) and two LEDs:
- **Button 1**: Free Table pass (8 days validity; "Items will be discarded after specified date")
- **Button 2**: Temporary Storage pass (3 days validity; "Items may be discarded and disposal charges may be incurred...")

Users can interact in either order:
1. **Badge First, Then Select**:
   - Tap an authorized RFID badge.
   - The LEDs alternate blinking for up to 5 seconds.
   - Press **Button 1** or **Button 2** to print the requested label with the member name and expiration date.
2. **Select First, Then Badge**:
   - Press **Button 1** or **Button 2**; the chosen button's LED illuminates.
   - Tap an authorized RFID badge within 5 seconds to authenticate and print.

If no action is completed within the 5-second window, the state times out and returns to idle. Printed passes are logged and published to MQTT under `ratt/status/node/<client_id>/personality/<personality>`.

### HTTP ArUco Label Print API

The service exposes an HTTP server on `http_port` (default `8080`) for printing ArUco bin labels from backends (such as `authbackend`):

- **Health check**:
  - `GET /health`
  - Returns `{"status":"ok"}` with HTTP 200.
- **Print ArUco label**:
  - `POST /` or `POST /print_label`
  - Content-Type: `application/x-www-form-urlencoded`
  - Required Form Fields:
    - `name`: Member name (e.g. `Jane Doe` or `Jane.Doe`)
    - `number`: ArUco marker index (integer between 0 and 1023)
  - Example:
    ```bash
    curl -X POST http://localhost:8080/ \
      -d "name=Jane.Doe" \
      -d "number=42"
    ```
  - Successful response:
    ```json
    {"status":"ok","member":"Jane.Doe","number":42}
    ```
  - Requests missing `name` or `number`, non-POST requests, or invalid indices are rejected with HTTP 4xx to prevent unintentional blank label printing.

### Permissions & Device Access

The user running `labelprinter` must have read/write access to the USB printer character device (usually `/dev/usb/lp0`, typically by adding the user to the `lp` group or running as `root`) and access to the GPIO character device (`/dev/gpiochip*`).

## Video Display Settings

| Parameter | Description |
| --------- | ----------- |
| `video_enabled` | Enable framebuffer video display (requires screen build) |

**Note:** `video_enabled` is a top-level config option (not under `indicator:`). Requires building with `-tags=screen`. If enabled in config but not compiled in, the program will fail with an error.

The video display uses a stateful screen system that can handle events like button presses, rotary encoder input, and RFID swipes.

## Video Settings (`video:`)

| Parameter | Description |
| --------- | ----------- |
| `rotation` | Display rotation in degrees: `0`, `90`, `180`, or `270` |

Example:
```yaml
video:
  rotation: 180
```

The rotation is applied in software - the offscreen buffer is sized for the rotated dimensions, and pixels are transformed when blitting to the framebuffer. This works independently of any hardware rotation settings.

- **0**: No rotation (default)
- **90**: 90° clockwise
- **180**: Upside down
- **270**: 270° clockwise (90° counter-clockwise)

## Rotary Encoder Settings (`rotary:`)

| Parameter | Description |
| --------- | ----------- |
| `chip` | GPIO chip name (default: `gpiochip0`) |
| `clk_pin` | GPIO pin for CLK signal |
| `dt_pin` | GPIO pin for DT signal |
| `button_pin` | GPIO pin for button (optional, 0 to disable) |

Example:
```yaml
rotary:
  clk_pin: 17
  dt_pin: 27
  button_pin: 22
```

Rotary encoder events are sent to the current screen's `HandleEvent` method. The rotary encoder is independent of the video display - you can use it without a screen.

## Event Pipe (`event_pipe:`)

The event pipe allows external programs or shell scripts to inject events into the system for testing or integration.

| Parameter | Description |
| --------- | ----------- |
| `path` | Path to named pipe (e.g., `/tmp/goratt-events`) |

Example:
```yaml
event_pipe:
  path: /tmp/goratt-events
```

### Pipe Commands

Commands are sent as plain text lines. The pipe can be opened/closed by writers - the program handles reconnection automatically.

| Command | Description |
| ------- | ----------- |
| `rfid <tagid>` | Simulate RFID tag swipe (decimal or hex) |
| `tag <tagid>` | Alias for `rfid` |
| `rotary <delta>` | Simulate rotary turn (+1 for CW, -1 for CCW) |
| `rotary press` | Simulate rotary button press |
| `pin <name> <0\|1>` | Simulate pin state change (0=released, 1=pressed) |

**Pin Names:** `button1`, `button2`, `sensor1`, `sensor2`, `estop`, `door`, `safelight`, `activity`, `enable`

### Usage Examples

```bash
# Create pipe (done automatically by program)
# mkfifo /tmp/goratt-events

# Simulate tag swipe (decimal)
echo "rfid 1234567890" > /tmp/goratt-events

# Simulate tag swipe (hex)
echo "tag 0x499602D2" > /tmp/goratt-events

# Simulate rotary turn clockwise
echo "rotary 1" > /tmp/goratt-events

# Simulate rotary button press
echo "rotary press" > /tmp/goratt-events

# Simulate button press
echo "pin button1 1" > /tmp/goratt-events

# Simulate button release
echo "pin button1 0" > /tmp/goratt-events
```

Lines starting with `#` are treated as comments and ignored.

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

## Label Printer
```ini
[Unit]
Description=GoRATT Label Printer Service
After=network.target

[Service]
WorkingDirectory=/home/bkg
Type=idle
User=root
Restart=always
ExecStart=/home/bkg/goratt/labelprinter_arm -cfg /home/bkg/goratt/goratt.cfg
RestartSec=15s

[Install]
WantedBy=multi-user.target
```

### Note
Note: If you are using a keyboard-based RFID reader, it is recommended to stop and disable the getty@tty1.service with the enclosed kbdstop service.

# Troubleshooting

If ClientID is not unique, mqtt connections will be disrupted, and ACL update messages will get lost!
