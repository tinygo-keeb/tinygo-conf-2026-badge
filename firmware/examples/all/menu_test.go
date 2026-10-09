package main

import (
	"bytes"
	"fmt"
	"image/color"
	"testing"

	"tinygo.org/x/drivers/pixel"
)

type menuTransfer struct{ x, y, width, height int }

type menuTestScreen struct {
	img       pixel.Image[pixel.RGB565BE]
	clears    int
	transfers []menuTransfer
}

func newMenuTestScreen() *menuTestScreen {
	return &menuTestScreen{img: pixel.NewImage[pixel.RGB565BE](menuWidth, 240)}
}

func (s *menuTestScreen) FillScreen(c color.RGBA) {
	s.clears++
	s.img.FillSolidColor(pixel.NewColor[pixel.RGB565BE](c.R, c.G, c.B))
}

func (s *menuTestScreen) DrawBitmap(x, y int16, img pixel.Image[pixel.RGB565BE]) error {
	w, h := img.Size()
	if x < 0 || y < 0 || int(x)+w > menuWidth || int(y)+h > 240 {
		return fmt.Errorf("bitmap outside LCD: %d,%d %dx%d", x, y, w, h)
	}
	s.transfers = append(s.transfers, menuTransfer{int(x), int(y), w, h})
	for row := 0; row < h; row++ {
		for column := 0; column < w; column++ {
			s.img.Set(int(x)+column, int(y)+row, img.Get(column, row))
		}
	}
	return nil
}

var menuTestNames = []string{
	"aht21b", "audio", "audiotest", "ble-scanner", "ble-sensor", "blink",
	"demo", "dht20", "display", "i2cscan", "input", "ir", "irlearn",
	"joyraw", "rhythm", "selftest", "slotgame", "wifi-httpget", "wifi-joystick", "wifi-server",
}

func TestMenuSelectionUpdatesOnlyChangedRowsAndCounter(t *testing.T) {
	screen := newMenuTestScreen()
	menu := menuView{screen: screen}
	if err := menu.draw(menuTestNames, 0, 0); err != nil {
		t.Fatal(err)
	}
	screen.transfers = nil
	if err := menu.draw(menuTestNames, 1, 0); err != nil {
		t.Fatal(err)
	}
	if screen.clears != 1 {
		t.Fatal("selection movement cleared the LCD")
	}
	if len(screen.transfers) != 3 {
		t.Fatalf("selection movement transferred %d areas, want 2 rows and counter", len(screen.transfers))
	}
	for _, area := range screen.transfers {
		if !((area.y == 0 && area.height == menuHeaderHeight) ||
			((area.y == menuListY || area.y == menuListY+menuRowHeight) && area.height == menuRowHeight)) {
			t.Fatalf("selection movement modified an unrelated area: %+v", area)
		}
	}
	screen.transfers = nil
	if err := menu.draw(menuTestNames, 1, 0); err != nil {
		t.Fatal(err)
	}
	if len(screen.transfers) != 0 {
		t.Fatal("unchanged selection redrew the menu")
	}
}

func TestMenuScrollAndWrapMatchCompleteRendering(t *testing.T) {
	screen := newMenuTestScreen()
	menu := menuView{screen: screen}
	for _, position := range [][2]int{{0, 0}, {1, 0}, {7, 0}, {8, 1}, {9, 2}, {19, 12}, {0, 0}, {19, 12}, {18, 12}} {
		selected, first := position[0], position[1]
		if err := menu.draw(menuTestNames, selected, first); err != nil {
			t.Fatal(err)
		}
		fresh := newMenuTestScreen()
		full := menuView{screen: fresh}
		if err := full.draw(menuTestNames, selected, first); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(screen.img.RawBuffer(), fresh.img.RawBuffer()) {
			t.Fatalf("incremental menu differs from complete rendering at %d, first %d", selected, first)
		}
		if screen.clears != 1 {
			t.Fatal("scrolling or wrapping cleared the LCD")
		}
	}
}
