// 基板上の全デバイスを一度に動作確認するセルフテスト。
// 組み立てた基板の検査や、ワークショップ前の動作確認に使う。
//
// 起動直後にジョイスティックのセンター位置を測るので、電源投入・リセット直後は
// スティックに触らないこと。そのあと LCD を赤・緑・青の順に全面表示し
// (ドット欠けや色の確認)、起動音を鳴らしてテスト画面になる。
// テスト画面では各項目を次のように確認する。
//
//	SW1 / SW2   押すと緑になる。押すたびにビープ音が鳴る (スピーカーの確認)
//	JOY         ボタンを押す、スティックを上下左右いっぱいに倒すと U D L R が緑になる。
//	            右上の枠にスティックの位置を表示する
//	LED         WS2812B 2 個が赤、緑、青、白の順に変わる。画面の色見本と同じか目で見る
//	AHT21B      温湿度を 1 秒ごとに読む。読めて値が妥当なら緑
//	GROVE       Grove コネクタの I2C バスを 2 秒ごとにスキャンし、応答したアドレスを表示
//	            (何もつないでいなければ none。合否には含めない)
//	IR          1 秒ごとに NEC フォーマットで送信し、反射光を自分で受信できたら緑。
//	            表示は「自己受信できた回数/送信回数」。返ってこないときは白い紙などを
//	            赤外線 LED と受信モジュールの前にかざす。ほかのリモコンの信号を受けたら
//	            ext にアドレス:コマンド を表示する
//	AUDIO       I2S の初期化に成功したら緑。音が出ているかは耳で確認する
//
// 自動で判定できる項目 (SW1, SW2, JOY, AHT21B, IR, AUDIO) がすべて緑になると
// 最下行が ALL OK になる。LCD, LED, スピーカー, Grove は目と耳で確認する。
// 判定結果や受信データはシリアルにも出力する。
package main

import (
	"image/color"
	"runtime/volatile"
	"time"

	"tinygo.org/x/drivers/irremote"
	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/freesans"

	"github.com/sago35/tinygo-conf-2026-badge/firmware/badge"
)

var (
	black  = color.RGBA{0, 0, 0, 255}
	white  = color.RGBA{255, 255, 255, 255}
	gray   = color.RGBA{64, 64, 64, 255}
	green  = color.RGBA{0, 255, 0, 255}
	red    = color.RGBA{255, 0, 0, 255}
	blue   = color.RGBA{0, 0, 255, 255}
	yellow = color.RGBA{255, 220, 0, 255}
	cyan   = color.RGBA{0, 255, 255, 255}
)

var font = &freesans.Regular9pt7b

// 画面のレイアウト。1 行 22 ドット (freesans 9pt の行送り)。
const (
	labelX  = 4
	valueX  = 70
	rowH    = 22
	firstY  = 42 // 最初の項目行のベースライン
	padSize = 64 // ジョイスティック表示の枠
	padX    = badge.LCD_WIDTH - padSize - 6
	padY    = 26
)

// 音量 (0..255)。examples/audio と同じ。
const volume = 32

// 赤外線の送信アドレス。コマンドは送るたびに 1 ずつ増やす。
const irAddress = 0x10

// LED の点灯パターン。ledInterval ごとに次の色に進む。
var ledColors = []struct {
	name string
	c    color.RGBA
}{
	{"red", red},
	{"green", green},
	{"blue", blue},
	{"white", white},
}

const ledInterval = 500 * time.Millisecond

// 赤外線受信の割り込みからメインループへの受け渡し用 (examples/ir と同じ)。
// 割り込み内で println すると USB シリアルが止まることがあるので、保存だけする。
var (
	rxSeq     volatile.Register32
	rxCode    volatile.Register32
	rxAddress volatile.Register16
	rxCommand volatile.Register16
	rxFlags   volatile.Register16
)

func onReceive(d irremote.Data) {
	rxCode.Set(d.Code)
	rxAddress.Set(d.Address)
	rxCommand.Set(d.Command)
	rxFlags.Set(uint16(d.Flags))
	rxSeq.Set(rxSeq.Get() + 1)
}

// check は自動判定する項目。一度 OK になったら OK のまま。
type check struct {
	name string
	ok   bool
}

func (c *check) pass() {
	if !c.ok {
		c.ok = true
		println("PASS", c.name)
	}
}

func main() {
	// ジョイスティックのセンター位置は起動直後に測る (触らないこと)
	joy := badge.NewJoystick()
	joy.Calibrate(300 * time.Millisecond)

	time.Sleep(200 * time.Millisecond)
	println("selftest start, id", badge.SerialNumber())

	display := badge.NewDisplay()
	fb := badge.NewFramebuffer(display)
	w, h := fb.Size()

	// LCD の全面表示。この間にほかのデバイスを初期化する
	fb.FillScreen(red)
	fb.Display()
	leds := badge.NewLEDs()
	leds.SetBrightness(32)
	sw1, sw2 := badge.NewButtons()
	time.Sleep(300 * time.Millisecond)

	fb.FillScreen(green)
	fb.Display()
	sensor := badge.NewSensor()
	badge.ConfigureGroveI2C()
	rx := badge.NewIRReceiver()
	rx.SetCommandHandler(onReceive)
	tx, txErr := badge.NewIRTransmitter()
	if txErr != nil {
		println("ir tx init:", txErr.Error())
	}
	time.Sleep(300 * time.Millisecond)

	fb.FillScreen(blue)
	fb.Display()
	time.Sleep(300 * time.Millisecond)

	var (
		chkSW1   = check{name: "SW1"}
		chkSW2   = check{name: "SW2"}
		chkBtn   = check{name: "JOY button"}
		chkUp    = check{name: "JOY up"}
		chkDown  = check{name: "JOY down"}
		chkLeft  = check{name: "JOY left"}
		chkRight = check{name: "JOY right"}
		chkAHT   = check{name: "AHT21B"}
		chkIR    = check{name: "IR loopback"}
		chkAudio = check{name: "AUDIO"}
	)
	checks := []*check{&chkSW1, &chkSW2, &chkBtn, &chkUp, &chkDown, &chkLeft, &chkRight, &chkAHT, &chkIR, &chkAudio}

	audio, audioErr := badge.NewAudio()
	var tone *badge.ToneGenerator
	if audioErr != nil {
		println("audio init:", audioErr.Error())
	} else {
		chkAudio.pass()
		tone = badge.NewToneGenerator(audio)
		for _, n := range []uint32{523, 659, 784} {
			tone.Play(n, 80, volume)
		}
		tone.Stop()
	}
	beep := func(freq uint32) {
		if tone != nil {
			tone.Play(freq, 60, volume)
			tone.Stop()
		}
	}

	ledIdx := 0
	lastLED := time.Time{}

	lastSensor := time.Time{}
	sensorText := "reading..."

	lastScan := time.Time{}
	groveText := "scanning..."
	groveColor := yellow

	lastTX := time.Time{}
	irCmd := uint8(0)
	irSent := 0
	irBack := 0
	irOther := "" // 自分以外から受信したコード
	lastSeq := rxSeq.Get()

	prev1, prev2, prevBtn := false, false, false
	allOK := false

	for {
		frameStart := time.Now()
		now := frameStart

		// ---- 入力 ----
		x, y, btn := joy.Read()
		p1, p2 := sw1.Pressed(), sw2.Pressed()
		if p1 {
			chkSW1.pass()
		}
		if p2 {
			chkSW2.pass()
		}
		if btn {
			chkBtn.pass()
		}
		const edge = 900
		if y > edge {
			chkUp.pass()
		}
		if y < -edge {
			chkDown.pass()
		}
		if x < -edge {
			chkLeft.pass()
		}
		if x > edge {
			chkRight.pass()
		}

		// 押した瞬間にビープを鳴らす (ボタンごとに音程を変える)
		if p1 && !prev1 {
			beep(880)
		}
		if p2 && !prev2 {
			beep(660)
		}
		if btn && !prevBtn {
			beep(440)
		}
		prev1, prev2, prevBtn = p1, p2, btn

		// ---- LED ----
		if now.Sub(lastLED) >= ledInterval {
			lastLED = now
			ledIdx = (ledIdx + 1) % len(ledColors)
			c := ledColors[ledIdx].c
			leds.WriteColors([]color.RGBA{c, c})
		}

		// ---- 温湿度センサー ----
		if now.Sub(lastSensor) >= time.Second {
			lastSensor = now
			if err := sensor.Read(); err != nil {
				sensorText = "read error"
			} else {
				t, rh := sensor.DeciCelsius(), sensor.DeciRelHumidity()
				sensorText = deci(t) + "C " + deci(rh) + "%"
				if t > -200 && t < 800 && rh > 0 && rh <= 1000 {
					chkAHT.pass()
				} else {
					sensorText += " ?"
				}
			}
		}

		// ---- Grove I2C ----
		if now.Sub(lastScan) >= 2*time.Second {
			lastScan = now
			found := badge.ScanI2C(badge.GroveI2C)
			s := ""
			for _, addr := range found {
				s += "0x" + hex8(uint8(addr)) + " "
			}
			if s == "" {
				s = "none"
				groveColor = white
			} else {
				groveColor = cyan
			}
			if s != groveText {
				println("grove i2c:", s)
			}
			groveText = s
		}

		// ---- 赤外線 ----
		if seq := rxSeq.Get(); seq != lastSeq {
			lastSeq = seq
			repeat := rxFlags.Get()&uint16(irremote.DataFlagIsRepeat) != 0
			addr, cmd := rxAddress.Get(), rxCommand.Get()
			if !repeat && addr == irAddress && cmd == uint16(irCmd) {
				irBack++
				chkIR.pass()
			} else if !repeat {
				irOther = hex8(uint8(addr)) + ":" + hex8(uint8(cmd))
				println("ir rx: addr", addr, "cmd", cmd, "code 0x"+hex32(rxCode.Get()))
			}
		}
		if tx != nil && now.Sub(lastTX) >= time.Second {
			lastTX = now
			irCmd++
			irSent++
			tx.SendNEC(irAddress, irCmd)
		}

		// ---- 判定 ----
		if !allOK {
			allOK = true
			for _, c := range checks {
				allOK = allOK && c.ok
			}
			if allOK {
				println("ALL OK")
				if tone != nil {
					for _, n := range []uint32{784, 1047} {
						tone.Play(n, 100, volume)
					}
					tone.Stop()
				}
			}
		}

		// ---- 描画 ----
		fb.FillScreen(black)
		tinyfont.WriteLine(fb, font, labelX, 18, "SELF TEST #"+badge.SerialNumber(), white)

		row := int16(firstY)
		label := func(s string) {
			tinyfont.WriteLine(fb, font, labelX, row, s, white)
		}

		label("SW1")
		text(fb, valueX, row, pressedText(p1), state(chkSW1.ok))
		row += rowH

		label("SW2")
		text(fb, valueX, row, pressedText(p2), state(chkSW2.ok))
		row += rowH

		label("JOY")
		cx := text(fb, valueX, row, "B", state(chkBtn.ok))
		cx = text(fb, cx+6, row, "U", state(chkUp.ok))
		cx = text(fb, cx+6, row, "D", state(chkDown.ok))
		cx = text(fb, cx+6, row, "L", state(chkLeft.ok))
		text(fb, cx+6, row, "R", state(chkRight.ok))
		row += rowH

		label("LED")
		lc := ledColors[ledIdx]
		fb.FillRectangle(valueX, row-13, 24, 14, lc.c)
		text(fb, valueX+30, row, lc.name, white)
		row += rowH

		label("AHT")
		text(fb, valueX, row, sensorText, state(chkAHT.ok))
		row += rowH

		label("GROVE")
		text(fb, valueX, row, groveText, groveColor)
		row += rowH

		label("IR")
		irText := itoa(irBack) + "/" + itoa(irSent)
		if tx == nil {
			irText = "tx init error"
		}
		cx = text(fb, valueX, row, irText, state(chkIR.ok))
		if irOther != "" {
			text(fb, cx+8, row, "ext "+irOther, cyan)
		}
		row += rowH

		label("AUDIO")
		if audioErr != nil {
			text(fb, valueX, row, "init error", red)
		} else {
			text(fb, valueX, row, "ok (listen)", green)
		}

		// ジョイスティックの位置
		fb.FillRectangle(padX, padY, padSize, padSize, gray)
		fb.FillRectangle(padX+padSize/2, padY, 1, padSize, black)
		fb.FillRectangle(padX, padY+padSize/2, padSize, 1, black)
		nx := padX + int16((x+1000)*(padSize-6)/2000)
		ny := padY + int16((1000-y)*(padSize-6)/2000)
		dot := white
		if btn {
			dot = red
		}
		fb.FillRectangle(nx, ny, 6, 6, dot)

		// 最下行に総合判定
		if allOK {
			fb.FillRectangle(0, h-24, w, 24, green)
			text(fb, labelX, h-6, "ALL OK", black)
		} else {
			left := 0
			for _, c := range checks {
				if !c.ok {
					left++
				}
			}
			text(fb, labelX, h-6, itoa(left)+" check(s) left", yellow)
		}
		fb.Display()

		// 約 30fps に揃える
		if d := 33*time.Millisecond - time.Since(frameStart); d > 0 {
			time.Sleep(d)
		}
	}
}

// text は (x, y) に文字列を描き、描いた文字列の右端の x を返す。
func text(fb *badge.Framebuffer, x, y int16, s string, c color.RGBA) int16 {
	tinyfont.WriteLine(fb, font, x, y, s, c)
	_, width := tinyfont.LineWidth(font, s)
	return x + int16(width)
}

// state は判定済みなら緑、未確認なら黄色を返す。
func state(ok bool) color.RGBA {
	if ok {
		return green
	}
	return yellow
}

func pressedText(p bool) string {
	if p {
		return "pressed"
	}
	return "-"
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

const hexDigits = "0123456789abcdef"

func hex8(v uint8) string {
	return string([]byte{hexDigits[v>>4], hexDigits[v&0xf]})
}

func hex32(v uint32) string {
	var b [8]byte
	for i := range b {
		b[i] = hexDigits[v>>(28-4*i)&0xf]
	}
	return string(b[:])
}
