package main

// A minimal Pico 2 W Wi-Fi + NTP example for TinyGo.
//
// Build with the cooperative scheduler; the CYW43439 network stack needs it:
//   tinygo flash -target=pico2-w -scheduler=tasks -stack-size=8kb -monitor .

import (
	"log/slog"
	"machine"
	"machine/usb/hid/keyboard"
	"runtime"
	"time"

	"github.com/soypat/cyw43439"
	"github.com/soypat/cyw43439/examples/cywnet"
)

const (
	hostname = "picopass"
	ntpHost  = "time.cloudflare.com"
	pollTime = 5 * time.Millisecond
	ledBlink = 300 * time.Millisecond
)

// Importing keyboard configures the Pico's USB interface as a composite CDC
// serial port and HID keyboard. USB serial logging remains available.
var hidKeyboard = keyboard.Port()

// Wire normally-open pushbuttons from these pins to GND. Internal pull-ups
// mean no external resistors are needed.
var (
	timeButton     = machine.GP14 // physical pin 19
	passwordButton = machine.GP15 // physical pin 20
)

func main() {
	logger := slog.New(slog.NewTextHandler(machine.Serial, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	timeButton.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	passwordButton.Configure(machine.PinConfig{Mode: machine.PinInputPullup})

	// Give `tinygo monitor` time to attach before emitting diagnostics.
	time.Sleep(2 * time.Second)
	logger.Info("starting Wi-Fi time sync")

	// This creates the CYW43439 Wi-Fi device and its small, embedded network
	// stack.
	deviceConfig := cyw43439.DefaultWifiConfig()
	deviceConfig.Logger = logger
	stack, err := cywnet.NewConfiguredPicoWithStack(
		wifiSSID,
		wifiPassword,
		deviceConfig,
		cywnet.StackConfig{Hostname: hostname},
	)
	if err != nil {
		panic("set up Wi-Fi stack: " + err.Error())
	}

	// The network device has to be pumped continually in the background. That
	// loop also blinks the Wi-Fi-chip LED until time synchronization completes.
	timeSynchronized := make(chan struct{}, 1)
	go pumpNetwork(stack, logger, timeSynchronized)

	lease, err := stack.SetupWithDHCP(cywnet.DHCPConfig{})
	if err != nil {
		panic("get DHCP lease: " + err.Error())
	}
	logger.Info("connected", slog.String("ip", lease.AssignedAddr.String()))

	// Resolve an NTP host using the DNS servers supplied by DHCP, then ask it
	// for the time. The retrying wrapper handles a few transient Wi-Fi hiccups.
	retryingStack := stack.LnetoStack().StackRetrying(cywnet.DefaultStackBackoff)
	servers, err := retryingStack.DoLookupIP(ntpHost, 5*time.Second, 3)
	if err != nil {
		panic("resolve NTP server: " + err.Error())
	}
	if len(servers) == 0 {
		panic("resolve NTP server: no addresses returned")
	}

	offset, err := retryingStack.DoNTP(servers[0], 5*time.Second, 3)
	if err != nil {
		panic("query NTP server: " + err.Error())
	}

	// TinyGo starts at the Unix epoch. Apply the NTP correction so future
	// calls to time.Now() report the current UTC time.
	runtime.AdjustTimeOffset(int64(offset))
	synchronizedTime := time.Now().UTC().Format(time.RFC3339)
	logger.Info("time synchronized", slog.String("utc", synchronizedTime))
	timeSynchronized <- struct{}{}

	// Wait for presses forever. This starts only after NTP succeeds, so every
	// timestamp typed by the button contains a valid UTC time.
	runTypeButtons(logger)
}

func pumpNetwork(stack *cywnet.Stack, logger *slog.Logger, timeSynchronized <-chan struct{}) {
	ledOn := false
	synchronizing := true
	lastBlink := time.Now()
	setOnboardLED(stack, false, logger)

	for {
		select {
		case <-timeSynchronized:
			synchronizing = false
		default:
		}

		if synchronizing {
			now := time.Now()
			if now.Sub(lastBlink) >= ledBlink {
				ledOn = !ledOn
				setOnboardLED(stack, ledOn, logger)
				lastBlink = now
			}
		} else if !ledOn {
			// NTP succeeded: leave the onboard LED solidly on.
			ledOn = true
			setOnboardLED(stack, true, logger)
		}

		sent, received, _ := stack.RecvAndSend()
		if sent == 0 && received == 0 {
			time.Sleep(pollTime)
		}
	}
}

func setOnboardLED(stack *cywnet.Stack, on bool, logger *slog.Logger) {
	// The Pico 2 W LED is wireless-chip GPIO 0, not RP2350 GPIO 25.
	if err := stack.Device().GPIOSet(0, on); err != nil {
		logger.Error("set onboard LED", slog.String("error", err.Error()))
	}
}

// typeText sends ordinary text as HID keyboard keypresses. A short delay after
// every character prevents long messages from overflowing the USB HID queue.
// Keyboard shortcuts can be sent separately with hidKeyboard.Down and Up.
func typeText(text string) error {
	for i := 0; i < len(text); i++ {
		if _, err := hidKeyboard.Write([]byte{text[i]}); err != nil {
			return err
		}
		time.Sleep(8 * time.Millisecond)
	}
	return nil
}

func buttonPressed(button machine.Pin) bool {
	if button.Get() {
		return false
	}

	time.Sleep(20 * time.Millisecond) // debounce the press
	return !button.Get()
}

func waitForButtonRelease(button machine.Pin) {
	// Holding a button produces only one entry.
	for !button.Get() {
		time.Sleep(5 * time.Millisecond)
	}
	time.Sleep(20 * time.Millisecond) // debounce the release
}

func runTypeButtons(logger *slog.Logger) {
	lastReport := time.Now()
	for {
		if buttonPressed(timeButton) { // active-low: button connects GP14 to GND
			now := time.Now().UTC().Format(time.RFC3339)
			logger.Info("typing current UTC time", slog.String("utc", now))
			if err := typeText(now + "\n"); err != nil {
				logger.Error("type HID time", slog.String("error", err.Error()))
			}
			waitForButtonRelease(timeButton)
		} else if buttonPressed(passwordButton) { // active-low: button connects GP15 to GND
			// Do not log the password, and do not send Enter: this just fills the
			// currently focused password field.
			logger.Info("typing system password")
			if err := typeText(systemPassword); err != nil {
				logger.Error("type HID system password", slog.String("error", err.Error()))
			}
			waitForButtonRelease(passwordButton)
		}

		now := time.Now()
		if now.Sub(lastReport) >= time.Minute {
			logger.Info("current UTC time", slog.String("utc", now.UTC().Format(time.RFC3339)))
			lastReport = now
		}
		time.Sleep(5 * time.Millisecond)
	}
}
