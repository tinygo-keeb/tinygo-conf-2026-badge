package main

import (
	"image/color"
	"strconv"

	"tinygo.org/x/drivers/pixel"
	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/freesans"
)

const (
	menuWidth        = 240
	menuHeaderHeight = 28
	menuListY        = 34
	menuRowHeight    = 21
	menuFooterY      = 210
)

var (
	launcherBlack = color.RGBA{0, 0, 0, 255}
	launcherWhite = color.RGBA{255, 255, 255, 255}
	launcherGray  = color.RGBA{150, 150, 150, 255}
	launcherBlue  = color.RGBA{24, 64, 110, 255}
)

type menuScreen interface {
	FillScreen(color.RGBA)
	DrawBitmap(x, y int16, bitmap pixel.Image[pixel.RGB565BE]) error
}

// menuView renders complete rows in a reusable 13.4KB buffer. Selection
// changes replace only the old/new rows and counter, without clearing the LCD.
// The buffer becomes reclaimable when chooseProgram returns.
type menuView struct {
	screen          menuScreen
	buffer          []byte
	visible         bool
	selected, first int
}

func (m *menuView) tile(height int) menuCanvas {
	if m.buffer == nil {
		m.buffer = make([]byte, menuWidth*menuHeaderHeight*2)
	}
	c := menuCanvas{img: pixel.NewImageFromBytes[pixel.RGB565BE](menuWidth, height, m.buffer[:menuWidth*height*2])}
	c.img.FillSolidColor(pixel.NewColor[pixel.RGB565BE](0, 0, 0))
	return c
}

func (m *menuView) draw(names []string, selected, first int) error {
	if m.visible && m.selected == selected && m.first == first {
		return nil
	}
	if !m.visible {
		m.screen.FillScreen(launcherBlack)
		footer := m.tile(menuHeaderHeight)
		tinyfont.WriteLine(footer, &tinyfont.TomThumb, 8, 6, "UP/DOWN: select", launcherGray)
		tinyfont.WriteLine(footer, &tinyfont.TomThumb, 8, 22, "SW1 / JOY press: start    LEFT: TOP", launcherGray)
		if err := m.screen.DrawBitmap(0, menuFooterY, footer.img); err != nil {
			return err
		}
	}
	if !m.visible || m.first != first {
		for row := 0; row < menuRows; row++ {
			if err := m.drawRow(names, first, row, selected); err != nil {
				return err
			}
		}
	} else {
		for _, index := range [...]int{m.selected, selected} {
			if err := m.drawRow(names, first, index-first, selected); err != nil {
				return err
			}
		}
	}
	header := m.tile(menuHeaderHeight)
	tinyfont.WriteLine(header, &freesans.Regular9pt7b, 8, 23, "Examples", launcherWhite)
	tinyfont.WriteLine(header, &tinyfont.TomThumb, 190, 19,
		strconv.Itoa(selected+1)+"/"+strconv.Itoa(len(names)), launcherGray)
	if err := m.screen.DrawBitmap(0, 0, header.img); err != nil {
		return err
	}
	m.visible, m.selected, m.first = true, selected, first
	return nil
}

func (m *menuView) drawRow(names []string, first, row, selected int) error {
	c := m.tile(menuRowHeight)
	index := first + row
	if index < len(names) {
		if index == selected {
			blue := pixel.NewColor[pixel.RGB565BE](launcherBlue.R, launcherBlue.G, launcherBlue.B)
			for y := 0; y < menuRowHeight; y++ {
				for x := 4; x < menuWidth-4; x++ {
					c.img.Set(x, y, blue)
				}
			}
		}
		tinyfont.WriteLine(c, &freesans.Regular9pt7b, 12, 16, names[index], launcherWhite)
	}
	return m.screen.DrawBitmap(0, int16(menuListY+row*menuRowHeight), c.img)
}

type menuCanvas struct {
	img pixel.Image[pixel.RGB565BE]
}

func (c menuCanvas) Size() (int16, int16) {
	w, h := c.img.Size()
	return int16(w), int16(h)
}

func (c menuCanvas) SetPixel(x, y int16, col color.RGBA) {
	w, h := c.img.Size()
	if x >= 0 && y >= 0 && int(x) < w && int(y) < h {
		c.img.Set(int(x), int(y), pixel.NewColor[pixel.RGB565BE](col.R, col.G, col.B))
	}
}

func (c menuCanvas) Display() error { return nil }
