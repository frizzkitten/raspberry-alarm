# Raspberry Alarm

A wake-up alarm that plays random songs from `~/WakeUpSongs` at 9AM every day for 15 minutes. Press a button to dismiss early and get a robot compliment.

## Pi Dependencies

```
sudo apt install mpv espeak-ng
```

## Pi Setup (one-time)

Allow reading button input:

```
sudo usermod -aG input pi-username
```

Auto-start on boot:

```
mkdir -p ~/.config/autostart
nano ~/.config/autostart/raspberry-alarm.desktop
```

```ini
[Desktop Entry]
Type=Application
Name=Raspberry Alarm
Exec=lxterminal -e /home/pi-username/raspberry-alarm
```

Then make a folder in the root directory of the Pi called WakeUpSongs. Inside it, put as many .mp3 files as you'd like!

Then add a file called success.mp3 and another called failure.mp3 to the root directory. success.mp3 will play when the alarm is stopped via input, and failure.mp3 will play if the songs are not stopped within the allotted time.


## Build & Deploy

Rebuild for the Pi:

```
GOOS=linux GOARCH=arm64 go build -o raspberry-alarm .
```

(You may need to have a different GOARCH depending on the version of your Pi.)

Transfer to the Pi:

```
sftp pi-username@pi-ip
lcd go/src/raspberry-alarm
put raspberry-alarm
```

You can find the Pi's IP by hovering over the wifi symbol in the top right on the Pi.

## Run Manually

```
chmod +x raspberry-alarm
./raspberry-alarm
```

## Note

I have the compliments turned off at the moment. You can turn them back on by uncommenting `randomCompliment()`.