package badge

import (
	"device/esp"
	"machine"
	"runtime/volatile"
	"time"
	"unsafe"
)

// ConfigureI2C は machine.I2C.Configure 相当の初期化を行う。
//
// machine.I2C.Configure は最後にバスクリア (SCL パルス 9 個) を発行し、
// ハードウェアが完了ビットを落とすまで無限に待つ。SDA/SCL が浮いている
// (プルアップがない) バスでは busy 判定のまま完了せず、そこで永久に止まる
// ことがある。また Configure はピン設定を上書きするため、事前に内部
// プルアップを有効にしても外れてしまう。
//
// この関数は同じ手順を、内部プルアップの有効化 (pullup=true) とバスクリアの
// 有限待ちに置き換えて実装している。
func ConfigureI2C(bus *machine.I2C, cfg machine.I2CConfig, pullup bool) {
	if cfg.Frequency == 0 {
		cfg.Frequency = 400 * machine.KHz
	}
	b := bus.Bus

	// ペリフェラルクロックとリセット
	var sclSig, sdaSig uint32
	if bus == machine.I2C0 {
		esp.SYSTEM.SetPERIP_RST_EN0_I2C_EXT0_RST(1)
		esp.SYSTEM.SetPERIP_CLK_EN0_I2C_EXT0_CLK_EN(1)
		esp.SYSTEM.SetPERIP_RST_EN0_I2C_EXT0_RST(0)
		sclSig, sdaSig = machine.I2CEXT0_SCL_OUT_IDX, machine.I2CEXT0_SDA_OUT_IDX
	} else {
		esp.SYSTEM.SetPERIP_RST_EN0_I2C_EXT1_RST(1)
		esp.SYSTEM.SetPERIP_CLK_EN0_I2C_EXT1_CLK_EN(1)
		esp.SYSTEM.SetPERIP_RST_EN0_I2C_EXT1_RST(0)
		sclSig, sdaSig = machine.I2CEXT1_SCL_OUT_IDX, machine.I2CEXT1_SDA_OUT_IDX
	}
	b.INT_CLR.Set(0x3fff)
	b.INT_ENA.ClearBits(0x3fff)

	const clkSrc = 40_000_000 // XTAL
	b.SetCLK_CONF_SCLK_SEL(0) // XTAL
	b.SetCLK_CONF_SCLK_ACTIVE(1)
	b.SetCLK_CONF_SCLK_DIV_NUM(clkSrc / (cfg.Frequency * 1024))
	b.SetCTR_CLK_EN(1)
	b.FILTER_CFG.Set(0x377)

	// ピン: オープンドレイン出力 + 入力を I2C 信号に接続
	configureI2CPin(cfg.SDA, sdaSig, pullup)
	b.SetCTR_SDA_FORCE_OUT(1)
	configureI2CPin(cfg.SCL, sclSig, pullup)
	b.SetCTR_SCL_FORCE_OUT(1)

	// タイミング (machine.I2C.initFrequency と同じ計算)
	clkmDiv := clkSrc/(cfg.Frequency*1024) + 1
	sclkFreq := clkSrc / clkmDiv
	halfCycle := sclkFreq / cfg.Frequency / 2
	sclWaitHigh := uint32(0)
	if cfg.Frequency > 50000 {
		sclWaitHigh = halfCycle / 8
	}
	b.SetSCL_LOW_PERIOD(halfCycle - 1)
	b.SetSCL_HIGH_PERIOD(halfCycle - sclWaitHigh)
	b.SetSCL_HIGH_PERIOD_SCL_WAIT_HIGH_PERIOD(25)
	b.SetSCL_RSTART_SETUP_TIME(halfCycle)
	b.SetSCL_STOP_SETUP_TIME(halfCycle)
	b.SetSCL_START_HOLD_TIME(halfCycle - 1)
	b.SetSCL_STOP_HOLD_TIME(halfCycle - 1)
	b.SetSDA_SAMPLE_TIME(halfCycle / 2)
	b.SetSDA_HOLD_TIME(halfCycle / 4)

	// マスターモード開始
	b.SetFIFO_CONF_NONFIFO_EN(0)
	b.SetFIFO_CONF_RX_FIFO_RST(1)
	b.SetFIFO_CONF_RX_FIFO_RST(0)
	b.SetFIFO_CONF_TX_FIFO_RST(1)
	b.SetFIFO_CONF_TX_FIFO_RST(0)
	b.TO.Set(0x10)
	b.CTR.Set(0x113) // SDA/SCL_FORCE_OUT, MS_MODE, CLK_EN
	b.SetCTR_CONF_UPGATE(1)

	// FSM リセットとバスクリア (有限待ち)
	b.SetCTR_FSM_RST(1)
	b.SetSCL_SP_CONF_SCL_RST_SLV_NUM(9)
	b.SetSCL_SP_CONF_SCL_RST_SLV_EN(1)
	b.SetSCL_STRETCH_CONF_SLAVE_SCL_STRETCH_EN(1)
	b.SetCTR_CONF_UPGATE(1)
	deadline := time.Now().Add(5 * time.Millisecond)
	for b.GetSCL_SP_CONF_SCL_RST_SLV_EN() != 0 && time.Now().Before(deadline) {
	}
	b.SetSCL_SP_CONF_SCL_RST_SLV_EN(0)
	b.SetSCL_SP_CONF_SCL_RST_SLV_NUM(0)
	b.SetCTR_CONF_UPGATE(1)
}

// configureI2CPin は pin をオープンドレイン出力にし、GPIO マトリクスで
// I2C の出力信号と入力信号を接続する。machine.Pin.configure + initPins 相当。
func configureI2CPin(pin machine.Pin, signal uint32, pullup bool) {
	// IO_MUX: GPIO 機能、入力有効、ドライブ 20mA、必要ならプルアップ
	io := uint32(1)<<esp.IO_MUX_GPIO_MCU_SEL_Pos | esp.IO_MUX_GPIO_FUN_IE | uint32(2)<<esp.IO_MUX_GPIO_FUN_DRV_Pos
	if pullup {
		io |= esp.IO_MUX_GPIO_FUN_WPU
	}
	ioMuxReg(pin).Set(io)

	// 出力有効
	if pin < 32 {
		esp.GPIO.ENABLE_W1TS.Set(1 << pin)
	} else {
		esp.GPIO.ENABLE1_W1TS.Set(1 << (pin - 32))
	}
	// 出力信号: I2C SDA/SCL、入力信号: このピン
	gpioReg(&esp.GPIO.FUNC0_OUT_SEL_CFG, pin).Set(signal)
	gpioReg(&esp.GPIO.FUNC0_IN_SEL_CFG, machine.Pin(signal)).Set(esp.GPIO_FUNC_IN_SEL_CFG_SEL | uint32(pin)<<esp.GPIO_FUNC_IN_SEL_CFG_IN_SEL_Pos)
	// オープンドレイン
	gpioReg(&esp.GPIO.PIN0, pin).SetBits(esp.GPIO_PIN_PAD_DRIVER)
	pin.High()
}

func ioMuxReg(pin machine.Pin) *volatile.Register32 {
	return (*volatile.Register32)(unsafe.Add(unsafe.Pointer(&esp.IO_MUX.GPIO0), uintptr(pin)*4))
}

func gpioReg(base *volatile.Register32, idx machine.Pin) *volatile.Register32 {
	return (*volatile.Register32)(unsafe.Add(unsafe.Pointer(base), uintptr(idx)*4))
}

// EnablePullup はピンの現在の設定を変えずに内部プルアップだけを有効にする。
// machine.Pin.Configure はペリフェラルに割り当てたピンのプルアップを設定
// できないため、IO_MUX レジスタを直接操作する。
func EnablePullup(p machine.Pin) {
	reg := ioMuxReg(p)
	reg.SetBits(esp.IO_MUX_GPIO_FUN_WPU)
	reg.ClearBits(esp.IO_MUX_GPIO_FUN_WPD)
}

// ResetI2C は I2C コントローラの FSM と TX/RX FIFO をリセットする。
//
// TinyGo の ESP32 I2C ドライバは NACK などでトランザクションが失敗しても
// FIFO をリセットしないため、未送信のバイトが FIFO に残り、次の
// トランザクションからデータがずれる。ESP-IDF は毎トランザクション前に
// FIFO をリセットしているので、それに倣ってエラー後に呼ぶ。
func ResetI2C(bus *machine.I2C) {
	bus.Bus.SetCTR_FSM_RST(1)
	bus.Bus.SetFIFO_CONF_TX_FIFO_RST(1)
	bus.Bus.SetFIFO_CONF_TX_FIFO_RST(0)
	bus.Bus.SetFIFO_CONF_RX_FIFO_RST(1)
	bus.Bus.SetFIFO_CONF_RX_FIFO_RST(0)
	bus.Bus.SetCTR_CONF_UPGATE(1)
}

// I2C コマンドレジスタ (COMDn) の値。TinyGo の machine パッケージと同じ定義。
const (
	i2cCmdRestart = 6 << 11
	i2cCmdWrite   = 1<<11 | 1<<8 // WRITE + ack_check_en
	i2cCmdStop    = 2 << 11
)

// ProbeI2C は addr にアドレスバイトだけを書き込み (データなし)、
// ACK が返れば true を返す。i2cdetect の "quick write" と同じ動作。
//
// machine.I2C.Tx を使わずレジスタを直接操作しているのは次の理由による。
//   - Tx は読み出し時のアドレス NACK をエラーにしないので、読み出しでは
//     全アドレスが応答ありに見える。
//   - Tx で 1 バイト書き込むと、NACK 時に未送信のデータが TX FIFO に残り、
//     次のトランザクションでジェネラルコール (アドレス 0x00) として送られて
//     しまう。AHT21B はこれに ACK するため、以降の全アドレスが応答ありになる。
//
// バスは事前に ConfigureI2C などで初期化しておくこと。
func ProbeI2C(bus *machine.I2C, addr uint16) bool {
	const (
		intNACK     = esp.I2C_INT_RAW_NACK_INT_RAW_Msk
		intComplete = esp.I2C_INT_RAW_TRANS_COMPLETE_INT_RAW_Msk
		intTimeout  = esp.I2C_INT_RAW_TIME_OUT_INT_RAW_Msk
		intArbLost  = esp.I2C_INT_RAW_ARBITRATION_LOST_INT_RAW_Msk
		intMask     = intNACK | intComplete | intTimeout | intArbLost |
			esp.I2C_INT_RAW_END_DETECT_INT_RAW_Msk
	)
	b := bus.Bus

	// 前のトランザクションの STOP が完了するのを待ってからリセットする。
	// NACK 検出直後に次を始めると、前の STOP 完了 (TRANS_COMPLETE) が
	// 次のプローブの成功に見えてしまう。
	waitI2CIdle(bus, time.Millisecond)
	ResetI2C(bus)

	// RSTART, WRITE (アドレス 1 バイト、ACK チェックあり), STOP
	b.SetDATA_FIFO_RDATA(uint32(addr&0x7f) << 1)
	b.COMD0.Set(i2cCmdRestart)
	b.COMD1.Set(i2cCmdWrite | 1)
	b.COMD2.Set(i2cCmdStop)
	b.SetCTR_CONF_UPGATE(1)

	// 割り込みフラグのクリアは開始直前に行う
	b.INT_CLR.Set(intMask)
	for i := 0; i < 100 && b.INT_RAW.Get()&intMask != 0; i++ {
	}
	b.SetCTR_TRANS_START(1)

	ack := false
	deadline := time.Now().Add(20 * time.Millisecond)
	for {
		raw := b.INT_RAW.Get()
		if raw&(intNACK|intTimeout|intArbLost) != 0 {
			break
		}
		if raw&intComplete != 0 {
			ack = true
			break
		}
		if time.Now().After(deadline) {
			break
		}
	}
	// 終端処理が終わるまで待ってからフラグを消す
	waitI2CIdle(bus, time.Millisecond)
	b.INT_CLR.Set(intMask)
	if !ack {
		ResetI2C(bus)
	}
	return ack
}

// waitI2CIdle はバスが busy でなくなるまで最大 d 待つ。
func waitI2CIdle(bus *machine.I2C, d time.Duration) {
	deadline := time.Now().Add(d)
	for bus.Bus.GetSR_BUS_BUSY() != 0 && time.Now().Before(deadline) {
	}
}

// ScanI2C は 0x08..0x77 の各アドレスを ProbeI2C で調べ、
// 応答したアドレスの一覧を返す。
func ScanI2C(bus *machine.I2C) []uint16 {
	var found []uint16
	for addr := uint16(0x08); addr < 0x78; addr++ {
		if ProbeI2C(bus, addr) {
			found = append(found, addr)
		}
	}
	return found
}
