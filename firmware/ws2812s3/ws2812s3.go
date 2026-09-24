// Package ws2812s3 は ESP32-S3 (CPU 240MHz) 向けの WS2812B ビットバンギング
// ドライバ。tinygo.org/x/drivers/ws2812 は xtensa では 80/160MHz にしか
// 対応していないため、240MHz 用のタイミングで別実装している。
package ws2812s3

import (
	"errors"
	"image/color"
	"machine"
)

var errUnknownClockSpeed = errors.New("ws2812s3: unsupported CPU clock (240MHz only)")

// Device は 1 本の信号線に接続された WS2812B 列を表す。
type Device struct {
	Pin        machine.Pin
	brightness uint8
}

// New はピンを出力に設定し、ドライバを返す。
func New(pin machine.Pin) Device {
	pin.Configure(machine.PinConfig{Mode: machine.PinOutput})
	pin.Low()
	return Device{Pin: pin, brightness: 255}
}

// SetBrightness は WriteColors 時に掛ける明るさ (0..255) を設定する。
func (d *Device) SetBrightness(b uint8) {
	d.brightness = b
}

// WriteByte は 1 バイトを WS2812 プロトコルで送信する。
func (d Device) WriteByte(c byte) error {
	freq, err := machine.GetCPUFrequency()
	if err != nil {
		return err
	}
	switch freq {
	case 240_000_000:
		d.writeByte240(c)
		return nil
	default:
		return errUnknownClockSpeed
	}
}

// Write は GRB 順に並んだバイト列をそのまま送信する。
func (d Device) Write(buf []byte) (n int, err error) {
	for _, c := range buf {
		if err := d.WriteByte(c); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

// WriteColors は色の配列を LED に送信する。
// 呼び出し後は 50us 以上信号線を Low に保つ必要があるが、
// 通常の描画間隔なら自然に満たされる。
func (d Device) WriteColors(buf []color.RGBA) error {
	for _, c := range buf {
		r, g, b := applyBrightness(c, d.brightness)
		if err := d.WriteByte(g); err != nil {
			return err
		}
		if err := d.WriteByte(r); err != nil {
			return err
		}
		if err := d.WriteByte(b); err != nil {
			return err
		}
	}
	return nil
}

func applyBrightness(c color.RGBA, brightness uint8) (r, g, b uint8) {
	if brightness == 255 {
		return c.R, c.G, c.B
	}
	scale := uint16(brightness) + 1
	return uint8(uint16(c.R) * scale >> 8), uint8(uint16(c.G) * scale >> 8), uint8(uint16(c.B) * scale >> 8)
}
