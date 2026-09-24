package badge

import (
	"machine"
	"time"

	"tinygo.org/x/drivers/irremote"
)

// IRCarrierPeriod は赤外線 LED の変調キャリア (38kHz) の周期 (ns)。
const IRCarrierPeriod = 1_000_000_000 / 38_000

// IRTransmitter は赤外線 LED を 38kHz で変調して NEC フォーマットを送信する。
type IRTransmitter struct {
	pwm  *machine.LEDCPWM
	ch   uint8
	duty uint32
}

// NewIRTransmitter は PWM0 を 38kHz で初期化して赤外線送信機を返す。
func NewIRTransmitter() (*IRTransmitter, error) {
	pwm := machine.PWM0
	if err := pwm.Configure(machine.PWMConfig{Period: IRCarrierPeriod}); err != nil {
		return nil, err
	}
	ch, err := pwm.Channel(IR_LED)
	if err != nil {
		return nil, err
	}
	t := &IRTransmitter{pwm: pwm, ch: ch, duty: pwm.Top() / 3}
	t.carrier(false)
	return t, nil
}

func (t *IRTransmitter) carrier(on bool) {
	if on {
		t.pwm.Set(t.ch, t.duty)
	} else {
		t.pwm.Set(t.ch, 0)
	}
}

// mark はキャリアを on、space は off にして d だけ待つ。
func (t *IRTransmitter) mark(d time.Duration) {
	t.carrier(true)
	busyWait(d)
}

func (t *IRTransmitter) space(d time.Duration) {
	t.carrier(false)
	busyWait(d)
}

// SendNEC は NEC フォーマットで address / command を送信する。
// 8bit アドレスと 8bit コマンドを、それぞれ反転値と組にして送る。
func (t *IRTransmitter) SendNEC(address, command uint8) {
	t.SendNEC32(uint32(address)<<24 | uint32(^address)<<16 | uint32(command)<<8 | uint32(^command))
}

// SendNEC32 は 32bit のデータをそのまま NEC フォーマットで送信する (MSB first)。
// 拡張 NEC (16bit アドレス) を送りたい場合はこちらを使う。
func (t *IRTransmitter) SendNEC32(data uint32) {
	t.mark(9000 * time.Microsecond)
	t.space(4500 * time.Microsecond)
	for i := 0; i < 32; i++ {
		t.mark(562 * time.Microsecond)
		if data&(1<<(31-i)) != 0 {
			t.space(1687 * time.Microsecond)
		} else {
			t.space(562 * time.Microsecond)
		}
	}
	t.mark(562 * time.Microsecond)
	t.carrier(false)
}

func busyWait(d time.Duration) {
	start := time.Now()
	for time.Since(start) < d {
	}
}

// NewIRReceiver は赤外線受信モジュール用の NEC デコーダを返す。
// ir.SetCommandHandler(func(d irremote.Data) {...}) で受信を開始する。
func NewIRReceiver() *irremote.ReceiverDevice {
	ir := irremote.NewReceiver(IR_DATA)
	ir.Configure()
	return &ir
}
