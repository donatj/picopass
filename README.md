# picopass

A tiny Go/TinyGo program for a Raspberry Pi Pico 2 W. It joins Wi-Fi, acquires
an address with DHCP, resolves `time.cloudflare.com`, performs an NTP request,
sets TinyGo's clock, then prints UTC time over USB serial once a minute.

## Set Wi-Fi credentials

Edit `wifi_secrets.go` and fill in the SSID and password for a 2.4 GHz WPA2
network. That file is ignored by Git so the password is not accidentally
committed. An empty password selects an open network. If you ever need a fresh
placeholder, `wifi_secrets.go.example` contains one.

## USB keyboard

The USB connection now appears as both its normal serial port and a standard
HID keyboard; no host driver is required. After NTP synchronization, each
button press types the current UTC time into whichever application has keyboard
focus.

The onboard LED blinks while the program is connecting and getting time, then
stays solid once NTP synchronization succeeds.

### Button wiring

With the USB connector at the top, use a normally-open momentary pushbutton to
bridge these two adjacent pins on the left-side header:

```text
physical pin 18: GND  ──┐
                        ├── momentary pushbutton ── GP14: physical pin 19
physical pin 19: GP14 ─┘
```

No external resistor is needed: the program enables GP14's internal pull-up.
Do not connect the button to 3.3 V or to physical pin 21 (which is GP16).

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
