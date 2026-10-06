package badge

import (
	"machine"

	"tinygo.org/x/drivers/st7789"
)

// LCDSPIFrequency は LCD 用 SPI のクロック周波数。
const LCDSPIFrequency = 40_000_000

// NewDisplay は SPI0 と ST7789 を初期化して返す。
// 画面は黒でクリアされた状態になる。
func NewDisplay() *st7789.Device {
	LCDSPI.Configure(machine.SPIConfig{
		Frequency: LCDSPIFrequency,
		SCK:       LCD_SCK,
		SDO:       LCD_MOSI,
		SDI:       machine.NoPin,
		Mode:      0,
	})
	display := st7789.New(LCDSPI, LCD_RST, LCD_DC, LCD_CS, LCD_BL)
	display.Configure(st7789.Config{
		Width:      LCD_WIDTH,
		Height:     LCD_HEIGHT,
		Rotation:   st7789.NO_ROTATION,
		FrameRate:  st7789.FRAMERATE_60,
		VSyncLines: st7789.MAX_VSYNC_SCANLINES,
	})
	return &display
}
