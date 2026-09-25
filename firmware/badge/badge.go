// Package badge は TinyGo Conference 2026 バッジ (ESP32-S3-DevKit ベース) の
// ピン割り当てと、各ペリフェラルの初期化ヘルパーを提供する。
//
// ピン割り当ては tinygo-conf-2026.kicad_sch から抽出したもの。
// ビルドターゲットは esp32s3-box-3 を使う。
//
//	tinygo flash --target esp32s3-box-3 --size short ./examples/blink
package badge

import "machine"

// ST7789 1.3" 240x240 LCD (J3)。SPI0 (FSPI) に接続。
// BLK (バックライト) は 3V3 直結のため GPIO 制御不可。
const (
	LCD_SCK    = machine.GPIO12 // SCL
	LCD_MOSI   = machine.GPIO11 // SDA
	LCD_CS     = machine.GPIO10
	LCD_DC     = machine.GPIO5
	LCD_RST    = machine.GPIO4
	LCD_BL     = machine.NoPin
	LCD_WIDTH  = 240
	LCD_HEIGHT = 240
)

// MAX98357 I2S アンプ (U3) + スピーカー (LS1)。
// GAIN は GND 接続 (12dB)、SD_MODE は未接続 ((L+R)/2 出力)。
// TinyGo の machine パッケージは ESP32-S3 の I2S を未サポートのため、
// firmware/i2s パッケージでレジスタを直接操作して出力する (NewAudio 参照)。
const (
	I2S_BCLK = machine.GPIO45
	I2S_LRC  = machine.GPIO21
	I2S_DIN  = machine.GPIO47
)

// WS2812B (D1 -> D2 の直列接続)。
const (
	WS2812_PIN   = machine.GPIO16
	WS2812_COUNT = 2
)

// アナログジョイスティック ALPS RKJXV122400R (U2)。
// X/Y は 3V3-GND 間のポテンショメータで、中点が ADC 入力。
// ボタンは押下で GND (内部プルアップ必要)。
const (
	JOY_X   = machine.GPIO7 // ADC1_CH6
	JOY_Y   = machine.GPIO6 // ADC1_CH5
	JOY_BTN = machine.GPIO15
)

// プッシュスイッチ SW1 / SW2。押下で GND (内部プルアップ必要)。
const (
	BUTTON1 = machine.GPIO13
	BUTTON2 = machine.GPIO14
)

// 赤外線 LED (D5, R1 経由で GND、High で点灯) と赤外線受信モジュール (J2)。
const (
	IR_LED  = machine.GPIO17
	IR_DATA = machine.GPIO18
)

// Grove 互換 I2C コネクタ (J1): GND / 3V3 / SDA / SCL。I2C0 を使う。
const (
	I2C_SDA = machine.GPIO8
	I2C_SCL = machine.GPIO9
)

// AHT21B 温湿度センサー (J4)。I2C1 を使う。
// 同じ信号は拡張ヘッダ J5 の 11/12 ピンにも出ている。
const (
	AHT21B_SDA = machine.GPIO41
	AHT21B_SCL = machine.GPIO42
)

// 拡張ヘッダ J5 (2x8, 2.54mm)。
//
//	 1: 3V3     2: 3V3
//	 3: GPIO1   4: GPIO2
//	 5: GPIO3   6: GPIO38
//	 7: GPIO39  8: GND
//	 9: GND    10: GPIO40
//	11: GPIO41 12: GPIO42  (AHT21B の SDA/SCL と共用)
//	13: GPIO45 14: GPIO46  (GPIO45 は I2S_BCLK と共用)
//	15: GPIO48 16: VCC(5V)
const (
	EXT_GPIO1  = machine.GPIO1
	EXT_GPIO2  = machine.GPIO2
	EXT_GPIO3  = machine.GPIO3
	EXT_GPIO38 = machine.GPIO38
	EXT_GPIO39 = machine.GPIO39
	EXT_GPIO40 = machine.GPIO40
	EXT_GPIO41 = machine.GPIO41
	EXT_GPIO42 = machine.GPIO42
	EXT_GPIO45 = machine.GPIO45
	EXT_GPIO46 = machine.GPIO46
	EXT_GPIO48 = machine.GPIO48
)

// 各バスの割り当て。
var (
	LCDSPI    = machine.SPI0 // FSPI (SPI2)
	GroveI2C  = machine.I2C0
	SensorI2C = machine.I2C1
)
