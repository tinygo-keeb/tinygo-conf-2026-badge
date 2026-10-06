// Grove コネクタにつないだ DHT20 温湿度センサーの値をシリアルに出力する。
// DHT20 は AHT20 と同じチップ (I2C アドレス 0x38) なので aht20 ドライバを使う。
//
// 起動時に Grove バスをスキャンして 0x38 が見えるかを表示し、
// その後 1 秒ごとに温度と湿度を出力する。
package main

import (
	"time"

	"tinygo.org/x/drivers/aht20"

	"github.com/sago35/tinygo-conf-2026-badge/firmware/badge"
)

func main() {
	time.Sleep(2 * time.Second)
	badge.ConfigureGroveI2C()

	print("grove i2c scan:")
	found := badge.ScanI2C(badge.GroveI2C)
	if len(found) == 0 {
		print(" (none)")
	}
	for _, addr := range found {
		print(" 0x", hex(addr))
	}
	println()

	sensor := aht20.New(badge.GroveI2C)
	sensor.Configure()

	for {
		if err := sensor.Read(); err != nil {
			println("dht20 read error:", err.Error())
		} else {
			// DeciCelsius / DeciRelHumidity は 1/10 単位の整数
			t := sensor.DeciCelsius()
			h := sensor.DeciRelHumidity()
			println("temp:", deci(t), "C  humidity:", deci(h), "%")
		}
		time.Sleep(time.Second)
	}
}

func hex(v uint16) string {
	const digits = "0123456789abcdef"
	return string([]byte{digits[v>>4&0xf], digits[v&0xf]})
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
