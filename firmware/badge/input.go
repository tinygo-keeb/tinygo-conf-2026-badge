package badge

import "machine"

// Button は GND に落ちるプッシュスイッチ。
type Button struct {
	Pin machine.Pin
}

// Configure は内部プルアップ付き入力として設定する。
func (b Button) Configure() {
	b.Pin.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
}

// Pressed は押されていれば true。
func (b Button) Pressed() bool {
	return !b.Pin.Get()
}

// Joystick はアナログジョイスティック (X/Y + プッシュボタン)。
type Joystick struct {
	x, y machine.ADC
	btn  Button
}

// NewJoystick は ADC とボタンを初期化したジョイスティックを返す。
func NewJoystick() *Joystick {
	machine.InitADC()
	j := &Joystick{
		x:   machine.ADC{Pin: JOY_X},
		y:   machine.ADC{Pin: JOY_Y},
		btn: Button{Pin: JOY_BTN},
	}
	j.x.Configure(machine.ADCConfig{})
	j.y.Configure(machine.ADCConfig{})
	j.btn.Configure()
	return j
}

// Raw は X/Y の ADC 生値 (0..65535) を返す。
func (j *Joystick) Raw() (x, y uint16) {
	return j.x.Get(), j.y.Get()
}

// Read は X/Y を -1000..1000 に正規化した値 (中立 = 0) と、ボタン状態を返す。
func (j *Joystick) Read() (x, y int, pressed bool) {
	rx, ry := j.Raw()
	x = (int(rx) - 32768) * 1000 / 32768
	y = (int(ry) - 32768) * 1000 / 32768
	return x, y, j.btn.Pressed()
}

// Pressed はジョイスティックのボタンが押されていれば true。
func (j *Joystick) Pressed() bool {
	return j.btn.Pressed()
}

// NewButtons は SW1, SW2 を初期化して返す。
func NewButtons() (sw1, sw2 Button) {
	sw1 = Button{Pin: BUTTON1}
	sw2 = Button{Pin: BUTTON2}
	sw1.Configure()
	sw2.Configure()
	return sw1, sw2
}
