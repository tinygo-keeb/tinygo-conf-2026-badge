package badge

import "device/esp"

// MAC は eFuse に書かれた ESP32-S3 のベース MAC アドレス (チップごとに固有) を返す。
// ESP-IDF の esp_read_mac(ESP_MAC_BASE) と同じ並び。
func MAC() [6]byte {
	low := esp.EFUSE.RD_MAC_SPI_SYS_0.Get()
	high := esp.EFUSE.GetRD_MAC_SPI_SYS_1_MAC_1()
	return [6]byte{
		byte(high >> 8), byte(high),
		byte(low >> 24), byte(low >> 16), byte(low >> 8), byte(low),
	}
}

// SerialNumber はチップ固有の短い ID を返す (MAC の下位 3 バイトを 16 進 6 桁で)。
// ワークショップなどで複数のバッジを区別するための表示用。
func SerialNumber() string {
	const digits = "0123456789ABCDEF"
	m := MAC()
	var b [6]byte
	for i := 0; i < 3; i++ {
		b[2*i] = digits[m[3+i]>>4]
		b[2*i+1] = digits[m[3+i]&0xf]
	}
	return string(b[:])
}
