// 赤外線の送受信。
//   - 受信モジュールで NEC フォーマットを受け取るとシリアルに出力する。
//   - SW1 を押すと NEC フォーマットで (address=0x10, command=0x01) を送信する。
//   - SW2 を押すと command=0x02 を送信する。
//
// 2 台のバッジを向かい合わせると相互に通信できる。
package main

import (
	"time"

	"tinygo.org/x/drivers/irremote"

	"github.com/sago35/tinygo-conf-2026-badge/firmware/badge"
)

const irAddress = 0x10

func main() {
	sw1, sw2 := badge.NewButtons()

	rx := badge.NewIRReceiver()
	rx.SetCommandHandler(func(d irremote.Data) {
		println("IR rx: addr=", d.Address, "cmd=", d.Command, "flags=", d.Flags)
	})

	tx, err := badge.NewIRTransmitter()
	if err != nil {
		println("ir tx init:", err.Error())
		select {}
	}

	prev1, prev2 := false, false
	for {
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
