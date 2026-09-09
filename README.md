# picopass

<img width="200" align="right" alt="Example Device" src="https://github.com/user-attachments/assets/505ece17-1a36-4fbd-b79b-e4ec3843afc2" />

`picopass` is a TinyGo toy project that turns a Raspberry Pi Pico 2 W into a
physical TOTP keyboard. It gets time over Wi-Fi and NTP, then presents itself
over USB as both a serial port and an HID keyboard. Three physical buttons type
the current UTC time, a stored password, or a freshly generated TOTP code.

## Not a secure authenticator

**This is an experiment, not a security product. Do not use it for real
accounts or in a secure environment.**

It intentionally keeps Wi-Fi credentials, a system password, and the TOTP seed
as plaintext in the committed `secrets.go` file. It has no encryption, secure
element, user verification, tamper resistance, access control, lockout, secure
updates, or security review. Anyone who can access the device, its firmware,
the repository, or the computer currently receiving the HID input may be able
to use or copy those credentials. HID typing can also go to the wrong focused
application. Use only disposable test credentials you are comfortable exposing.

For anything that actually needs protection, you are far better off using a
purpose-built, security-reviewed hardware authenticator such as a
[YubiKey](https://www.yubico.com/products/yubikey-5-overview/) or a similar
FIDO2/WebAuthn security key.

## What it does

- Connects to Wi-Fi, obtains an address with DHCP, resolves
  `time.cloudflare.com`, and sets TinyGo's clock with NTP.
- Turns its onboard LED solid once the time sync succeeds.
- Provides a USB serial log and USB HID keyboard at the same time.
- Types the configured password or a six-digit TOTP code when its corresponding
  button is pressed.

## Set Wi-Fi credentials and password

Edit `secrets.go` and fill in the SSID and password for a 2.4 GHz WPA2 network,
the password to type, and `totpSeed`. `totpSeed` is the Base32 secret supplied
by the service for authenticator apps, not the QR-code image. This project
intentionally keeps `secrets.go` under version control and starts it with
placeholders to replace. An empty Wi-Fi password selects an open network.

## USB keyboard

The USB connection now appears as both its normal serial port and a standard
HID keyboard; no host driver is required. After NTP synchronization, the time
button types the current UTC time without Return, while the password button types
`systemPassword` exactly as stored, without Return. The TOTP button generates
and types the current six-digit, 30-second code from `totpSeed`, also without
Return. The password, seed, and code are never written to the serial log.

Press the same button again within one second to type only Return (`\n`)
instead of its usual value. This applies to all three buttons. Pressing a
different button starts a new sequence. After a double press, the next press
types its usual value again. Each press requires releasing the button first;
the one-second window is measured between detected presses.

The onboard LED stays solid once NTP synchronization succeeds.

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
