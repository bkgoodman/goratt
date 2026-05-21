package audio

import (
	"fmt"
	"log"
	"os/exec"
	"sync"
)

// Params defines the audio format parameters.
type Params struct {
	Format   string
	Rate     int
	Channels int
	Type	 string
}

// AudioManager manages audio playback.
type AudioManager struct {
	mu       sync.Mutex
	stopCh   chan struct{}
	cmd      *exec.Cmd
	format   string
	rate     int
	channels int
	device   string
	typ	 string
}

// NewAudioManager creates a new audio manager.
func NewAudioManager(params Params, device string) *AudioManager {
	if device == "default" {
		device = ""
	}
	return &AudioManager{
		stopCh:   make(chan struct{}),
		format:   params.Format,
		rate:     params.Rate,
		channels: params.Channels,
		typ:	  params.Type,
		device:   device,
	}
}

// Stop stops any current playback.
func (am *AudioManager) Stop() {
	if am == nil {
		return
	}
	am.mu.Lock()
	defer am.mu.Unlock()
	if am.cmd != nil && am.cmd.Process != nil {
		am.cmd.Process.Kill()
		am.cmd = nil
	}
}

// PlayBuffer plays PCM data from a byte slice. Stops any current playback.
func (am *AudioManager) PlayBuffer(data []byte) {
	am.Stop()
 
	am.mu.Lock()
	// Use '-' to read from stdin, or just omit the filename argument for aplay to read from stdin by default
	// The -B 1 reduces buffer time to minimize latency
	args := []string{"-f", am.format, "-r", fmt.Sprintf("%d", am.rate), "-t", am.typ, "-c", fmt.Sprintf("%d", am.channels)}
	if am.device != "" {
		args = append(args, "-D", am.device)
	}
	am.cmd = exec.Command("aplay", args...)
	cmd := am.cmd
 
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Printf("Audio: failed to get stdin pipe: %v", err)
		am.cmd = nil
		am.mu.Unlock()
		return
	}
	am.mu.Unlock()
 
	if err := cmd.Start(); err != nil {
		log.Printf("Audio: aplay start error: %v", err)
		am.mu.Lock()
		am.cmd = nil
		am.mu.Unlock()
		return
	}
 
	// Write data to stdin in a goroutine to avoid blocking
	go func() {
		defer stdin.Close()
		if _, err := stdin.Write(data); err != nil {
			// This might happen if aplay is killed (e.g. Stop() called), so generic logging is fine
			// log.Printf("Audio: write to stdin failed: %v", err)
		}
	}()
 
	// Wait for completion in background
	go func() {
		defer func() {
			am.mu.Lock()
			// Only clear if it's still our command
			if am.cmd == cmd {
				am.cmd = nil
			}
			am.mu.Unlock()
		}()
		_ = cmd.Wait()
	}()
}
 
// PlayPCM plays a PCM file. Stops any current playback.
// Uses the configured audio format, rate, and channels.
func (am *AudioManager) PlayPCM(filename string) {
	am.Stop()

	am.mu.Lock()
	args := []string{"-f", am.format, "-r", fmt.Sprintf("%d", am.rate), "-c", fmt.Sprintf("%d", am.channels)}
	if am.device != "" {
		args = append(args, "-D", am.device)
	}
	args = append(args, filename)
	am.cmd = exec.Command("aplay", args...)
	cmd := am.cmd
	am.mu.Unlock()

	err := cmd.Start()
	if err != nil {
		log.Printf("Audio: aplay start error: %v", err)
		am.mu.Lock()
		am.cmd = nil
		am.mu.Unlock()
		return
	}

	go func() {
		defer func() {
			am.mu.Lock()
			am.cmd = nil
			am.mu.Unlock()
		}()

		err := cmd.Wait()
		if err != nil {
			log.Printf("Audio: aplay wait error: %v", err)
		}
	}()
}
