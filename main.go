package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	alarmHour     = 9
	alarmMinute   = 0
	alarmDuration = 1 * time.Minute
	songsDir      = "WakeUpSongs"
)

func main() {
	log.Println("raspberry-alarm started")
	playAlarm()
	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), alarmHour, alarmMinute, 0, 0, now.Location())
		if !next.After(now) {
			next = next.Add(24 * time.Hour)
		}
		log.Printf("next alarm at %s", next.Format(time.DateTime))
		time.Sleep(time.Until(next))
		playAlarm()
	}
}

func playAlarm() {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Printf("cannot determine home directory: %v", err)
		return
	}
	dir := filepath.Join(home, songsDir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("cannot read %s: %v", dir, err)
		return
	}

	var songs []string
	for _, e := range entries {
		if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".mp3") {
			songs = append(songs, filepath.Join(dir, e.Name()))
		}
	}
	if len(songs) == 0 {
		log.Printf("no songs found in %s", dir)
		return
	}

	log.Printf("alarm! playing from %d songs for %s (press 'a' to stop)", len(songs), alarmDuration)

	ctx, cancel := context.WithTimeout(context.Background(), alarmDuration)
	defer cancel()

	// Listen for 'a' keypress on all input devices (works without window focus).
	dismissed := make(chan struct{}, 1)
	listenForButton(ctx, cancel, dismissed)

	for ctx.Err() == nil {
		song := songs[rand.Intn(len(songs))]
		log.Printf("playing %s", filepath.Base(song))

		cmd := exec.CommandContext(ctx, "mpv", "--no-video", "--no-input-terminal", song)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil && ctx.Err() == nil {
			log.Printf("playback error: %v", err)
		}
	}

	// Play a different sound depending on how the alarm ended.
	var outroFile string
	select {
	case <-dismissed:
		outroFile = filepath.Join(home, "wow.mp3")
	default:
		outroFile = filepath.Join(home, "wkuk.mp3")
	}
	log.Printf("playing %s", outroFile)
	cmd := exec.Command("mpv", "--no-video", outroFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Printf("error playing outro: %v", err)
	}

	log.Println("alarm finished")
}

// listenForButton reads raw input events from /dev/input/event* devices
// to detect an 'a' keypress regardless of window focus.
func listenForButton(ctx context.Context, cancel context.CancelFunc, dismissed chan<- struct{}) {
	const (
		evKey    = 1  // EV_KEY event type
		keyA     = 30 // KEY_A scancode
		keyPress = 1  // key pressed (vs released/held)
		eventSize = 24 // sizeof(struct input_event) on 64-bit Linux
	)

	matches, err := filepath.Glob("/dev/input/event*")
	if err != nil || len(matches) == 0 {
		log.Println("warning: no input devices found")
		return
	}

	var files []*os.File
	for _, dev := range matches {
		f, err := os.Open(dev)
		if err != nil {
			continue
		}
		files = append(files, f)
	}

	if len(files) == 0 {
		log.Println("warning: cannot open any input devices (try: sudo usermod -aG input $USER)")
		return
	}

	log.Printf("listening for button on %d input devices", len(files))

	// Close all files when context is done to unblock readers.
	go func() {
		<-ctx.Done()
		for _, f := range files {
			f.Close()
		}
	}()

	for _, f := range files {
		go func(f *os.File) {
			buf := make([]byte, eventSize)
			for {
				if _, err := io.ReadFull(f, buf); err != nil {
					return
				}
				typ := binary.LittleEndian.Uint16(buf[16:18])
				code := binary.LittleEndian.Uint16(buf[18:20])
				value := binary.LittleEndian.Uint32(buf[20:24])
				if typ == evKey && code == keyA && value == keyPress {
					log.Println("alarm dismissed by button press")
					select {
					case dismissed <- struct{}{}:
					default:
					}
					cancel()
					return
				}
			}
		}(f)
	}
}

func init() {
	log.SetFlags(log.Ldate | log.Ltime)
	fmt.Println("raspberry-alarm — daily wake-up alarm")
}
