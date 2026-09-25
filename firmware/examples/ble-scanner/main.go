// BLE で周囲のデバイスをスキャンし、アドレス、RSSI、名前をシリアルに出力する。
// 同じアドレスは 1 回だけ表示する (RSSI の更新は表示しない)。
//
//	make flash-ble-scanner
package main

import (
	"time"

	"tinygo.org/x/bluetooth"
)

var adapter = bluetooth.DefaultAdapter

// 表示済みのアドレス
var seen [64]bluetooth.Address
var seenCount int

func main() {
	time.Sleep(2 * time.Second)
	println("enabling BLE...")
	must("enable BLE stack", adapter.Enable())

	println("scanning (address / RSSI / name)...")
	must("scan", adapter.Scan(func(a *bluetooth.Adapter, dev bluetooth.ScanResult) {
		for i := 0; i < seenCount; i++ {
			if seen[i] == dev.Address {
				return
			}
		}
		if seenCount < len(seen) {
			seen[seenCount] = dev.Address
			seenCount++
		}
		println(dev.Address.String(), dev.RSSI, dev.LocalName())
	}))
}

func must(action string, err error) {
	for err != nil {
		println("failed to " + action + ": " + err.Error())
		time.Sleep(time.Second)
	}
}
