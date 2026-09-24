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
