// BLE ペリフェラルとして動作し、基板上のセンサーと入出力を公開する。
//
// アドバタイズ名は "TinyGo Conf 2026 #XXXXXX" (XXXXXX はチップ固有の ID)。
// 同じ ID を LCD にも表示するので、複数のバッジが同時に動いていても
// 自分のバッジを見分けられる。スマートフォンの nRF Connect などから
// 接続して確認できる。
//
//	Environmental Sensing (0x181A)
//	  Temperature (0x2A6E)  int16、0.01 度単位、Read/Notify (2 秒ごと)
//	  Humidity    (0x2A6F)  uint16、0.01% 単位、Read/Notify (2 秒ごと)
//	Badge (7a0d0001-2026-4b61-8467-65bad9e00000)
//	  LED    (7a0d0002-...)  Write (応答あり)。6 バイト (LED1 の R G B、LED2 の R G B) で
//	                         WS2812B 2 個の色を個別に、3 バイトなら両方を同じ色にする
//	  Button (7a0d0003-...)  Read/Notify、1 バイト (bit0 SW1、bit1 SW2、bit2 JOY)
//	  Mode   (7a0d0004-...)  Read/Write、1 バイト。0 = 手動 (LED に書いた色)、1 = 自動 (虹色)
//
// BLE が接続されていない間は LED を blink と同じ虹色で自動的に光らせる。
// 接続中は Mode で切り替える (接続直後は手動)。LED に色を書き込むと手動に戻る。
//
//	make flash-ble-sensor
//	make flash-ble-sensor BLE_DEBUG=1   (ATT/HCI の処理をシリアルに出すデバッグビルド)
//
// 同じディレクトリの webble.html を Chrome / Edge で開くと、Web Bluetooth で
// 接続して温湿度とボタンの表示、LED の色の変更ができる。
package main

import (
	"image/color"
	"time"

	"tinygo.org/x/bluetooth"
	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/freesans"

	"github.com/sago35/tinygo-conf-2026-badge/firmware/badge"
)

var adapter = bluetooth.DefaultAdapter

const namePrefix = "TinyGo Conf 2026 #"

var (
	serviceUUIDBadge = bluetooth.NewUUID([16]byte{0x7a, 0x0d, 0x00, 0x01, 0x20, 0x26, 0x4b, 0x61, 0x84, 0x67, 0x65, 0xba, 0xd9, 0xe0, 0x00, 0x00})
	charUUIDLED      = bluetooth.NewUUID([16]byte{0x7a, 0x0d, 0x00, 0x02, 0x20, 0x26, 0x4b, 0x61, 0x84, 0x67, 0x65, 0xba, 0xd9, 0xe0, 0x00, 0x00})
	charUUIDButton   = bluetooth.NewUUID([16]byte{0x7a, 0x0d, 0x00, 0x03, 0x20, 0x26, 0x4b, 0x61, 0x84, 0x67, 0x65, 0xba, 0xd9, 0xe0, 0x00, 0x00})
	charUUIDMode     = bluetooth.NewUUID([16]byte{0x7a, 0x0d, 0x00, 0x04, 0x20, 0x26, 0x4b, 0x61, 0x84, 0x67, 0x65, 0xba, 0xd9, 0xe0, 0x00, 0x00})
)

// LED のモード (Mode キャラクタリスティックの値)
const (
	modeManual = 0 // BLE で書き込んだ色
	modeAuto   = 1 // 虹色の自動点灯 (blink と同じ)
)

var (
	tempChar   bluetooth.Characteristic
	humChar    bluetooth.Characteristic
	ledChar    bluetooth.Characteristic
	buttonChar bluetooth.Characteristic
	modeChar   bluetooth.Characteristic
)

var (
	black = color.RGBA{0, 0, 0, 255}
	white = color.RGBA{255, 255, 255, 255}
	gray  = color.RGBA{160, 160, 160, 255}
	green = color.RGBA{0, 255, 0, 255}
	cyan  = color.RGBA{0, 200, 255, 255}
)

// LCD に出す状態。変化したときだけ描き直す。
type screenState struct {
	connected bool
	peer      string
	sensorOK  bool
	deciTemp  int32
	deciHum   int32
	ledColors [badge.WS2812_COUNT]color.RGBA
	ledAuto   bool // 自動点灯中か (未接続、または Mode が自動)
}

func main() {
	time.Sleep(2 * time.Second)

	serial := badge.SerialNumber()
	name := namePrefix + serial
	println("device name:", name)

	display := badge.NewDisplay()
	fb := badge.NewFramebuffer(display)
	leds := badge.NewLEDs()
	leds.SetBrightness(32)
	ledColors := make([]color.RGBA, badge.WS2812_COUNT)
	sw1, sw2 := badge.NewButtons()
	joy := badge.NewJoystick()
	sensor := badge.NewSensor()

	state := screenState{}
	mode := byte(modeManual) // 接続中の LED モード。未接続時は常に自動
	dirty := true
	drawScreen(fb, name, serial, &state)

	println("enabling BLE...")
	must("enable BLE stack", adapter.Enable())

	adapter.SetConnectHandler(func(dev bluetooth.Device, connected bool) {
		if connected {
			println("connected:", dev.Address.String())
			state.peer = dev.Address.String()
		} else {
			println("disconnected:", dev.Address.String())
			state.peer = ""
			mode = modeManual // 次の接続は手動から
			modeChar.Write([]byte{modeManual})
		}
		state.connected = connected
		dirty = true
	})

	must("add ESS service", adapter.AddService(&bluetooth.Service{
		UUID: bluetooth.ServiceUUIDEnvironmentalSensing,
		Characteristics: []bluetooth.CharacteristicConfig{
			{
				Handle: &tempChar,
				UUID:   bluetooth.CharacteristicUUIDTemperature,
				Value:  []byte{0, 0},
				Flags:  bluetooth.CharacteristicReadPermission | bluetooth.CharacteristicNotifyPermission,
			},
			{
				Handle: &humChar,
				UUID:   bluetooth.CharacteristicUUIDHumidity,
				Value:  []byte{0, 0},
				Flags:  bluetooth.CharacteristicReadPermission | bluetooth.CharacteristicNotifyPermission,
			},
		},
	}))
	must("add badge service", adapter.AddService(&bluetooth.Service{
		UUID: serviceUUIDBadge,
		Characteristics: []bluetooth.CharacteristicConfig{
			{
				Handle: &ledChar,
				UUID:   charUUIDLED,
				Value:  make([]byte, 3*badge.WS2812_COUNT),
				// Write Command (応答なし) は HCI バックエンドが処理しないので Write Request のみ
				Flags: bluetooth.CharacteristicReadPermission | bluetooth.CharacteristicWritePermission,
				WriteEvent: func(client bluetooth.Connection, offset int, value []byte) {
					if offset != 0 || len(value) < 3 {
						return
					}
					// 3 バイトごとに LED1, LED2, ... の色。足りない分は最後の色を使う
					for i := range ledColors {
						k := 3 * i
						if k+3 > len(value) {
							k = len(value) - len(value)%3 - 3
						}
						ledColors[i] = color.RGBA{R: value[k], G: value[k+1], B: value[k+2], A: 255}
					}
					for i, c := range ledColors {
						println("LED", i+1, ":", c.R, c.G, c.B)
					}
					leds.WriteColors(ledColors)
					copy(state.ledColors[:], ledColors)
					// 色を書き込まれたら手動に戻す
					if mode != modeManual {
						mode = modeManual
						modeChar.Write([]byte{modeManual})
					}
					dirty = true
				},
			},
			{
				Handle: &modeChar,
				UUID:   charUUIDMode,
				Value:  []byte{modeManual},
				Flags:  bluetooth.CharacteristicReadPermission | bluetooth.CharacteristicWritePermission,
				WriteEvent: func(client bluetooth.Connection, offset int, value []byte) {
					if offset != 0 || len(value) < 1 {
						return
					}
					if value[0] == modeAuto {
						mode = modeAuto
					} else {
						mode = modeManual
					}
					println("LED mode:", mode)
					dirty = true
				},
			},
			{
				Handle: &buttonChar,
				UUID:   charUUIDButton,
				Value:  []byte{0},
				Flags:  bluetooth.CharacteristicReadPermission | bluetooth.CharacteristicNotifyPermission,
			},
		},
	}))

	adv := adapter.DefaultAdvertisement()
	must("configure advertisement", adv.Configure(bluetooth.AdvertisementOptions{
		LocalName:    name,
		ServiceUUIDs: []bluetooth.UUID{bluetooth.ServiceUUIDEnvironmentalSensing},
	}))
	must("start advertisement", adv.Start())
	println("advertising as", name)

	lastSensor := time.Time{}
	prevButtons := byte(0xff)
	hue := 0
	for {
		// 未接続中、または接続中でも Mode が自動なら虹色 (blink と同じ)
		auto := !state.connected || mode == modeAuto
		if auto {
			hue++
			for i := range ledColors {
				ledColors[i] = hueColor(uint8(hue*4 + i*128))
			}
			leds.WriteColors(ledColors)
		}
		if auto != state.ledAuto {
			state.ledAuto = auto
			if !auto {
				// 手動に戻るときは BLE で指定された色に戻す
				copy(ledColors, state.ledColors[:])
				leds.WriteColors(ledColors)
			}
			dirty = true
		}

		// ボタン状態は変化したときだけ通知
		var b byte
		if sw1.Pressed() {
			b |= 1 << 0
		}
		if sw2.Pressed() {
			b |= 1 << 1
		}
		if joy.Pressed() {
			b |= 1 << 2
		}
		if b != prevButtons {
			prevButtons = b
			buttonChar.Write([]byte{b})
		}

		// 温湿度は 2 秒ごとに更新
		if time.Since(lastSensor) > 2*time.Second {
			lastSensor = time.Now()
			if err := sensor.Read(); err == nil {
				state.deciTemp = sensor.DeciCelsius()
				state.deciHum = sensor.DeciRelHumidity()
				state.sensorOK = true
				t := state.deciTemp * 10 // 0.01 度単位
				h := state.deciHum * 10  // 0.01% 単位
				tempChar.Write([]byte{byte(t), byte(t >> 8)})
				humChar.Write([]byte{byte(h), byte(h >> 8)})
			} else {
				println("sensor read error:", err.Error())
				state.sensorOK = false
			}
			dirty = true
		}

		if dirty {
			dirty = false
			drawScreen(fb, name, serial, &state)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// drawScreen は LCD に名前、ID、BLE の状態、温湿度を描く。
func drawScreen(fb *badge.Framebuffer, name, serial string, s *screenState) {
	w, h := fb.Size()
	fb.FillScreen(black)
	tinyfont.WriteLine(fb, &freesans.Bold12pt7b, 8, 28, "TinyGo Conf 2026", white)

	// ID は大きく表示
	tinyfont.WriteLine(fb, &freesans.Regular9pt7b, 8, 60, "ID", gray)
	tinyfont.WriteLine(fb, &freesans.Bold24pt7b, 8, 100, serial, cyan)

	// BLE の状態
	status := "advertising"
	statusColor := gray
	if s.connected {
		status = "connected"
		statusColor = green
	}
	tinyfont.WriteLine(fb, &freesans.Regular9pt7b, 8, 130, "BLE: "+status, statusColor)
	if s.connected && s.peer != "" {
		tinyfont.WriteLine(fb, &freesans.Regular9pt7b, 8, 150, s.peer, gray)
	}

	// 温湿度
	if s.sensorOK {
		tinyfont.WriteLine(fb, &freesans.Regular12pt7b, 8, 190, deci(s.deciTemp)+" C   "+deci(s.deciHum)+" %", white)
	} else {
		tinyfont.WriteLine(fb, &freesans.Regular9pt7b, 8, 190, "sensor: --", gray)
	}

	// LED の現在色を右下に表示 (左から LED1, LED2, ...)。自動点灯中は "auto"
	if s.ledAuto {
		tinyfont.WriteLine(fb, &freesans.Regular9pt7b, w-56, h-16, "auto", gray)
	} else {
		for i, c := range s.ledColors {
			x := w - 8 - int16(len(s.ledColors)-i)*36
			fb.FillRectangle(x, h-40, 32, 32, c)
		}
	}
	tinyfont.WriteLine(fb, &freesans.Regular9pt7b, 8, h-12, "LED", gray)
	fb.Display()
}

func must(action string, err error) {
	for err != nil {
		println("failed to " + action + ": " + err.Error())
		time.Sleep(time.Second)
	}
}

// hueColor は 0..255 の色相を RGB に変換する (examples/blink と同じ)。
func hueColor(h uint8) color.RGBA {
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
