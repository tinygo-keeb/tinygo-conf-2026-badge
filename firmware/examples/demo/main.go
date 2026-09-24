// バッジの全機能デモ。
//   - LCD にジョイスティック位置、ボタン状態、温湿度を表示
//   - ジョイスティックの位置に応じて WS2812B の色が変わる
//   - SW1 / SW2 で LED の明るさを変える
package main

import (
	"image/color"
	"time"

	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/freesans"

	"github.com/sago35/tinygo-conf-2026-badge/firmware/badge"
)

var (
	black = color.RGBA{0, 0, 0, 255}
	white = color.RGBA{255, 255, 255, 255}
	gray  = color.RGBA{64, 64, 64, 255}
	green = color.RGBA{0, 255, 0, 255}
	red   = color.RGBA{255, 0, 0, 255}
)

func main() {
	display := badge.NewDisplay()
	leds := badge.NewLEDs()
	sw1, sw2 := badge.NewButtons()
	joy := badge.NewJoystick()
	time.Sleep(100 * time.Millisecond)
	sensor := badge.NewSensor()

	w, h := display.Size()
	display.FillScreen(black)
	tinyfont.WriteLine(display, &freesans.Bold12pt7b, 8, 24, "TinyGo Conf 2026", white)

	// ジョイスティック表示エリア (中央の四角)
	const padSize = 100
	padX := (w - padSize) / 2
	padY := int16(40)
	display.FillRectangle(padX, padY, padSize, padSize, gray)

	brightness := uint8(32)
	colors := make([]color.RGBA, badge.WS2812_COUNT)
	var dotX, dotY int16 = -1, -1
	lastSensor := time.Time{}
	prevSW1, prevSW2 := false, false

	for {
		x, y, btn := joy.Read()
		p1, p2 := sw1.Pressed(), sw2.Pressed()

		// ボタンで LED の明るさを調整
		if p1 && !prevSW1 && brightness < 224 {
			brightness += 32
		}
		if p2 && !prevSW2 && brightness > 32 {
			brightness -= 32
		}
		prevSW1, prevSW2 = p1, p2
		leds.SetBrightness(brightness)

		// ジョイスティックの位置を LED の色に反映
		c := color.RGBA{R: uint8((x + 1000) / 8), G: uint8((y + 1000) / 8), B: 0, A: 255}
		if btn {
			c.B = 255
		}
		for i := range colors {
			colors[i] = c
		}
		leds.WriteColors(colors)

		// ジョイスティックの位置をドットで表示
		nx := padX + int16((x+1000)*(padSize-8)/2000)
		ny := padY + int16((1000-y)*(padSize-8)/2000)
		if nx != dotX || ny != dotY {
			if dotX >= 0 {
				display.FillRectangle(dotX, dotY, 8, 8, gray)
			}
			dotColor := white
			if btn {
				dotColor = red
			}
			display.FillRectangle(nx, ny, 8, 8, dotColor)
			dotX, dotY = nx, ny
		}

		// ボタン状態
		display.FillRectangle(8, padY+padSize+10, w-16, 20, black)
		tinyfont.WriteLine(display, &freesans.Regular9pt7b, 8, padY+padSize+26, "SW1:"+onoff(p1)+" SW2:"+onoff(p2)+" JOY:"+onoff(btn), green)

		// 温湿度は 1 秒ごとに更新
		if time.Since(lastSensor) > time.Second {
			lastSensor = time.Now()
			display.FillRectangle(8, h-30, w-16, 24, black)
			if err := sensor.Read(); err == nil {
				t := sensor.DeciCelsius()
				hu := sensor.DeciRelHumidity()
				tinyfont.WriteLine(display, &freesans.Regular9pt7b, 8, h-12,
					deci(t)+"C  "+deci(hu)+"%", white)
			} else {
				tinyfont.WriteLine(display, &freesans.Regular9pt7b, 8, h-12, "sensor error", red)
			}
		}

		time.Sleep(30 * time.Millisecond)
	}
}

func onoff(b bool) string {
	if b {
		return "ON "
	}
	return "off"
}

// deci は 1/10 単位の整数を "12.3" 形式の文字列にする。
func deci(v int32) string {
	neg := v < 0
	if neg {
		v = -v
	}
	s := itoa(int(v/10)) + "." + itoa(int(v%10))
	if neg {
		return "-" + s
	}
	return s
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}
