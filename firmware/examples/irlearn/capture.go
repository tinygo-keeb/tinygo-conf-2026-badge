package main

import (
	"machine"
	"runtime/volatile"
	"time"

	"github.com/sago35/tinygo-conf-2026-badge/firmware/badge"
)

// 赤外線信号の生パルス記録。
//
// 受信モジュールの出力 (アイドル High、受光中 Low) の両エッジで割り込みを受け、
// 前のエッジからの経過時間 (マイクロ秒) を順に記録する。先頭は Low の期間
// (mark) から始まり、以降 space, mark, ... と交互に並ぶ。
// 最後のエッジから FrameGap 以上経ったらフレーム終了とみなす (メインループで判定)。

const (
	// MaxEdges は 1 フレームに記録できるパルス数の上限。
	// NEC は 67、AEHA (家電協) は 100 前後、長いものでも 200 あれば足りる。
	MaxEdges = 200
	// FrameGap はこれ以上信号が来なければフレーム終了とみなす時間。
	FrameGap = 30 * time.Millisecond
	// MinEdges はこれより短い記録はノイズとして捨てる。
	MinEdges = 10
)

// capture は割り込みから書かれるので、カウンタは volatile にする。
var (
	capBuf   [MaxEdges]uint16
	capCount volatile.Register32
	capLast  time.Time
	capArmed volatile.Register8 // 1 なら記録中
)

func onIREdge(_ machine.Pin) {
	if capArmed.Get() == 0 {
		return
	}
	now := time.Now()
	n := capCount.Get()
	if n == 0 {
		// 最初のエッジ (立ち下がり) はフレームの開始。長さは次のエッジで決まる
		capLast = now
		capCount.Set(1)
		return
	}
	d := now.Sub(capLast)
	capLast = now
	if n-1 >= MaxEdges {
		return
	}
	us := d.Microseconds()
	if us > 65535 {
		us = 65535
	}
	capBuf[n-1] = uint16(us)
	capCount.Set(n + 1)
}

// startCapture は記録を開始する (受信モジュールの割り込みを有効にする)。
func startCapture() {
	capCount.Set(0)
	capArmed.Set(1)
	badge.IR_DATA.SetInterrupt(0, nil)
	badge.IR_DATA.SetInterrupt(machine.PinFalling|machine.PinRising, onIREdge)
}

// stopCapture は記録を止める。
func stopCapture() {
	capArmed.Set(0)
	badge.IR_DATA.SetInterrupt(0, nil)
}

// captured は 1 フレーム分の記録が完了していればそのパルス列を返す。
// 完了していなければ nil。取り出したあとは次の記録に備えてクリアする。
func captured() []uint16 {
	n := capCount.Get()
	if n == 0 || time.Since(capLast) < FrameGap {
		return nil
	}
	// n はエッジ数、パルス数は n-1
	pulses := int(n) - 1
	if pulses > MaxEdges {
		pulses = MaxEdges
	}
	capCount.Set(0)
	if pulses < MinEdges {
		return nil
	}
	out := make([]uint16, pulses)
	copy(out, capBuf[:pulses])
	return out
}
