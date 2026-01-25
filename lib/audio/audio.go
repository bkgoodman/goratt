package audio

import (
	"fmt"
	"log"
	"os/exec"
	"sync"
)

// AudioManager manages audio playback.
type AudioManager struct {
	mu       sync.Mutex
	stopCh   chan struct{}
	cmd      *exec.Cmd
	format   string
	rate     int
	channels int
}

// NewAudioManager creates a new audio manager.
func NewAudioManager(format string, rate int, channels int) *AudioManager {
	return &AudioManager{
		stopCh:   make(chan struct{}),
		format:   format,
		rate:     rate,
		channels: channels,
	}
}

// Stop stops any current playback.
func (am *AudioManager) Stop() {
	am.mu.Lock()
	defer am.mu.Unlock()
	if am.cmd != nil && am.cmd.Process != nil {
		am.cmd.Process.Kill()
		am.cmd = nil
	}
}

// PlayPCM plays a PCM file. Stops any current playback.
// Uses the configured audio format, rate, and channels.
func (am *AudioManager) PlayPCM(filename string) {
	am.Stop()

	am.mu.Lock()
	am.cmd = exec.Command("aplay", "-f", am.format, "-r", fmt.Sprintf("%d", am.rate), "-c", fmt.Sprintf("%d", am.channels), "-D", "default", "-B", "1", filename)
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
