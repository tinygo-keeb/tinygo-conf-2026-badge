//go:build xtensa

package ws2812s3

import (
	"device"
	"runtime/interrupt"
	"unsafe"
)

// writeByte240 は CPU クロック 240MHz 向けに 1 バイト送信する。
//
// tinygo.org/x/drivers/ws2812 の xtensa 160MHz 実装をもとに、
// nop の数を 1.5 倍にしたもの。全命令 1 サイクルと仮定している。
// 実際にはストアや分岐で数サイクル余計にかかるが、WS2812 は
// 各パルスが多少長くなる方向には寛容なので問題ない。
//
//	T0H: 84 cycles = 350ns  (規格 350ns ±150)
//	T0L: 217 cycles = 904ns  (規格 900ns ±150)
//	T1H: 192 cycles = 800ns  (規格 800ns ±150)
//	T1L: 108 cycles = 450ns  (規格 450ns ±150)
func (d Device) writeByte240(c byte) {
	portSet, maskSet := d.Pin.PortMaskSet()
	portClear, maskClear := d.Pin.PortMaskClear()
	bitbang240(c, portSet, maskSet, portClear, maskClear)
}

// warmup240 は bitbang240 の命令列を、ピンに影響しないマスク 0 で 1 バイト分
// 実行して命令キャッシュに載せる。
//
// このコードはフラッシュ上にありキャッシュ経由で実行される。連続送信中は
// キャッシュに残っているが、BLE や Wi-Fi のスタックが動く合間に一度だけ送る
// ような使い方では送信開始時にキャッシュミスが起きて先頭のビットの High が
// 数 us に伸び、先頭の LED が先頭ビットを 1 と読んでしまう (先頭は G なので
// 緑に固定される)。GPIO の W1TS/W1TC レジスタにマスク 0 を書いても何も
// 起きないので、これで安全に温められる。
func (d Device) warmup240() {
	portSet, _ := d.Pin.PortMaskSet()
	portClear, _ := d.Pin.PortMaskClear()
	bitbang240(0, portSet, 0, portClear, 0)
}

// bitbang240 は 1 バイトを WS2812 のタイミングで送る (CPU 240MHz 用)。
// warmup240 と writeByte240 が同じコードを共有するように、インライン展開させない
// (展開されると温めたコピーと本番のコピーが別物になる)。
//
//go:noinline
func bitbang240(c byte, portSet *uint32, maskSet uint32, portClear *uint32, maskClear uint32) {
	mask := interrupt.Disable()
	device.AsmFull(`
		1: // send_bit
			s32i  {maskSet}, {portSet}, 0     // [1]  T0H and T1H start here
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop                                          // [80]
			slli  {value}, {value}, 1         // [1]  shift {value} to the left by 1
			bbsi  {value}, 8, 2f              // [1]  branch to skip_store if bit 8 is set
			s32i  {maskClear}, {portClear}, 0 // [1]  T0H -> T0L transition
		2: // skip_store
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop                                          // [108]
			s32i  {maskClear}, {portClear}, 0 // [1]  T1H -> T1L transition
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop
			nop                                          // [106]
			addi  {i}, {i}, -1                // [1]
			bnez  {i}, 1b                      // [1]  send_bit, T1H and T1L end here

			// Restore original values after modifying them in the inline
			// assembly. Not doing that would result in undefined behavior as
			// the compiler doesn't know we're modifying these values.
			movi.n {i}, 8
			slli  {value}, {value}, 8
	`, map[string]interface{}{
		"value":     uint32(c),
		"i":         8,
		"maskSet":   maskSet,
		"portSet":   uintptr(unsafe.Pointer(portSet)),
		"maskClear": maskClear,
		"portClear": uintptr(unsafe.Pointer(portClear)),
	})
	interrupt.Restore(mask)
}
