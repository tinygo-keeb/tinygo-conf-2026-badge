package badge

import (
	"github.com/sago35/tinygo-conf-2026-badge/firmware/i2s"
)

// AudioSampleRate は NewAudio が使うサンプリング周波数。
const AudioSampleRate = 16000

// AudioSlotBits は NewAudio が使うスロット幅 (16 か 32)。
// 16 なら BCLK = 32fs (512kHz)、32 なら BCLK = 64fs (1.024MHz)。
// MAX98357A はどちらも受け付けるはずだが、動かない場合は切り替えて試す。
const AudioSlotBits = 16

// NewAudio は MAX98357 につながる I2S0 を初期化して返す。
// 16bit ステレオ、16kHz。無音の状態で送信が始まる。
func NewAudio() (*i2s.Device, error) {
	dev := i2s.New()
	err := dev.Configure(i2s.Config{
		BCLK:       I2S_BCLK,
		LRC:        I2S_LRC,
		DIN:        I2S_DIN,
		SampleRate: AudioSampleRate,
		SlotBits:   AudioSlotBits,
	})
	if err != nil {
		return nil, err
	}
	return dev, nil
}

// sineTable は 1 周期を 256 分割した正弦波 (振幅 32767)。
var sineTable [256]int16

func init() {
	// math パッケージを避けて、回転行列で 1 周期分を生成する
	const (
		cosStep = 0.99969881869620424996 // cos(2π/256)
		sinStep = 0.02454122852291228803 // sin(2π/256)
	)
	x, y := 1.0, 0.0
	for i := range sineTable {
		sineTable[i] = int16(y * 32767)
		x, y = x*cosStep-y*sinStep, x*sinStep+y*cosStep
	}
}

// ToneGenerator は正弦波を生成して I2S に流す。単音のほか、複数の周波数を
// 重ねた和音も鳴らせる (MaxVoices まで)。
// 振幅の急な変化はクリック音になるので、音の始まりと終わり、音量の変化点では
// RampMs の長さで直線的にフェードする。
type ToneGenerator struct {
	dev    *i2s.Device
	buf    []int16
	phases [MaxVoices]uint32 // 32bit の位相 (上位 8bit がテーブルの添字)
	freqs  [MaxVoices]uint32 // 最後に鳴らした周波数 (フェードアウト用)
	voices int
	amp    int32 // 現在の振幅 (0..255)
}

// MaxVoices は同時に重ねられる正弦波の数。
const MaxVoices = 4

// NewToneGenerator は dev に出力するトーンジェネレータを返す。
func NewToneGenerator(dev *i2s.Device) *ToneGenerator {
	return &ToneGenerator{dev: dev, buf: make([]int16, dev.BufferFrames()*2)}
}

// RampMs はフェードイン・フェードアウトにかける時間 (ms)。
const RampMs = 5

// MaxVolume は volume の上限。これを超える指定はここに丸める。
// MAX98357 (GAIN=12dB、5V) では最大振幅付近でクリップして音割れするため、
// 余裕を持たせている。
const MaxVolume = 128

func (t *ToneGenerator) rampFrames() int {
	return int(t.dev.SampleRate()) * RampMs / 1000
}

func (t *ToneGenerator) frames(ms int) int {
	return int(uint64(t.dev.SampleRate()) * uint64(ms) / 1000)
}

// setVoices は鳴らす周波数を設定する。周波数が変わった声部は位相を 0 に戻す。
func (t *ToneGenerator) setVoices(freqs []uint32) {
	n := len(freqs)
	if n > MaxVoices {
		n = MaxVoices
	}
	for i := 0; i < n; i++ {
		if t.freqs[i] != freqs[i] {
			t.freqs[i] = freqs[i]
			t.phases[i] = 0
		}
	}
	t.voices = n
}

func clampVolume(volume uint8) int32 {
	if volume > MaxVolume {
		return MaxVolume
	}
	return int32(volume)
}

// gen は設定済みの声部を frames フレーム書く。振幅は from から to へ直線的に変える。
// 複数の声部は足し合わせて声部数で割るので、和音でも振幅は volume を超えない。
func (t *ToneGenerator) gen(frames int, from, to int32) error {
	if frames <= 0 {
		t.amp = to
		return nil
	}
	rate := t.dev.SampleRate()
	var steps [MaxVoices]uint32
	for i := 0; i < t.voices; i++ {
		steps[i] = uint32(uint64(t.freqs[i]) << 32 / uint64(rate))
	}
	silent := t.voices == 0 || (from == 0 && to == 0)
	for pos := 0; pos < frames; {
		n := len(t.buf) / 2
		if n > frames-pos {
			n = frames - pos
		}
		for i := 0; i < n; i++ {
			var v int16
			if !silent {
				amp := from + (to-from)*int32(pos+i)/int32(frames)
				var sum int32
				for k := 0; k < t.voices; k++ {
					sum += int32(sineTable[t.phases[k]>>24])
					t.phases[k] += steps[k]
				}
				v = int16(sum / int32(t.voices) * amp / 256)
			}
			t.buf[2*i] = v
			t.buf[2*i+1] = v
		}
		if _, err := t.dev.Write(t.buf[:2*n]); err != nil {
			return err
		}
		pos += n
	}
	t.amp = to
	return nil
}

// Play は freq (Hz) の正弦波を ms ミリ秒鳴らす。volume は 0..255。
// freq が 0 なら無音を ms ミリ秒流す。前後に RampMs のフェードが入り、
// 鳴り終わりは無音になる。再生が終わるまでブロックする。
// DMA が進まない場合は i2s.ErrTimeout を返す。
func (t *ToneGenerator) Play(freq uint32, ms int, volume uint8) error {
	return t.PlayChord([]uint32{freq}, ms, volume)
}

// PlayChord は複数の周波数を重ねた和音を ms ミリ秒鳴らす。
// それ以外は Play と同じ。
func (t *ToneGenerator) PlayChord(freqs []uint32, ms int, volume uint8) error {
	if len(freqs) == 0 || volume == 0 || allZero(freqs) {
		return t.Rest(ms)
	}
	total := t.frames(ms)
	ramp := t.rampFrames()
	if ramp*2 > total {
		ramp = total / 2
	}
	t.setVoices(freqs)
	vol := clampVolume(volume)
	if err := t.gen(ramp, t.amp, vol); err != nil {
		return err
	}
	if err := t.gen(total-2*ramp, vol, vol); err != nil {
		return err
	}
	return t.gen(ramp, vol, 0)
}

// Rest は ms ミリ秒の無音を流す。鳴っている途中ならフェードアウトしてから無音にする。
func (t *ToneGenerator) Rest(ms int) error {
	total := t.frames(ms)
	if t.amp != 0 {
		ramp := t.rampFrames()
		if ramp > total {
			ramp = total
		}
		if err := t.gen(ramp, t.amp, 0); err != nil {
			return err
		}
		total -= ramp
	}
	return t.gen(total, 0, 0)
}

// Hold は freq の正弦波を ms ミリ秒分書く。連続して呼ぶと途切れずに鳴り続け、
// 周波数や音量が変わった直後だけ RampMs でフェードする。
// 鳴り終えるときは Stop を呼ぶこと。
func (t *ToneGenerator) Hold(freq uint32, ms int, volume uint8) error {
	return t.HoldChord([]uint32{freq}, ms, volume)
}

// HoldChord は複数の周波数を重ねた和音を Hold と同じ要領で鳴らし続ける。
func (t *ToneGenerator) HoldChord(freqs []uint32, ms int, volume uint8) error {
	total := t.frames(ms)
	vol := clampVolume(volume)
	if len(freqs) == 0 || allZero(freqs) {
		vol = 0
	} else {
		t.setVoices(freqs)
	}
	if t.amp != vol {
		ramp := t.rampFrames()
		if ramp > total {
			ramp = total
		}
		if err := t.gen(ramp, t.amp, vol); err != nil {
			return err
		}
		total -= ramp
	}
	return t.gen(total, vol, vol)
}

// Stop は鳴っている音をフェードアウトしてから無音にする。
func (t *ToneGenerator) Stop() {
	if t.amp != 0 {
		t.gen(t.rampFrames(), t.amp, 0)
	}
	t.voices = 0
	t.dev.Silence()
}

func allZero(freqs []uint32) bool {
	for _, f := range freqs {
		if f != 0 {
			return false
		}
	}
	return true
}
