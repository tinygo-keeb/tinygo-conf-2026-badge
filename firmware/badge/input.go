package badge

import (
	"machine"
	"time"
)

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
//
// Read の出力は -1000..1000 に正規化した値で、次の処理を通す。
//   - センター補正: Calibrate で実測した中立位置を 0 とする
//   - 可動範囲: 中心から Range カウント倒した位置を 1000 とする
//   - デッドゾーン: 中心から DeadZone 以内はドリフト防止のため 0 にする
//   - 飽和: 中心から Saturation 以上倒すと最大値 (半径 1000 の円周上) になる
//
// デッドゾーンと飽和は倒した距離 (半径方向) に対して掛けるので、
// 出力は常に半径 1000 の円の内側に収まり、斜めに倒しても円の端で止まる。
type Joystick struct {
	x, y machine.ADC
	btn  Button

	// センター位置の ADC 生値。Calibrate で更新される。初期値は中点。
	CenterX, CenterY uint16

	// Range は中心からいっぱいまで倒したときの ADC 生値の差分。
	// 本基板のスティックは可動範囲を絞ってあり、実測で全方向とも
	// 16000..17000 カウント程度 (examples/joyraw で計測)。既定 16000。
	Range int

	// DeadZone は 0 とみなす半径 (0..1000)。既定 50。
	// 静止時のノイズは ±100 カウント (= 約 6) なので十分に余裕がある。
	DeadZone int
	// Saturation は最大値に張り付く半径 (0..1000)。既定 850。
	Saturation int

	// InvertX / InvertY は軸の向きを反転する。配線の向きに合わせる。
	InvertX, InvertY bool
}

// NewJoystick は ADC とボタンを初期化したジョイスティックを返す。
// センターは ADC の中点と仮定するので、起動時に Calibrate を呼ぶことを推奨する。
func NewJoystick() *Joystick {
	machine.InitADC()
	j := &Joystick{
		x:          machine.ADC{Pin: JOY_X},
		y:          machine.ADC{Pin: JOY_Y},
		btn:        Button{Pin: JOY_BTN},
		CenterX:    32768,
		CenterY:    32768,
		Range:      16000,
		DeadZone:   50,
		Saturation: 850,
		// X のポテンショメータは右に倒すと電圧が下がる向きに配線されている (実機で確認)
		InvertX: true,
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

// Calibrate は d の間スティックを触らない前提で ADC を読み続け、
// その平均をセンター位置として記録する。
func (j *Joystick) Calibrate(d time.Duration) {
	var sx, sy, n uint32
	end := time.Now().Add(d)
	for time.Now().Before(end) || n == 0 {
		rx, ry := j.Raw()
		sx += uint32(rx)
		sy += uint32(ry)
		n++
		time.Sleep(5 * time.Millisecond)
	}
	j.CenterX = uint16(sx / n)
	j.CenterY = uint16(sy / n)
}

// axis は生値をセンター基準で -1000..1000 に正規化する。
// rng カウント倒した位置を 1000 とし、それ以上は 1000 に張り付く。
func axis(raw, center uint16, rng int, invert bool) int {
	if rng < 1 {
		rng = 1
	}
	v := (int(raw) - int(center)) * 1000 / rng
	if v > 1000 {
		v = 1000
	} else if v < -1000 {
		v = -1000
	}
	if invert {
		v = -v
	}
	return v
}

// Read は X/Y を -1000..1000 に正規化した値 (中立 = 0、右が +X、上が +Y) と、
// ボタン状態を返す。出力は半径 1000 の円の内側に収まる。
func (j *Joystick) Read() (x, y int, pressed bool) {
	rx, ry := j.Raw()
	x = axis(rx, j.CenterX, j.Range, j.InvertX)
	y = axis(ry, j.CenterY, j.Range, j.InvertY)

	// 半径方向にデッドゾーンと飽和を掛ける
	mag := isqrt(x*x + y*y)
	if mag <= j.DeadZone {
		return 0, 0, j.btn.Pressed()
	}
	span := j.Saturation - j.DeadZone
	if span < 1 {
		span = 1
	}
	scaled := (mag - j.DeadZone) * 1000 / span
	if scaled > 1000 {
		scaled = 1000
	}
	x = x * scaled / mag
	y = y * scaled / mag
	return x, y, j.btn.Pressed()
}

// Pressed はジョイスティックのボタンが押されていれば true。
func (j *Joystick) Pressed() bool {
	return j.btn.Pressed()
}

// isqrt は整数の平方根 (切り捨て) を返す。
func isqrt(v int) int {
	if v <= 0 {
		return 0
	}
	r := v
	for {
		n := (r + v/r) / 2
		if n >= r {
			return r
		}
		r = n
	}
}

// NewButtons は SW1, SW2 を初期化して返す。
func NewButtons() (sw1, sw2 Button) {
	sw1 = Button{Pin: BUTTON1}
	sw2 = Button{Pin: BUTTON2}
	sw1.Configure()
	sw2.Configure()
	return sw1, sw2
}
