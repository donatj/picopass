# picopass

A tiny Go/TinyGo program for a Raspberry Pi Pico 2 W. It joins Wi-Fi, acquires
an address with DHCP, resolves `time.cloudflare.com`, performs an NTP request,
sets TinyGo's clock, then prints UTC time over USB serial once a minute.

## Set Wi-Fi credentials and password

Edit `secrets.go` and fill in the SSID and password for a 2.4 GHz WPA2 network,
the password to type, and `totpSeed`. `totpSeed` is the Base32 secret supplied
by the service for authenticator apps, not the QR-code image. This project
intentionally keeps `secrets.go` under version control; `secrets.go.example`
retains the placeholders. An empty Wi-Fi password selects an open network.

## USB keyboard

The USB connection now appears as both its normal serial port and a standard
HID keyboard; no host driver is required. After NTP synchronization, the time
button types the current UTC time plus Return, while the password button types
`systemPassword` exactly as stored, without Return. The TOTP button generates
and types the current six-digit, 30-second code from `totpSeed`, also without
Return. The password, seed, and code are never written to the serial log.

The onboard LED blinks while the program is connecting and getting time, then
stays solid once NTP synchronization succeeds.

### Button wiring

With the USB connector at the top, connect one leg of each normally-open
momentary pushbutton to a shared GND rail, then connect the other legs to:

```text
physical pin 18: GND   ── shared ground rail
physical pin 19: GP14  ── time button
physical pin 20: GP15  ── system-password button
physical pin 21: GP16  ── TOTP button
```

No external resistors are needed: the program enables GP14, GP15, and GP16's
internal pull-ups. Do not connect any button to 3.3 V. This uses the host's
keyboard layout; a US layout is safest if the password includes punctuation.

For future input actions, call `typeText("your message\\n")` from the program.
It sends ordinary text with a small pacing delay, so longer messages are not
dropped. HID key combinations are also available through `hidKeyboard.Down`
and `hidKeyboard.Up`.

## Flash and monitor

Install a current TinyGo toolchain, then from this directory run:

```sh
go mod download
tinygo flash -target=pico2-w -scheduler=tasks -stack-size=8kb -monitor .
```

Hold the Pico 2 W's BOOTSEL button while connecting USB for its first flash if
the UF2 drive does not appear automatically. The USB serial output reports the
DHCP address and the synchronized UTC time.

`main.go` uses the Pico 2 W target and the CYW43439 driver supported by TinyGo.
