// 赤外線の送受信。
//   - 受信モジュールで NEC フォーマットを受け取るとシリアルに出力する。
//   - SW1 を押すと NEC フォーマットで (address=0x10, command=0x01) を送信する。
//   - SW2 を押すと command=0x02 を送信する。
//
// バッジ 1 台でも、赤外線 LED の光が反射して自分の受信モジュールに届くため、
// 送信すると自分で受信する (ループバックテストになる)。
// 2 台のバッジを向かい合わせると相互に通信できる。
//
// 受信コールバックは GPIO 割り込みの中で呼ばれる。割り込み内で println
// (USB シリアルへの書き込み) を行うと出力が止まることがあるため、
// コールバックでは受信データを保存するだけにして、出力はメインループで行う。
package main

import (
	"runtime/volatile"
	"time"

	"tinygo.org/x/drivers/irremote"

	"github.com/sago35/tinygo-conf-2026-badge/firmware/badge"
)

const irAddress = 0x10

// 割り込みからメインループへの受け渡し用。すべて volatile にして、
// メインループ側で毎回メモリから読み直されるようにする。
// rxSeq が変わったら他のフィールドに新しいデータが入っている。
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

func main() {
	sw1, sw2 := badge.NewButtons()

	rx := badge.NewIRReceiver()
	rx.SetCommandHandler(onReceive)

	tx, err := badge.NewIRTransmitter()
	if err != nil {
		println("ir tx init:", err.Error())
		select {}
	}
	println("ir ready: SW1/SW2 to send, waiting for NEC frames")

	prev1, prev2 := false, false
	lastSeq := rxSeq.Get()
	for {
		if seq := rxSeq.Get(); seq != lastSeq {
			lastSeq = seq
			repeat := ""
			if rxFlags.Get()&uint16(irremote.DataFlagIsRepeat) != 0 {
				repeat = " (repeat)"
			}
			println("IR rx: addr=", rxAddress.Get(), "cmd=", rxCommand.Get(), "code=", rxCode.Get(), repeat)
		}

		p1, p2 := sw1.Pressed(), sw2.Pressed()
		if p1 && !prev1 {
			println("IR tx: cmd=1")
			tx.SendNEC(irAddress, 0x01)
		}
		if p2 && !prev2 {
			println("IR tx: cmd=2")
			tx.SendNEC(irAddress, 0x02)
		}
		prev1, prev2 = p1, p2
		time.Sleep(10 * time.Millisecond)
	}
}
