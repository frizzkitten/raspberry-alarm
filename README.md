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

But it should also run automatically on startup.