package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"golang.org/x/term"
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
		if !e.IsDir() {
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

	// Put terminal in raw mode to capture individual keypresses.
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		log.Printf("warning: cannot set raw terminal mode: %v", err)
	} else {
		defer term.Restore(int(os.Stdin.Fd()), oldState)
	}

	// Listen for 'a' keypress in background.
	go func() {
		buf := make([]byte, 1)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil || n == 0 {
				return
			}
			if buf[0] == 'a' || buf[0] == 'A' {
				log.Println("\nalarm dismissed by keypress")
				cancel()
				return
			}
		}
	}()

	for ctx.Err() == nil {
		song := songs[rand.Intn(len(songs))]
		log.Printf("playing %s", filepath.Base(song))

		cmd := exec.CommandContext(ctx, "mpv", "--no-video", song)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil && ctx.Err() == nil {
			log.Printf("playback error: %v", err)
		}
	}

	log.Println("alarm finished")
}

func init() {
	log.SetFlags(log.Ldate | log.Ltime)
	fmt.Println("raspberry-alarm — daily wake-up alarm")
}
