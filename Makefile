BUILD_VERSION := $(shell git branch --show-current)-$(shell date +%a-%b-%d-%Y+%I:%M%p)
LDFLAGS := -ldflags="-X main.myBuild=$(BUILD_VERSION)"
 
# Vending and Darkroom ALWAYS require screen support.
# Doorlock can be built either with or without screen support.
 
# Screen-enabled targets (for all apps)
SCREEN_TARGETS := vending_x86_screen darkroom_x86_screen doorlock_x86_screen \
                  vending_arm_screen  darkroom_arm_screen  doorlock_arm_screen \
                  vending_arm64_screen darkroom_arm64_screen doorlock_arm64_screen
 
# No-screen targets (Doorlock only)
NOSCREEN_TARGETS := doorlock_x86 doorlock_arm doorlock_arm64
 
all: $(NOSCREEN_TARGETS) $(SCREEN_TARGETS)
 
# Architecture Groups
x86: doorlock_x86 doorlock_x86_screen vending_x86_screen darkroom_x86_screen
arm: doorlock_arm doorlock_arm_screen vending_arm_screen darkroom_arm_screen
arm64: doorlock_arm64 doorlock_arm64_screen vending_arm64_screen darkroom_arm64_screen
 
# --- x86 Builds ---
doorlock_x86:
	go build $(LDFLAGS) -o doorlock_x86 ./cmd/doorlock
 
doorlock_x86_screen:
	go build -tags=screen $(LDFLAGS) -o doorlock_x86_screen ./cmd/doorlock
 
vending_x86_screen:
	go build -tags=screen $(LDFLAGS) -o vending_x86_screen ./cmd/vending
 
darkroom_x86_screen:
	go build -tags=screen $(LDFLAGS) -o darkroom_x86_screen ./cmd/darkroom
 
# --- ARM (32-bit) Builds ---
doorlock_arm:
	GOARCH=arm go build $(LDFLAGS) -o doorlock_arm ./cmd/doorlock
 
doorlock_arm_screen:
	GOARCH=arm go build -tags=screen $(LDFLAGS) -o doorlock_arm_screen ./cmd/doorlock
 
vending_arm_screen:
	GOARCH=arm go build -tags=screen $(LDFLAGS) -o vending_arm_screen ./cmd/vending
 
darkroom_arm_screen:
	GOARCH=arm go build -tags=screen $(LDFLAGS) -o darkroom_arm_screen ./cmd/darkroom
 
# --- ARM64 Builds ---
doorlock_arm64:
	GOARCH=arm64 go build $(LDFLAGS) -o doorlock_arm64 ./cmd/doorlock
 
doorlock_arm64_screen:
	GOARCH=arm64 go build -tags=screen $(LDFLAGS) -o doorlock_arm64_screen ./cmd/doorlock
 
vending_arm64_screen:
	GOARCH=arm64 go build -tags=screen $(LDFLAGS) -o vending_arm64_screen ./cmd/vending
 
darkroom_arm64_screen:
	GOARCH=arm64 go build -tags=screen $(LDFLAGS) -o darkroom_arm64_screen ./cmd/darkroom
 
# Build and deploy to neopi
run-vending: vending_arm64_screen
	ssh bkg@neopi ./beforecopy.sh
	scp vending_arm64_screen bkg@neopi:vending

run-darkroom: darkroom_arm64_screen
	ssh bkg@neopi ./beforecopy.sh
	scp darkroom_arm64_screen bkg@neopi:darkroom

run-doorlock: doorlock_arm64_screen
	ssh bkg@neopi ./beforecopy.sh
	scp doorlock_arm64_screen bkg@neopi:doorlock
 
clean:
	rm -f vending_* doorlock_* darkroom_*
 
.PHONY: all x86 arm arm64 clean run $(SCREEN_TARGETS) $(NOSCREEN_TARGETS)
