// WS2812B を虹色に光らせながら、シリアルにカウンタを出力する。
package main

import (
	"image/color"
	"time"

	"github.com/sago35/tinygo-conf-2026-badge/firmware/badge"
)

func main() {
	leds := badge.NewLEDs()
	leds.SetBrightness(32)
	colors := make([]color.RGBA, badge.WS2812_COUNT)

	for i := 0; ; i++ {
		for n := range colors {
			colors[n] = hue(uint8(i*4 + n*128))
		}
		if err := leds.WriteColors(colors); err != nil {
			println("ws2812:", err.Error())
		}
		if i%50 == 0 {
			println("blink", i)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// hue は 0..255 の色相を RGB に変換する。
func hue(h uint8) color.RGBA {
	h3 := uint16(h) * 3
	switch {
	case h3 < 256:
		return color.RGBA{R: 255 - uint8(h3), G: uint8(h3), B: 0, A: 255}
	case h3 < 512:
		h3 -= 256
		return color.RGBA{R: 0, G: 255 - uint8(h3), B: uint8(h3), A: 255}
	default:
		h3 -= 512
		return color.RGBA{R: uint8(h3), G: 0, B: 255 - uint8(h3), A: 255}
	}
}
