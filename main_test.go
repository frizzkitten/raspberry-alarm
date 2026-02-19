package main

import (
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// nextAlarmTime
// ---------------------------------------------------------------------------

func TestNextAlarmTime(t *testing.T) {
	loc := time.Local

	tests := []struct {
		name string
		now  time.Time
		want time.Time
	}{
		{
			name: "before alarm, same day",
			now:  time.Date(2025, 6, 15, 7, 30, 0, 0, loc),
			want: time.Date(2025, 6, 15, alarmHour, alarmMinute, 0, 0, loc),
		},
		{
			name: "after alarm, next day",
			now:  time.Date(2025, 6, 15, 10, 0, 0, 0, loc),
			want: time.Date(2025, 6, 16, alarmHour, alarmMinute, 0, 0, loc),
		},
		{
			name: "exactly at alarm time, next day",
			now:  time.Date(2025, 6, 15, alarmHour, alarmMinute, 0, 0, loc),
			want: time.Date(2025, 6, 16, alarmHour, alarmMinute, 0, 0, loc),
		},
		{
			name: "one second before alarm",
			now:  time.Date(2025, 6, 15, 8, 59, 59, 0, loc),
			want: time.Date(2025, 6, 15, alarmHour, alarmMinute, 0, 0, loc),
		},
		{
			name: "midnight",
			now:  time.Date(2025, 6, 15, 0, 0, 0, 0, loc),
			want: time.Date(2025, 6, 15, alarmHour, alarmMinute, 0, 0, loc),
		},
		{
			name: "end of year rolls to next year",
			now:  time.Date(2025, 12, 31, 23, 59, 0, 0, loc),
			want: time.Date(2026, 1, 1, alarmHour, alarmMinute, 0, 0, loc),
		},
		{
			name: "one nanosecond before alarm",
			now:  time.Date(2025, 6, 15, alarmHour, alarmMinute, 0, -1, loc),
			want: time.Date(2025, 6, 15, alarmHour, alarmMinute, 0, 0, loc),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nextAlarmTime(tt.now)
			if !got.Equal(tt.want) {
				t.Errorf("nextAlarmTime(%v) = %v, want %v", tt.now, got, tt.want)
			}
		})
	}
}

func TestNextAlarmTimeDST(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skip("timezone America/New_York not available")
	}

	// 2025 spring forward: March 9 at 2:00 AM -> 3:00 AM (23-hour day)
	t.Run("spring forward", func(t *testing.T) {
		now := time.Date(2025, 3, 8, 22, 0, 0, 0, loc)
		got := nextAlarmTime(now)
		want := time.Date(2025, 3, 9, alarmHour, alarmMinute, 0, 0, loc)
		if !got.Equal(want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	// 2025 fall back: November 2 at 2:00 AM -> 1:00 AM (25-hour day)
	t.Run("fall back", func(t *testing.T) {
		now := time.Date(2025, 11, 1, 22, 0, 0, 0, loc)
		got := nextAlarmTime(now)
		want := time.Date(2025, 11, 2, alarmHour, alarmMinute, 0, 0, loc)
		if !got.Equal(want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})
}

// ---------------------------------------------------------------------------
// findSongs
// ---------------------------------------------------------------------------

func TestFindSongs(t *testing.T) {
	t.Run("nonexistent directory", func(t *testing.T) {
		songs := findSongs("/nonexistent/path/that/does/not/exist")
		if songs != nil {
			t.Errorf("expected nil, got %v", songs)
		}
	})

	t.Run("empty directory", func(t *testing.T) {
		songs := findSongs(t.TempDir())
		if len(songs) != 0 {
			t.Errorf("expected 0 songs, got %d", len(songs))
		}
	})

	t.Run("only non-mp3 files", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "readme.txt"), nil, 0644)
		os.WriteFile(filepath.Join(dir, "photo.jpg"), nil, 0644)
		songs := findSongs(dir)
		if len(songs) != 0 {
			t.Errorf("expected 0 songs, got %d", len(songs))
		}
	})

	t.Run("finds mp3 files", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "song1.mp3"), nil, 0644)
		os.WriteFile(filepath.Join(dir, "song2.mp3"), nil, 0644)
		songs := findSongs(dir)
		if len(songs) != 2 {
			t.Errorf("expected 2 songs, got %d", len(songs))
		}
	})

	t.Run("filters mixed files to mp3 only", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "song.mp3"), nil, 0644)
		os.WriteFile(filepath.Join(dir, "notes.txt"), nil, 0644)
		os.WriteFile(filepath.Join(dir, "image.png"), nil, 0644)
		songs := findSongs(dir)
		if len(songs) != 1 {
			t.Fatalf("expected 1 song, got %d", len(songs))
		}
		if filepath.Base(songs[0]) != "song.mp3" {
			t.Errorf("expected song.mp3, got %s", filepath.Base(songs[0]))
		}
	})

	t.Run("case insensitive extension", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "upper.MP3"), nil, 0644)
		os.WriteFile(filepath.Join(dir, "mixed.Mp3"), nil, 0644)
		songs := findSongs(dir)
		if len(songs) != 2 {
			t.Errorf("expected 2 songs, got %d", len(songs))
		}
	})

	t.Run("skips subdirectories with mp3 name", func(t *testing.T) {
		dir := t.TempDir()
		os.Mkdir(filepath.Join(dir, "subdir.mp3"), 0755)
		os.WriteFile(filepath.Join(dir, "song.mp3"), nil, 0644)
		songs := findSongs(dir)
		if len(songs) != 1 {
			t.Errorf("expected 1 song, got %d", len(songs))
		}
	})

	t.Run("returns full paths", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "song.mp3"), nil, 0644)
		songs := findSongs(dir)
		if len(songs) != 1 {
			t.Fatalf("expected 1 song, got %d", len(songs))
		}
		want := filepath.Join(dir, "song.mp3")
		if songs[0] != want {
			t.Errorf("expected %s, got %s", want, songs[0])
		}
	})
}

// ---------------------------------------------------------------------------
// watchDevice
// ---------------------------------------------------------------------------

// makeInputEvent builds a fake 24-byte Linux input_event struct.
func makeInputEvent(typ uint16, code uint16, value uint32) []byte {
	buf := make([]byte, 24)
	binary.LittleEndian.PutUint16(buf[16:18], typ)
	binary.LittleEndian.PutUint16(buf[18:20], code)
	binary.LittleEndian.PutUint32(buf[20:24], value)
	return buf
}

const (
	testEvKey      = 1
	testKeyA       = 30
	testKeyB       = 48
	testKeyPress   = 1
	testKeyRelease = 0
)

func TestWatchDevice(t *testing.T) {
	t.Run("KEY_A press dismisses and cancels", func(t *testing.T) {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		defer r.Close()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		dismissed := make(chan struct{}, 1)

		w.Write(makeInputEvent(testEvKey, testKeyA, testKeyPress))
		w.Close()

		watchDevice(r, cancel, dismissed)

		select {
		case <-dismissed:
		default:
			t.Error("expected dismissed signal")
		}
		if ctx.Err() == nil {
			t.Error("expected context to be cancelled")
		}
	})

	t.Run("KEY_A release is ignored", func(t *testing.T) {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		defer r.Close()

		_, cancel := context.WithCancel(context.Background())
		defer cancel()
		dismissed := make(chan struct{}, 1)

		w.Write(makeInputEvent(testEvKey, testKeyA, testKeyRelease))
		w.Close()

		watchDevice(r, cancel, dismissed)

		select {
		case <-dismissed:
			t.Error("should not dismiss on key release")
		default:
		}
	})

	t.Run("other key press is ignored", func(t *testing.T) {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		defer r.Close()

		_, cancel := context.WithCancel(context.Background())
		defer cancel()
		dismissed := make(chan struct{}, 1)

		w.Write(makeInputEvent(testEvKey, testKeyB, testKeyPress))
		w.Close()

		watchDevice(r, cancel, dismissed)

		select {
		case <-dismissed:
			t.Error("should not dismiss on non-A key")
		default:
		}
	})

	t.Run("non-key event type is ignored", func(t *testing.T) {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		defer r.Close()

		_, cancel := context.WithCancel(context.Background())
		defer cancel()
		dismissed := make(chan struct{}, 1)

		w.Write(makeInputEvent(0x03, testKeyA, testKeyPress)) // EV_ABS
		w.Close()

		watchDevice(r, cancel, dismissed)

		select {
		case <-dismissed:
			t.Error("should not dismiss on non-EV_KEY event")
		default:
		}
	})

	t.Run("skips non-matching events then matches", func(t *testing.T) {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		defer r.Close()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		dismissed := make(chan struct{}, 1)

		w.Write(makeInputEvent(testEvKey, testKeyB, testKeyPress))    // wrong key
		w.Write(makeInputEvent(testEvKey, testKeyA, testKeyRelease))  // release, not press
		w.Write(makeInputEvent(0x03, testKeyA, testKeyPress))         // wrong event type
		w.Write(makeInputEvent(testEvKey, testKeyA, testKeyPress))    // match
		w.Close()

		watchDevice(r, cancel, dismissed)

		select {
		case <-dismissed:
		default:
			t.Error("expected dismissed after skipping non-matching events")
		}
		if ctx.Err() == nil {
			t.Error("expected context to be cancelled")
		}
	})

	t.Run("pipe closed returns without dismissing", func(t *testing.T) {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		defer r.Close()

		_, cancel := context.WithCancel(context.Background())
		defer cancel()
		dismissed := make(chan struct{}, 1)

		w.Close() // immediate EOF

		watchDevice(r, cancel, dismissed)

		select {
		case <-dismissed:
			t.Error("should not dismiss on EOF")
		default:
		}
	})
}

// ---------------------------------------------------------------------------
// playSongs — just verify it respects a cancelled context
// ---------------------------------------------------------------------------

func TestPlaySongsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Should return immediately without trying to exec mpv.
	done := make(chan struct{})
	go func() {
		playSongs(ctx, []string{"/fake/song.mp3"})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("playSongs did not return promptly on cancelled context")
	}
}
