If you make edits, rebuild for the Pi like this:

```
GOOS=linux GOARCH=arm64 go build -o raspberry-alarm main.go
```

Then transfer the new executable to the pi like this:

```
sftp frizzkitten@10.0.0.68
(enter password)
lcd go/src/raspberry-alarm
put raspberry-alarm
```

Then on the Pi open a terminal and enable execution:

```
chmod +x raspberry-alarm
```

And then you can run it like this:

```
./raspberry-alarm
```

But it should also run automatically on startup if you've done the setup for that on the Pi:

```
mkdir -p ~/.config/autostart
nano ~/.config/autostart/raspberry-alarm.desktop
```

```
[Desktop Entry]
Type=Application
Name=Raspberry Alarm
Exec=lxterminal -e /home/your-username/raspberry-alarm
```

You'll also need to give permission for the user to read inputs directly so we can read the button press:

```
sudo usermod -aG input your-username
```