BUILD_VERSION := $(shell git branch --show-current)-$(shell date +%a-%b-%d-%Y+%I:%M%p)
LDFLAGS := -ldflags="-X main.myBuild=$(BUILD_VERSION)"

# Default: build all variants without screen support
all: goratt_x86 goratt_arm goratt_arm64

# Build all variants with screen support
all-screen: goratt_x86_screen goratt_arm_screen goratt_arm64_screen

# x86 builds
goratt_x86:
	go build -o goratt $(LDFLAGS)

goratt_x86_screen:
	go build -tags=screen -o goratt_screen $(LDFLAGS)

# ARM (32-bit) builds
goratt_arm:
	GOARCH=arm go build -o goratt_arm $(LDFLAGS)

goratt_arm_screen:
	GOARCH=arm go build -tags=screen -o goratt_arm_screen $(LDFLAGS)

# ARM64 builds
goratt_arm64:
	GOARCH=arm64 go build -o goratt_arm64 $(LDFLAGS)

goratt_arm64_screen:
	GOARCH=arm64 go build -tags=screen -o goratt_arm64_screen $(LDFLAGS)

# Build and deploy to neopi
run: goratt_arm64_screen
	ssh bkg@neopi ./beforecopy.sh
	scp goratt_arm64_screen bkg@neopi:
	#ssh bkg@neopi ./aftercopy.sh

clean:
	rm -f goratt goratt_screen goratt_arm goratt_arm_screen goratt_arm64 goratt_arm64_screen

.PHONY: all all-screen clean run goratt_x86 goratt_x86_screen goratt_arm goratt_arm_screen goratt_arm64 goratt_arm64_screen
