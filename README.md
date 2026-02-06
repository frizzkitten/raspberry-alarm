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


## Build & Deploy

Rebuild for the Pi:

```
GOOS=linux GOARCH=arm64 go build -o raspberry-alarm .
```

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
