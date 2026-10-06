// ST7789 にカラーバーと文字を表示する。
package main

import (
	"image/color"
	"time"

	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/freesans"

	"github.com/sago35/tinygo-conf-2026-badge/firmware/badge"
)

func main() {
	display := badge.NewDisplay()
	w, h := display.Size()
	println("display", w, h)

	black := color.RGBA{0, 0, 0, 255}
	white := color.RGBA{255, 255, 255, 255}
	bars := []color.RGBA{
		{255, 0, 0, 255}, {0, 255, 0, 255}, {0, 0, 255, 255},
		{255, 255, 0, 255}, {0, 255, 255, 255}, {255, 0, 255, 255},
	}

	display.FillScreen(black)
	bw := w / int16(len(bars))
	for i, c := range bars {
		display.FillRectangle(int16(i)*bw, 0, bw, h/3, c)
	}
	tinyfont.WriteLine(display, &freesans.Bold12pt7b, 8, h/2, "TinyGo Conf 2026", white)

	for i := 0; ; i++ {
		display.FillRectangle(0, h-40, w, 40, black)
		tinyfont.WriteLine(display, &freesans.Regular9pt7b, 8, h-12, "uptime: "+itoa(i)+"s", white)
		time.Sleep(time.Second)
	}
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	neg := v < 0
	if neg {
		v = -v
	}
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
