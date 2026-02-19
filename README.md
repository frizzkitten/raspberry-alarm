# Raspberry Alarm

A wake-up alarm that plays random songs from `~/WakeUpSongs` at 9 AM every day for 15 minutes. Press a button to dismiss early. On boot, it plays a chime so you know it's running.

I have mine set up on a Raspberry Pi with a speaker and a USB button that sends the key "a" on press, but of course configure however you please :)

## How It Works

1. On startup, plays `~/success.mp3` as a boot chime.
2. Waits until 9:00 AM (re-checking the clock every minute to handle NTP corrections).
3. Shuffles and plays songs from `~/WakeUpSongs/` for 15 minutes.
4. If dismissed via the button, plays `~/success.mp3`. If time runs out, plays `~/failure.mp3`.
5. Repeats from step 2.

If no songs are found in `~/WakeUpSongs/`, the alarm plays `~/failure.mp3` so you still hear something.

## File Setup

Place these files in your home directory (`~`):

```
~/success.mp3          # played on boot and when alarm is dismissed
~/failure.mp3          # played when alarm times out (or no songs found)
~/WakeUpSongs/         # folder of .mp3 files (add as many as you like)
    song1.mp3
    song2.mp3
    ...
```

## Dependencies

```
sudo apt install mpv
```

`espeak-ng` is also needed if you enable the robot compliments feature (see Note below):

```
sudo apt install espeak-ng
```

## Pi Setup (one-time)

Allow reading button input without root:

```
sudo usermod -aG input $USER
```

Log out and back in (or reboot) for the group change to take effect.

### Auto-start on boot

**Option A: systemd service (headless / Lite OS)**

Create `/etc/systemd/system/raspberry-alarm.service`:

```ini
[Unit]
Description=Raspberry Alarm
After=network-online.target sound.target
Wants=network-online.target

[Service]
Type=simple
User=YOUR_USERNAME
ExecStart=/home/YOUR_USERNAME/raspberry-alarm
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Then enable it:

```
sudo systemctl daemon-reload
sudo systemctl enable --now raspberry-alarm
```

**Option B: desktop autostart (Raspberry Pi OS with desktop)**

```
mkdir -p ~/.config/autostart
```

Create `~/.config/autostart/raspberry-alarm.desktop`:

```ini
[Desktop Entry]
Type=Application
Name=Raspberry Alarm
Exec=lxterminal -e /home/YOUR_USERNAME/raspberry-alarm
```

## Build & Deploy

Cross-compile for your Pi. Run `uname -m` on the Pi to determine the architecture:

| `uname -m` | Pi models                                  | Build command                                           |
|-------------|--------------------------------------------|---------------------------------------------------------|
| `aarch64`   | Pi 3/4/5, Zero 2 W (64-bit OS)            | `GOOS=linux GOARCH=arm64 go build -o raspberry-alarm .` |
| `armv7l`    | Pi 2/3/4 (32-bit OS)                       | `GOOS=linux GOARCH=arm GOARM=7 go build -o raspberry-alarm .` |
| `armv6l`    | Pi 1, Zero, Zero W                         | `GOOS=linux GOARCH=arm GOARM=6 go build -o raspberry-alarm .` |

Transfer to the Pi:

```
scp raspberry-alarm YOUR_USERNAME@PI_IP:~/
```
You can find the Pi's IP by hovering over the wifi symbol in the top right on the Pi.

## Run Manually

```
chmod +x raspberry-alarm
./raspberry-alarm
```

## Run Tests

```
go test -v ./...
```

## Note

Robot compliments are currently disabled. To enable them, uncomment the `randomCompliment()` call in `playOutro`. This requires `espeak-ng` to be installed.
