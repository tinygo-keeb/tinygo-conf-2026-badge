// Grove コネクタ (I2C0) と AHT21B (I2C1) の両方のバスをスキャンし、
// 応答したアドレスをシリアルに出力する。AHT21B は 0x38 に見えるはず。
//
// USB シリアルは改行までバッファするため、途中経過も 1 行ずつ出す。
package main

import (
	"machine"
	"time"

	"github.com/sago35/tinygo-conf-2026-badge/firmware/badge"
)

func main() {
	time.Sleep(500 * time.Millisecond)
	println("i2cscan start")

	badge.ConfigureGroveI2C()
	println("grove i2c configured")
	badge.ConfigureSensorI2C()
	println("sensor i2c configured")

	for {
		report("Grove  (I2C0)", badge.GroveI2C)
		report("AHT21B (I2C1)", badge.SensorI2C)
		time.Sleep(3 * time.Second)
	}
}

func report(name string, bus *machine.I2C) {
	println("scanning", name, "...")
	start := time.Now()
	found := badge.ScanI2C(bus)
	elapsed := time.Since(start)
	print(name, ":")
	if len(found) == 0 {
		print(" (none)")
	}
	for _, addr := range found {
		print(" 0x", hex(addr))
	}
	println("  [", elapsed.Milliseconds(), "ms ]")
}

func hex(v uint16) string {
	const digits = "0123456789abcdef"
	return string([]byte{digits[v>>4&0xf], digits[v&0xf]})
}
