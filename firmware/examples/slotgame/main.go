//go:build tinygo

// TinyGo Conference 2026 badge port of https://github.com/sat0ken/tinygo-slotgame.
package main

import (
	"machine"
	"time"

	"github.com/sago35/tinygo-conf-2026-badge/firmware/badge"
)

func main() {
	sw1 := badge.Button{Pin: badge.BUTTON1}
	sw1.Configure()
	// The upstream game has no sound. Keep the amplifier's inputs low so
	// they don't float while the game is running.
	for _, pin := range [...]machine.Pin{badge.I2S_BCLK, badge.I2S_LRC, badge.I2S_DIN} {
		pin.Configure(machine.PinConfig{Mode: machine.PinOutput})
		pin.Low()
	}

	g := newGame(badge.NewDisplay(), layoutSquare)
	g.oneButton = true
	// SW1 starts the game, then stops left, middle and right in order.
	prev := [3]bool{sw1.Pressed()}
	println("slotgame: SW1 = start / stop left, middle, right / restart")
	for {
		start := time.Now()
		down := [3]bool{sw1.Pressed()}
		g.step(edges(down, &prev), start)
		if d := frameTime - time.Since(start); d > 0 {
			time.Sleep(d)
		} else {
			// Keep the launcher's return-to-menu monitor responsive even
			// when drawing takes longer than the frame budget.
			time.Sleep(time.Millisecond)
		}
	}
}
