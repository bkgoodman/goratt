all: goratt_x86 goratt_arm goratt_arm64

goratt_x86: *.go
	go build -o goratt -ldflags="-X main.myBuild="`git branch --show-current`-`date +%a-%b-%d-%Y+%I:%M%p`""

goratt_arm: *.go
	GOARCH=arm go build -o goratt_arm -ldflags="-X main.myBuild="`git branch --show-current`-`date +%a-%b-%d-%Y+%I:%M%p`""

goratt_arm64: *.go
	GOARCH=arm64 go build -o goratt_arm64 -ldflags="-X main.myBuild="`git branch --show-current`-`date +%a-%b-%d-%Y+%I:%M%p`""
