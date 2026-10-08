// All standalone badge examples in one firmware, with a logo and launcher.
package main

//go:generate env GO111MODULE=off go run generate.go

import (
	_ "embed"
	"image/color"
	"time"

	"github.com/sago35/tinygo-conf-2026-badge/firmware/badge"
	"tinygo.org/x/drivers/st7789"
	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/freesans"
)

// The source image is converted on the host, so neither its large decoded
// bitmap nor an image decoder occupies the badge's limited RAM.
//
//go:embed img/logo.rgb565
var logoPixels string

// Shared by all three Wi-Fi examples; set with -X main.ssid/main.password.
var ssid, password string

type program struct {
	name string
	run  func()
	wifi bool
	hint string
}

func main() {
	nav := navigationFromMenuReturn(takeMenuReturn(), len(programs))
	joy := badge.NewJoystick()
	sw1, sw2 := badge.NewButtons()
	display := badge.NewDisplay()
	index := chooseProgram(display, joy, sw1, sw2, nav)
	selected := programs[index]
	showProgram(display, selected)
	println("all: starting", selected.name, "(hold SW1+SW2 for 1s: menu)")

	// Reset on return so no old goroutine, DMA buffer, IR interrupt or radio
	// stack survives into the next example. This also works during long sleeps
	// and blocking HTTP/BLE operations in the original examples.
	go watchMenu(sw1, sw2, index)
	if selected.wifi && ssid == "" {
		display.FillScreen(launcherBlack)
		launcherText(display, 50, "Wi-Fi SSID is not set", launcherWhite)
		launcherText(display, 85, "Build with SSID/PASS", launcherWhite)
		launcherText(display, 130, "See examples/all/README.md", launcherGray)
		launcherText(display, 210, "SW1+SW2 1s: MENU", launcherWhite)
		select {}
	}
	selected.run()
	// Some examples may finish (for example after a server error). Keep the
	// return-to-menu monitor alive even when their entry function returns.
	select {}
}

func showLogo(display *st7789.Device) {
	const rowBytes = badge.LCD_WIDTH * 2
	const rows = badge.DisplayChunkRows
	if len(logoPixels) != badge.LCD_HEIGHT*rowBytes {
		println("all: invalid generated logo; run go generate ./examples/all")
		return
	}
	// Only this 7.5KB transfer buffer lives in RAM; logoPixels stays in flash.
	var chunk [rows * rowBytes]byte
	for y := 0; y < badge.LCD_HEIGHT; y += rows {
		n := min(rows, badge.LCD_HEIGHT-y)
		copy(chunk[:n*rowBytes], logoPixels[y*rowBytes:(y+n)*rowBytes])
		if err := display.DrawRGBBitmap8(0, int16(y), chunk[:n*rowBytes], badge.LCD_WIDTH, int16(n)); err != nil {
			println("all: logo display:", err.Error())
			return
		}
	}
}

func chooseProgram(display *st7789.Device, joy *badge.Joystick, sw1, sw2 badge.Button, nav navigation) int {
	menu := menuView{screen: display}
	names := make([]string, len(programs))
	for i := range programs {
		names[i] = programs[i].name
	}
	drawMenu := func() {
		if err := menu.draw(names, nav.selected, nav.first); err != nil {
			println("all: menu display:", err.Error())
		}
	}
	if nav.menu {
		drawMenu()
	} else {
		showLogo(display)
	}
	calibrateJoystick(joy)
	for {
		x, y, pressed := joy.Read()
		changed, launch := nav.update(time.Now(), x, y, sw1.Pressed(), sw2.Pressed(), pressed, len(programs))
		if changed {
			if nav.menu {
				drawMenu()
			} else {
				menu.visible = false
				showLogo(display)
			}
		}
		if launch {
			// Don't pass the launch press or a tilted stick to an example's
			// controls or startup calibration. Left can cancel this wait.
			launcherText(display, 215, "Release buttons and stick", launcherWhite)
			if waitForRelease(joy, sw1, sw2) {
				return nav.selected
			}
			nav.menu = false
			menu.visible = false
			showLogo(display)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func calibrateJoystick(joy *badge.Joystick) {
	// A return-to-menu reset can finish while the stick is tilted. Wait for
	// neutral before calibration so a held direction never becomes the center.
	var centeredAt time.Time
	for {
		x, y, _ := joy.Read()
		if x > -directionRelease && x < directionRelease && y > -directionRelease && y < directionRelease {
			if centeredAt.IsZero() {
				centeredAt = time.Now()
			}
			if time.Since(centeredAt) >= 100*time.Millisecond {
				break
			}
		} else {
			centeredAt = time.Time{}
		}
		time.Sleep(10 * time.Millisecond)
	}
	joy.Calibrate(300 * time.Millisecond)
}

func waitForRelease(joy *badge.Joystick, sw1, sw2 badge.Button) bool {
	var centeredAt time.Time
	for {
		x, y, pressed := joy.Read()
		if x < -directionPress {
			return false
		}
		if x > -directionRelease && x < directionRelease && y > -directionRelease && y < directionRelease && !pressed && !sw1.Pressed() && !sw2.Pressed() {
			if centeredAt.IsZero() {
				centeredAt = time.Now()
			}
			if time.Since(centeredAt) >= 100*time.Millisecond {
				return true
			}
		} else {
			centeredAt = time.Time{}
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func showProgram(display *st7789.Device, selected program) {
	display.FillScreen(launcherBlack)
	launcherText(display, 40, selected.name, launcherWhite)
	launcherText(display, 85, "Running", launcherWhite)
	launcherText(display, 125, "Output: USB serial", launcherGray)
	if selected.hint != "" {
		launcherText(display, 165, selected.hint, launcherGray)
	}
	launcherText(display, 225, "SW1+SW2 1s: MENU", launcherWhite)
}

func launcherText(display *st7789.Device, y int16, text string, c color.RGBA) {
	tinyfont.WriteLine(display, &freesans.Regular9pt7b, 8, y, text, c)
}

func watchMenu(sw1, sw2 badge.Button, selected int) {
	var gesture menuReturnGesture
	for {
		if gesture.update(time.Now(), sw1.Pressed(), sw2.Pressed()) {
			println("all: returning to program menu")
			resetToMenu(selected)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
