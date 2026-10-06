// Package ws2812s3 は ESP32-S3 (CPU 240MHz) 向けの WS2812B ビットバンギング
// ドライバ。tinygo.org/x/drivers/ws2812 は xtensa では 80/160MHz にしか
// 対応していないため、240MHz 用のタイミングで別実装している。
package ws2812s3

import (
	"errors"
	"image/color"
	"machine"
	"runtime/interrupt"
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
//
// フレーム全体を割り込み禁止で送る。バイトごとに割り込みを許可すると、
// BLE や Wi-Fi の割り込みで 50us 以上の隙間ができたときに WS2812B が
// フレームの終わりと判定してラッチしてしまい、後続の LED に色が届かない。
// 2 個なら約 60us、10 個でも約 300us の割り込み禁止で済む。
func (d Device) Write(buf []byte) (n int, err error) {
	if _, err := machine.GetCPUFrequency(); err != nil {
		return 0, err
	}
	mask := interrupt.Disable()
	d.warmup240()
	for _, c := range buf {
		if err := d.WriteByte(c); err != nil {
			interrupt.Restore(mask)
			return n, err
		}
		n++
	}
	interrupt.Restore(mask)
	return n, nil
}

// WriteColors は色の配列を LED に送信する。フレーム全体を割り込み禁止で送る
// (理由は Write のコメントを参照)。
// 呼び出し後は 50us 以上信号線を Low に保つ必要があるが、
// 通常の描画間隔なら自然に満たされる。
func (d Device) WriteColors(buf []color.RGBA) error {
	mask := interrupt.Disable()
	defer interrupt.Restore(mask)
	if _, err := machine.GetCPUFrequency(); err != nil {
		return err
	}
	// 送信開始時の命令キャッシュミスで先頭ビットが伸びないよう、先にコードを温める
	d.warmup240()
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
