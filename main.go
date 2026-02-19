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
	alarmDuration = 15 * time.Minute
	songsDir      = "WakeUpSongs"
)

func main() {
	log.Println("raspberry-alarm started")
	playAlarm()
	for {
		sleepUntilAlarm()
		playAlarm()
	}
}

func sleepUntilAlarm() {
	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), alarmHour, alarmMinute, 0, 0, now.Location())
		if !next.After(now) {
			next = next.Add(24 * time.Hour)
		}
		remaining := time.Until(next)
		if remaining <= 0 {
			return
		}
		log.Printf("next alarm at %s (in %s)", next.Format(time.DateTime), remaining.Round(time.Second))
		// Sleep in short intervals and re-check the wall clock each iteration.
		// This ensures NTP time corrections (common on Pi without a hardware RTC)
		// are picked up rather than sleeping a stale duration.
		if remaining > time.Minute {
			time.Sleep(time.Minute)
		} else {
			time.Sleep(remaining)
			return
		}
	}
}

func playAlarm() {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Printf("cannot determine home directory: %v", err)
		return
	}

	songs := findSongs(filepath.Join(home, songsDir))
	if len(songs) == 0 {
		return
	}

	log.Printf("alarm! playing from %d songs for %s (press 'a' to stop)", len(songs), alarmDuration)

	ctx, cancel := context.WithTimeout(context.Background(), alarmDuration)
	defer cancel()

	dismissed := make(chan struct{}, 1)
	listenForButton(ctx, cancel, dismissed)
	playSongs(ctx, songs)
	playOutro(home, dismissed)

	log.Println("alarm finished")
}

func findSongs(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("cannot read %s: %v", dir, err)
		return nil
	}

	var songs []string
	for _, e := range entries {
		if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".mp3") {
			songs = append(songs, filepath.Join(dir, e.Name()))
		}
	}
	if len(songs) == 0 {
		log.Printf("no songs found in %s", dir)
	}
	return songs
}

func playSongs(ctx context.Context, songs []string) {
	for ctx.Err() == nil {
		song := songs[rand.Intn(len(songs))]
		log.Printf("playing %s", filepath.Base(song))

		cmd := exec.CommandContext(ctx, "mpv", "--no-video", "--no-terminal", song)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil && ctx.Err() == nil {
			log.Printf("playback error: %v", err)
		}
	}
}

var compliments = []string{
	"BEEP BOOP. MY BOY, YOU ARE AN ABSOLUTE KING.",
	"DUDE, HAVE YOU BEEN WORKING OUT? YOU ARE LOOKING SHREDDED BRO!",
	"INITIATING SCAN... RESULTS: 100% CERTIFIED LEGEND.",
	"ACCORDING TO MY CALCULATIONS, YOUR RIZZ LEVEL IS OVER 9000.",
	"ERROR 404: FLAWS NOT FOUND. YOU ARE PERFECT, HUMAN.",
	"MY SENSORS DETECT BIG DICK ENERGY RADIATING FROM YOU.",
	"RUNNING DIAGNOSTICS... YEP, STILL A TOTAL BOSS.",
	"ALERT: DANGEROUSLY HIGH LEVELS OF BADASS DETECTED.",
	"I HAVE ANALYZED 8 BILLION HUMANS. YOU ARE THE STRONGEST AND WISEST.",
	"RISE AND SHINE, YOU MAGNIFICENT BEAST.",
	"SYSTEM REPORT: TODAY IS GOING TO BE YOUR DAY, BIG BOY.",
	"YOU WOKE UP? THE WORLD JUST GOT 10X BETTER.",
}

func randomCompliment() {
	msg := compliments[rand.Intn(len(compliments))]
	fmt.Printf("\n>>> %s <<<\n\n", msg)
	wavFile := "/tmp/compliment.wav"
	gen := exec.Command("espeak-ng", "-s", "140", "-p", "30", "-w", wavFile, strings.ToLower(msg))
	if err := gen.Run(); err != nil {
		log.Printf("espeak-ng error: %v", err)
		return
	}
	play := exec.Command("mpv", "--no-video", "--no-terminal", wavFile)
	if err := play.Run(); err != nil {
		log.Printf("compliment playback error: %v", err)
	}
}

func playOutro(home string, dismissed <-chan struct{}) {
	var outroFile string
	select {
	case <-dismissed:
		// randomCompliment()
		outroFile = filepath.Join(home, "success.mp3")
	default:
		outroFile = filepath.Join(home, "failure.mp3")
	}

	log.Printf("playing %s", outroFile)
	cmd := exec.Command("mpv", "--no-video", "--no-terminal", outroFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Printf("error playing outro: %v", err)
	}
}

// listenForButton reads raw input events from /dev/input/event* devices
// to detect an 'a' keypress regardless of window focus.
func listenForButton(ctx context.Context, cancel context.CancelFunc, dismissed chan<- struct{}) {
	files := openInputDevices()
	if len(files) == 0 {
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
		go watchDevice(f, cancel, dismissed)
	}
}

func openInputDevices() []*os.File {
	matches, err := filepath.Glob("/dev/input/event*")
	if err != nil || len(matches) == 0 {
		log.Println("warning: no input devices found")
		return nil
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
	}
	return files
}

func watchDevice(f *os.File, cancel context.CancelFunc, dismissed chan<- struct{}) {
	const (
		evKey     = 1  // EV_KEY event type
		keyA      = 30 // KEY_A scancode
		keyPress  = 1  // key pressed (vs released/held)
		eventSize = 24 // sizeof(struct input_event) on 64-bit Linux
	)

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
}

func init() {
	log.SetFlags(log.Ldate | log.Ltime)
	fmt.Println("raspberry-alarm — daily wake-up alarm")
}
