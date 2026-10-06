package badge

import (
	"github.com/sago35/tinygo-conf-2026-badge/firmware/ws2812s3"
)

// NewLEDs は WS2812B 用のドライバを返す。LED は D1, D2 の 2 個。
func NewLEDs() ws2812s3.Device {
	return ws2812s3.New(WS2812_PIN)
}
