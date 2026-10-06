// Package i2s は ESP32-S3 の I2S0 を使った 16bit ステレオの音声出力ドライバ。
//
// TinyGo の machine パッケージは ESP32-S3 の I2S を提供していないため、
// device/esp のレジスタ定義で直接実装している。ESP32-S3 の I2S は CPU から
// 直接 FIFO に書けず GDMA 経由でしか送れないので、GDMA のディスクリプタを
// リング状につないで連続送信し、Write はそのリングのバッファを順に
// 埋めていく。レジスタ操作の手順は ESP-IDF v5.1 の hal/esp32s3/i2s_ll.h、
// gdma_ll.h、driver/i2s/i2s_std.c に合わせている。
//
// 想定するフォーマットは Philips 標準 I2S、16bit データ、2 チャネル。
// スロット幅は 16 (WS 1 周期 = 32 BCLK) か 32 (64 BCLK) を選べる。
// MAX98357 などの一般的な I2S アンプで使える。
package i2s

import (
	"device/esp"
	"errors"
	"machine"
	"runtime/volatile"
	"time"
	"unsafe"
)

// GPIO マトリクスの出力信号番号 (soc/esp32s3/gpio_sig_map.h)。
const (
	sigI2S0OBCKOut = 22 // I2S0O_BCK_OUT_IDX
	sigI2S0OWSOut  = 24 // I2S0O_WS_OUT_IDX
	sigI2S0OSDOut  = 25 // I2S0O_SD_OUT_IDX
)

const (
	gdmaPeriphI2S0 = 3 // SOC_GDMA_TRIG_PERIPH_I2S0
	pllClock       = 160_000_000
	mclkDividerMax = 511 // I2S_LL_MCLK_DIVIDER_MAX (9bit)
)

var (
	ErrInvalidConfig = errors.New("i2s: invalid config")
	ErrNotConfigured = errors.New("i2s: not configured")
	ErrTimeout       = errors.New("i2s: DMA is not advancing (timeout)")
)

// WaitTimeout は Write がバッファの空きを待つ最大時間。
// DMA が動いていないときに永久に止まらないための保険。
const WaitTimeout = 500 * time.Millisecond

// Config は I2S 出力の設定。
type Config struct {
	BCLK machine.Pin // ビットクロック
	LRC  machine.Pin // ワードセレクト (LRCLK / WS)
	DIN  machine.Pin // シリアルデータ (アンプ側の DIN)

	// SampleRate はサンプリング周波数 (Hz)。0 なら 16000。
	SampleRate uint32
	// SlotBits は 1 チャネル (スロット) あたりの BCLK 数で 16 か 32。0 なら 16。
	// データは常に 16bit で、32 のときは下位 16bit が 0 で埋められる。
	// BCLK は SampleRate * 2 * SlotBits になる (16: 32fs、32: 64fs)。
	SlotBits int
	// BufferFrames は DMA バッファ 1 個あたりのフレーム数 (L+R で 1 フレーム)。
	// 0 なら 256。バッファのバイト数 (4 * BufferFrames) は 4092 以下にすること。
	BufferFrames int
	// BufferCount は DMA バッファの数。0 なら 4。2 以上にすること。
	BufferCount int
}

// GDMA のリンクリストディスクリプタ (hal/dma_types.h の dma_descriptor_t)。
// 12 バイトで 4 バイト境界に置く必要がある。
type dmaDescriptor struct {
	dw0    volatile.Register32 // size[11:0] length[23:12] err_eof[28] suc_eof[30] owner[31]
	buffer uintptr
	next   uintptr
}

const (
	dmaSucEOF = 1 << 30
	dmaOwner  = 1 << 31
)

// Device は I2S0 の送信チャネル。
type Device struct {
	cfg        Config
	descs      []dmaDescriptor
	bufs       [][]byte
	bufBytes   int
	write      int // 次に書き込むバッファの番号
	offset     int // そのバッファ内の書き込み済みバイト数
	configured bool
}

// New は I2S0 のデバイスを返す。Configure を呼ぶまでハードウェアには触らない。
func New() *Device {
	return &Device{}
}

// Configure は I2S0 と GDMA を初期化して送信を開始する。
// バッファは無音 (0) で埋められているので、Write するまで無音が出続ける。
func (d *Device) Configure(cfg Config) error {
	if cfg.SampleRate == 0 {
		cfg.SampleRate = 16000
	}
	if cfg.BufferFrames == 0 {
		cfg.BufferFrames = 256
	}
	if cfg.BufferCount == 0 {
		cfg.BufferCount = 4
	}
	if cfg.SlotBits == 0 {
		cfg.SlotBits = 16
	}
	if cfg.BCLK == machine.NoPin || cfg.LRC == machine.NoPin || cfg.DIN == machine.NoPin ||
		cfg.BufferCount < 2 || cfg.BufferFrames*4 > 4092 ||
		(cfg.SlotBits != 16 && cfg.SlotBits != 32) {
		return ErrInvalidConfig
	}
	if d.configured {
		d.Stop()
	}
	d.cfg = cfg
	d.bufBytes = cfg.BufferFrames * 4
	d.allocBuffers()

	d.initI2S()
	d.initGDMA()
	d.routePins()
	d.start()

	d.write = 0
	d.offset = 0
	d.configured = true
	return nil
}

func (d *Device) allocBuffers() {
	n := d.cfg.BufferCount
	d.descs = make([]dmaDescriptor, n)
	d.bufs = make([][]byte, n)
	for i := 0; i < n; i++ {
		d.bufs[i] = make([]byte, d.bufBytes)
		d.descs[i].buffer = uintptr(unsafe.Pointer(&d.bufs[i][0]))
		d.descs[i].next = uintptr(unsafe.Pointer(&d.descs[(i+1)%n]))
		d.descs[i].dw0.Set(uint32(d.bufBytes) | uint32(d.bufBytes)<<12 | dmaSucEOF | dmaOwner)
	}
}

// initI2S は I2S0 を Philips 標準モード、16bit、2 スロットのマスター送信に設定する
// (i2s_hal_std_set_tx_slot / i2s_hal_set_tx_clock 相当)。
func (d *Device) initI2S() {
	hw := esp.I2S0

	// ペリフェラルクロックとリセット
	esp.SYSTEM.SetPERIP_CLK_EN0_I2S0_CLK_EN(1)
	esp.SYSTEM.SetPERIP_RST_EN0_I2S0_RST(1)
	esp.SYSTEM.SetPERIP_RST_EN0_I2S0_RST(0)
	hw.SetTX_CLKM_CONF_CLK_EN(1)

	// スロット設定
	slot := uint32(d.cfg.SlotBits)
	hw.SetTX_CONF_TX_RESET(1)
	hw.SetTX_CONF_TX_RESET(0)
	hw.SetTX_CONF_TX_SLAVE_MOD(0)
	hw.SetTX_CONF1_TX_BITS_MOD(16 - 1)        // データ 16bit
	hw.SetTX_CONF1_TX_TDM_CHAN_BITS(slot - 1) // スロット幅
	hw.SetTX_CONF1_TX_MSB_SHIFT(1)            // Philips (WS の 1 BCLK 後にデータ開始)
	hw.SetTX_CONF1_TX_TDM_WS_WIDTH(slot - 1)  // WS の半周期 = スロット幅
	hw.SetTX_CONF_TX_MONO(0)
	hw.SetTX_CONF_TX_CHAN_EQUAL(0)
	// スロット総数 2、有効スロットは ch0/ch1 のみ (リセット値は 16 スロットすべて有効)
	hw.TX_TDM_CTRL.Set(hw.TX_TDM_CTRL.Get()&0xffff0000 | 0x3)
	hw.SetTX_TDM_CTRL_TX_TDM_TOT_CHAN_NUM(2 - 1)
	hw.SetTX_TDM_CTRL_TX_TDM_SKIP_MSK_EN(0)
	hw.SetTX_CONF1_TX_HALF_SAMPLE_BITS(slot - 1)
	hw.SetTX_CONF_TX_WS_IDLE_POL(0)
	hw.SetTX_CONF_TX_BIT_ORDER(0)
	hw.SetTX_CONF_TX_LEFT_ALIGN(0)
	hw.SetTX_CONF_TX_BIG_ENDIAN(0)
	// 標準 (TDM) モード、PCM 圧縮なし
	hw.SetTX_CONF_TX_PDM_EN(0)
	hw.SetTX_CONF_TX_TDM_EN(1)
	hw.SetTX_PCM2PDM_CONF_PCM2PDM_CONV_EN(0)
	hw.SetTX_CONF_TX_PCM_CONF(0)
	hw.SetTX_CONF_TX_PCM_BYPASS(1)

	// クロック: PLL 160MHz を分周して MCLK = fs*256、BCLK = fs*2*SlotBits
	hw.SetTX_CLKM_CONF_TX_CLK_ACTIVE(1)
	hw.SetRX_CLKM_CONF_MCLK_SEL(0)
	hw.SetTX_CLKM_CONF_TX_CLK_SEL(2) // PLL_F160M
	mclk := d.cfg.SampleRate * 256
	bclk := d.cfg.SampleRate * 2 * slot
	integ, numer, denom := calcMCLKDiv(pllClock, mclk)
	// i2s_ll_tx_set_mclk: 低いレートから切り替えたときの誤差対策として
	// 一度あり得ない係数を書いてから本来の値を設定する
	setTXClkDiv(7, 317, 7, 3, 0)
	var x, y, z, yn1 uint32
	if numer != 0 && denom != 0 {
		if numer*2 > denom {
			yn1 = 1
			z = denom - numer
		} else {
			z = numer
		}
		x = denom/z - 1
		y = denom % z
	}
	setTXClkDiv(integ, x, y, z, yn1)
	hw.SetTX_CONF1_TX_BCK_DIV_NUM(mclk/bclk - 1)
}

func setTXClkDiv(integ, x, y, z, yn1 uint32) {
	hw := esp.I2S0
	hw.SetTX_CLKM_CONF_TX_CLKM_DIV_NUM(integ)
	hw.TX_CLKM_DIV_CONF.Set(z | y<<9 | x<<18 | yn1<<27)
}

// calcMCLKDiv は sclk/mclk を 整数 + numer/denom の形に近似する
// (i2s_hal_calc_mclk_precise_division を整数演算にしたもの)。
func calcMCLKDiv(sclk, mclk uint32) (integ, numer, denom uint32) {
	integ = sclk / mclk
	diff := sclk % mclk
	denom = 1
	if diff == 0 {
		return
	}
	// 小数部が 1 - 1/(2*max) を超えるなら切り上げ
	if uint64(diff)*uint64(2*mclkDividerMax) > uint64(mclk)*uint64(2*mclkDividerMax-1) {
		integ++
		return
	}
	best := uint64(1) << 62
	for a := uint32(2); a <= mclkDividerMax; a++ {
		b := uint32((uint64(a)*uint64(diff) + uint64(mclk)/2) / uint64(mclk))
		ma := uint64(diff) * uint64(a)
		mb := uint64(mclk) * uint64(b)
		var e uint64
		if ma > mb {
			e = ma - mb
		} else {
			e = mb - ma
		}
		if e < best {
			best = e
			denom, numer = a, b
			if e == 0 {
				break
			}
		}
	}
	return
}

// initGDMA は GDMA チャネル 0 を I2S0 の送信に接続する。
func (d *Device) initGDMA() {
	dma := esp.DMA
	esp.SYSTEM.SetPERIP_CLK_EN1_DMA_CLK_EN(1)
	esp.SYSTEM.SetPERIP_RST_EN1_DMA_RST(1)
	esp.SYSTEM.SetPERIP_RST_EN1_DMA_RST(0)
	dma.SetMISC_CONF_CLK_EN(1)

	dma.SetOUT_CONF0_CH0_OUT_RST(1)
	dma.SetOUT_CONF0_CH0_OUT_RST(0)
	dma.SetOUT_CONF0_CH0_OUTDSCR_BURST_EN(1)
	dma.SetOUT_CONF0_CH0_OUT_DATA_BURST_EN(1)
	dma.SetOUT_CONF0_CH0_OUT_EOF_MODE(1) // データが FIFO から送出された時点で EOF
	dma.SetOUT_CONF0_CH0_OUT_AUTO_WRBACK(0)
	dma.SetOUT_CONF1_CH0_OUT_CHECK_OWNER(0)
	dma.SetOUT_PRI_CH0_TX_PRI(0)
	dma.SetOUT_PERI_SEL_CH0_PERI_OUT_SEL(gdmaPeriphI2S0)
	dma.OUT_INT_ENA_CH0.Set(0)
	dma.OUT_INT_CLR_CH0.Set(0xffffffff)
}

// routePins は GPIO マトリクスで各ピンに I2S0 の出力信号をつなぐ。
func (d *Device) routePins() {
	routeOutput(d.cfg.BCLK, sigI2S0OBCKOut)
	routeOutput(d.cfg.LRC, sigI2S0OWSOut)
	routeOutput(d.cfg.DIN, sigI2S0OSDOut)
}

func routeOutput(pin machine.Pin, signal uint32) {
	pin.Configure(machine.PinConfig{Mode: machine.PinOutput})
	reg := (*volatile.Register32)(unsafe.Add(unsafe.Pointer(&esp.GPIO.FUNC0_OUT_SEL_CFG), uintptr(pin)*4))
	reg.Set(signal)
}

// start は i2s_tx_channel_start 相当。
func (d *Device) start() {
	hw := esp.I2S0
	dma := esp.DMA
	hw.SetTX_CONF_TX_RESET(1)
	hw.SetTX_CONF_TX_RESET(0)
	dma.SetOUT_CONF0_CH0_OUT_RST(1)
	dma.SetOUT_CONF0_CH0_OUT_RST(0)
	hw.SetTX_CONF_TX_FIFO_RESET(1)
	hw.SetTX_CONF_TX_FIFO_RESET(0)
	dma.OUT_INT_CLR_CH0.Set(0xffffffff)
	dma.SetOUT_LINK_CH0_OUTLINK_ADDR(uint32(uintptr(unsafe.Pointer(&d.descs[0]))) & 0xfffff)
	dma.SetOUT_LINK_CH0_OUTLINK_START(1)
	hw.SetTX_CONF_TX_UPDATE(1)
	for i := 0; i < 100000 && hw.GetTX_CONF_TX_UPDATE() != 0; i++ {
	}
	hw.SetTX_CONF_TX_START(1)
}

// Stop は送信を止める。再開するには Configure を呼び直す。
func (d *Device) Stop() {
	esp.I2S0.SetTX_CONF_TX_START(0)
	esp.DMA.SetOUT_LINK_CH0_OUTLINK_STOP(1)
	d.configured = false
}

// SampleRate は設定されたサンプリング周波数を返す。
func (d *Device) SampleRate() uint32 {
	return d.cfg.SampleRate
}

// BufferFrames は DMA バッファ 1 個あたりのフレーム数を返す。
func (d *Device) BufferFrames() int {
	return d.cfg.BufferFrames
}

// playing は DMA が現在読み出しているバッファの番号を返す。
func (d *Device) playing() int {
	eof := uintptr(esp.DMA.GetOUT_EOF_DES_ADDR_CH0())
	if eof == 0 {
		return 0 // まだ 1 個も送り終えていない = 先頭を再生中
	}
	base := uintptr(unsafe.Pointer(&d.descs[0]))
	i := int((eof - base) / unsafe.Sizeof(dmaDescriptor{}))
	return (i + 1) % len(d.descs)
}

// waitBuffer は書き込み先バッファを DMA が読み終えるまで待つ。
// WaitTimeout 以内に空かなければ ErrTimeout を返す。
func (d *Device) waitBuffer() error {
	deadline := time.Now().Add(WaitTimeout)
	for d.playing() == d.write {
		if time.Now().After(deadline) {
			return ErrTimeout
		}
	}
	return nil
}

// Writable は Write がブロックせずに書き込めるフレーム数を返す。
// DMA が読み終えたバッファ (次に再生される順) の空き容量の合計。
// ゲームなどで描画の合間に音を補充するときに、待たずに書ける量を知るために使う。
func (d *Device) Writable() int {
	if !d.configured {
		return 0
	}
	n := len(d.bufs)
	free := (d.playing() - d.write + n) % n
	if free == 0 {
		return 0
	}
	return free*d.cfg.BufferFrames - d.offset/4
}

// Write は L, R, L, R, ... の順に並んだ 16bit サンプルを送信する。
// バッファが空くまでブロックするので、再生速度に合わせて進む。
// バッファ 1 個分に満たない端数は書きかけのまま残り、次の Write で続きから
// 埋められる。その間に DMA がそのバッファに到達した場合に前回のデータが
// 鳴らないよう、書きかけのバッファの残りは無音で埋めておく。
//
// 注意: DMA バッファはリング状に再生され続けるため、Write をやめても
// 最後に書いたデータが繰り返し鳴る。止めるときは Silence を呼ぶこと。
func (d *Device) Write(samples []int16) (int, error) {
	if !d.configured {
		return 0, ErrNotConfigured
	}
	n := 0
	for n < len(samples) {
		if d.offset == 0 {
			if err := d.waitBuffer(); err != nil {
				return n, err
			}
		}
		buf := d.bufs[d.write]
		for d.offset < d.bufBytes && n < len(samples) {
			v := uint16(samples[n])
			buf[d.offset] = byte(v)
			buf[d.offset+1] = byte(v >> 8)
			d.offset += 2
			n++
		}
		if d.offset >= d.bufBytes {
			d.advance()
		}
	}
	if d.offset > 0 {
		// 書きかけのバッファの残りを無音にしておく
		buf := d.bufs[d.write]
		for i := d.offset; i < d.bufBytes; i++ {
			buf[i] = 0
		}
	}
	return n, nil
}

func (d *Device) advance() {
	d.offset = 0
	d.write = (d.write + 1) % len(d.bufs)
}

// Flush は書きかけのバッファを (残りは無音のまま) 確定し、次のバッファに進む。
func (d *Device) Flush() {
	if !d.configured || d.offset == 0 {
		return
	}
	d.advance()
}

// Silence は書きかけのバッファを Flush したうえで、リング全体を無音で埋める。
// 呼び出し後は Write するまで無音になる。リング 1 周分の時間ブロックする。
func (d *Device) Silence() {
	if !d.configured {
		return
	}
	d.Flush()
	for i := 0; i < len(d.bufs); i++ {
		if d.waitBuffer() != nil {
			return
		}
		buf := d.bufs[d.write]
		for j := range buf {
			buf[j] = 0
		}
		d.advance()
	}
}

// DumpRegisters は I2S0 と GDMA チャネル 0 の主なレジスタと、
// 出力ピンの GPIO マトリクス設定をシリアルに出力する (デバッグ用)。
func (d *Device) DumpRegisters() {
	hw := esp.I2S0
	dma := esp.DMA
	println("i2s: TX_CONF", hex(hw.TX_CONF.Get()), "TX_CONF1", hex(hw.TX_CONF1.Get()),
		"TX_CLKM_CONF", hex(hw.TX_CLKM_CONF.Get()), "TX_CLKM_DIV_CONF", hex(hw.TX_CLKM_DIV_CONF.Get()))
	println("i2s: TX_TDM_CTRL", hex(hw.TX_TDM_CTRL.Get()), "STATE", hex(hw.STATE.Get()), "INT_RAW", hex(hw.INT_RAW.Get()))
	println("dma: OUT_CONF0", hex(dma.OUT_CONF0_CH0.Get()), "OUT_CONF1", hex(dma.OUT_CONF1_CH0.Get()),
		"PERI_SEL", hex(dma.OUT_PERI_SEL_CH0.Get()), "MISC_CONF", hex(dma.MISC_CONF.Get()))
	println("dma: OUT_LINK", hex(dma.OUT_LINK_CH0.Get()), "OUT_STATE", hex(dma.OUT_STATE_CH0.Get()),
		"OUT_INT_RAW", hex(dma.OUT_INT_RAW_CH0.Get()), "OUTFIFO_STATUS", hex(dma.OUTFIFO_STATUS_CH0.Get()))
	println("dma: OUT_DSCR", hex(dma.OUT_DSCR_CH0.Get()), "OUT_EOF_DES_ADDR", hex(dma.OUT_EOF_DES_ADDR_CH0.Get()),
		"desc0", hex(uint32(uintptr(unsafe.Pointer(&d.descs[0])))), "dw0", hex(d.descs[0].dw0.Get()),
		"buf0", hex(uint32(d.descs[0].buffer)), "next", hex(uint32(d.descs[0].next)))
	println("sys: PERIP_CLK_EN0", hex(esp.SYSTEM.PERIP_CLK_EN0.Get()), "PERIP_CLK_EN1", hex(esp.SYSTEM.PERIP_CLK_EN1.Get()))
	for _, p := range []machine.Pin{d.cfg.BCLK, d.cfg.LRC, d.cfg.DIN} {
		out := (*volatile.Register32)(unsafe.Add(unsafe.Pointer(&esp.GPIO.FUNC0_OUT_SEL_CFG), uintptr(p)*4)).Get()
		var en uint32
		if p < 32 {
			en = esp.GPIO.ENABLE.Get() >> p & 1
		} else {
			en = esp.GPIO.ENABLE1.Get() >> (p - 32) & 1
		}
		println("gpio", uint8(p), ": OUT_SEL", hex(out), "ENABLE", en)
	}
}

// CountToggles は各出力ピンの入力値を n 回読み、値が変化した回数を返す。
// ピンが出力でも入力バッファは有効なので、クロックが出ていれば変化が見える。
// 戻り値の順は BCLK, LRC, DIN。
func (d *Device) CountToggles(n int) (bclk, lrc, din int) {
	pins := [3]machine.Pin{d.cfg.BCLK, d.cfg.LRC, d.cfg.DIN}
	var counts [3]int
	for i, p := range pins {
		prev := p.Get()
		for j := 0; j < n; j++ {
			v := p.Get()
			if v != prev {
				counts[i]++
				prev = v
			}
		}
	}
	return counts[0], counts[1], counts[2]
}

func hex(v uint32) string {
	const digits = "0123456789abcdef"
	var b [10]byte
	b[0], b[1] = '0', 'x'
	for i := 0; i < 8; i++ {
		b[9-i] = digits[v&0xf]
		v >>= 4
	}
	return string(b[:])
}
