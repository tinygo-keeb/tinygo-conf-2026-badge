package badge

import (
	"machine"

	"tinygo.org/x/drivers/aht20"
)

// NewSensor は I2C1 を初期化し、AHT21B (AHT20 互換) のドライバを返す。
// 戻り値の Configure() は呼び出し済み。
func NewSensor() *aht20.Device {
	ConfigureSensorI2C()
	sensor := aht20.New(SensorI2C)
	sensor.Configure()
	return &sensor
}

// ConfigureSensorI2C は AHT21B 用の I2C1 を初期化する。
// モジュール側にプルアップ抵抗があるので内部プルアップは使わない。
func ConfigureSensorI2C() {
	ConfigureI2C(SensorI2C, machine.I2CConfig{
		Frequency: 400 * machine.KHz,
		SDA:       AHT21B_SDA,
		SCL:       AHT21B_SCL,
	}, false)
}

// GroveI2CFrequency は Grove コネクタ用 I2C のクロック周波数。
// 基板上にプルアップ抵抗がなく内部プルアップ (約 45kΩ) に頼ることが
// あるため、控えめに 100kHz にしている。
const GroveI2CFrequency = 100 * machine.KHz

// ConfigureGroveI2C は Grove コネクタ用の I2C0 を初期化する。
// 基板上にプルアップ抵抗がないため、SDA/SCL の内部プルアップを有効にする。
// これにより何もつないでいない状態でもバスが浮かず、スキャンなどが正常に動く。
func ConfigureGroveI2C() {
	ConfigureI2C(GroveI2C, machine.I2CConfig{
		Frequency: GroveI2CFrequency,
		SDA:       I2C_SDA,
		SCL:       I2C_SCL,
	}, true)
}
