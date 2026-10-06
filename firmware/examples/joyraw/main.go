// ジョイスティックの可動範囲を計測するための例。
//
// 起動直後の 1 秒はスティックに触らない (センター補正)。その後スティックを
// 上下左右・円周に沿ってゆっくり倒すと、生値 (ADC 0..65535) と
// センター基準の差分、これまでの最小・最大を 100ms ごとに出力する。
// SW1 で最小・最大をリセット、SW2 でセンター補正をやり直す。
//
// 出力の見方:
//
//	raw=<x> <y>  d=<dx> <dy>  min=<minx> <miny>  max=<maxx> <maxy>  btn=<0|1>
//
// d はセンターからの差分、min/max は raw の範囲。
package main

import (
	"time"

	"github.com/sago35/tinygo-conf-2026-badge/firmware/badge"
)

func main() {
	sw1, sw2 := badge.NewButtons()
	joy := badge.NewJoystick()

	time.Sleep(500 * time.Millisecond)
	println("joyraw: calibrating center (do not touch the stick)")
	joy.Calibrate(1000 * time.Millisecond)
	println("joyraw: center =", joy.CenterX, joy.CenterY)

	var minX, minY uint16 = 65535, 65535
	var maxX, maxY uint16
	reset := func() {
		minX, minY = 65535, 65535
		maxX, maxY = 0, 0
	}
	prevSW1, prevSW2 := false, false
	for {
		s1, s2 := sw1.Pressed(), sw2.Pressed()
		if s1 && !prevSW1 {
			reset()
			println("joyraw: min/max reset")
		}
		if s2 && !prevSW2 {
			println("joyraw: recalibrating (do not touch the stick)")
			joy.Calibrate(1000 * time.Millisecond)
			println("joyraw: center =", joy.CenterX, joy.CenterY)
			reset()
		}
		prevSW1, prevSW2 = s1, s2

		rx, ry := joy.Raw()
		if rx < minX {
			minX = rx
		}
		if rx > maxX {
			maxX = rx
		}
		if ry < minY {
			minY = ry
		}
		if ry > maxY {
			maxY = ry
		}
		btn := 0
		if joy.Pressed() {
			btn = 1
		}
		println("raw=", rx, ry, " d=", int(rx)-int(joy.CenterX), int(ry)-int(joy.CenterY),
			" min=", minX, minY, " max=", maxX, maxY, " btn=", btn)
		time.Sleep(100 * time.Millisecond)
	}
}
