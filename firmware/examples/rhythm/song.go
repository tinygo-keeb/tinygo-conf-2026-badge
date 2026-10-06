package main

import "time"

// 曲は 8 分音符 (stepMs) 単位のステップの並び。テンポ 120 BPM。
const stepMs = 250

const stepsPerBar = 8

// レーン。画面の左から順に並ぶ。ジョイスティックの上と下は同じレーン。
const (
	laneLeft   = iota // ジョイスティック左
	laneUpDown        // ジョイスティック上または下
	laneRight         // ジョイスティック右
	laneSW2
	laneSW1
	laneCount
)

// メロディは 5 音 (ペンタトニック) で、c d e g a をそれぞれレーン 0..4 に割り当てる。
// 音が高いほど右のレーンになるので、鍵盤のように音の高さと位置が対応する。
const noteNames = "cdega"

// melodyFreq はレーンごとのメロディの周波数 (C5 D5 E5 G5 A5)。
var melodyFreq = [laneCount]uint32{523, 587, 659, 784, 880}

// ベースの根音 (C3..A3)。F はメロディには無いがベースでは使う。
const rootNames = "cdefga"

var rootFreq = [...]uint32{131, 147, 165, 175, 196, 220}

// bar は 1 小節分の譜面。melody は 1 文字が 8 分音符 1 個で、
// c d e g a が音 (= ノーツ)、'.' が休符。root はその小節のベースの根音。
type bar struct {
	melody string
	root   byte
}

var (
	phraseA = []bar{
		{"c.e.g.e.", 'c'},
		{"a.a.g...", 'f'},
		{"e.d.c.d.", 'c'},
		{"e...d...", 'g'},
	}
	phraseB = []bar{
		{"a.g.e.d.", 'f'},
		{"d.e.g.a.", 'g'},
		{"a.g.e.c.", 'a'},
		{"d...c...", 'c'},
	}
	phraseC = []bar{
		{"cdega.g.", 'c'},
		{"gaga.e..", 'f'},
		{"edcde.g.", 'c'},
		{"a.g.d...", 'g'},
	}
	phraseD = []bar{
		{"e.e.g.g.", 'c'},
		{"a.a.g...", 'f'},
		{"g.g.e.e.", 'g'},
		{"d.d.c...", 'c'},
	}
)

// song は曲全体。前奏 (ドラムのみ) のあとにフレーズを並べる。
var song = concat(phraseA, phraseB, phraseA, phraseB, phraseC, phraseD, phraseC, phraseD)

// introBars は前奏の小節数。ここでノーツが降り始めるまでの準備をする。
const introBars = 2

// outroSteps は曲の最後のノーツのあと、結果画面に移るまでのステップ数。
const outroSteps = 8

func concat(phrases ...[]bar) []bar {
	var out []bar
	for _, p := range phrases {
		out = append(out, p...)
	}
	return out
}

// step はシンセが 1 ステップの頭で鳴らす音。
type step struct {
	melody uint32 // メロディの周波数 (0 なら鳴らさない)
	bass   uint32 // ベースの周波数 (0 なら鳴らさない)
	kick   bool
	hat    bool
}

// note は降ってくるノーツ 1 個。
type note struct {
	t     time.Duration // 判定ラインに来る時刻 (曲の先頭から)
	lane  int
	state uint8
}

const (
	notePending = iota
	noteHit
	noteMissed
)

// difficulty は難易度。EASY は小節の 1 拍目と 3 拍目のノーツだけを残す。
type difficulty int

const (
	easy difficulty = iota
	normal
)

// buildSong は曲 (シンセ用のステップ列) と、難易度に応じた譜面を作る。
// メロディはどちらの難易度でも全部鳴らす。
func buildSong(d difficulty) ([]step, []note) {
	total := (introBars+len(song))*stepsPerBar + outroSteps
	steps := make([]step, total)
	var notes []note
	for i := range steps {
		st := &steps[i]
		pos := i % stepsPerBar
		bi := i/stepsPerBar - introBars
		st.kick = pos%4 == 0 && bi < len(song)
		st.hat = pos%2 == 1 && bi < len(song)
		if bi < 0 || bi >= len(song) {
			continue
		}
		b := song[bi]
		if pos%2 == 0 {
			st.bass = rootFreq[indexOf(rootNames, b.root)]
		}
		lane := indexOf(noteNames, b.melody[pos])
		if lane < 0 {
			continue
		}
		st.melody = melodyFreq[lane]
		if d == easy && pos%4 != 0 {
			continue
		}
		notes = append(notes, note{
			t:    time.Duration(i) * stepMs * time.Millisecond,
			lane: lane,
		})
	}
	return steps, notes
}

// indexOf は names の中の c の位置を返す。無ければ -1。
func indexOf(names string, c byte) int {
	for i := 0; i < len(names); i++ {
		if names[i] == c {
			return i
		}
	}
	return -1
}

// songLength は曲の長さ (最後のステップの終わり)。
func songLength(steps []step) time.Duration {
	return time.Duration(len(steps)) * stepMs * time.Millisecond
}
