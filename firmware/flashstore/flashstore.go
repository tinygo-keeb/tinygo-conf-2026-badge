// Package flashstore は ESP32-S3 の SPI フラッシュの一部を設定保存用に読み書きする。
//
// TinyGo の machine パッケージには ESP32-S3 用のフラッシュ API が無い (ESP32-C3 用は
// ある) ため、ROM に固定アドレスで存在する esp_rom_spiflash_* を CGo から直接呼ぶ。
// アドレスは ESP-IDF v5.1.2 の components/esp_rom/esp32s3/ld/esp32s3.rom.ld による。
// 手順は TinyGo の machine_esp32c3_flash.go と同じ (割り込み禁止、unlock、erase/write)。
//
// 保存領域はフラッシュ先頭から RegionOffset の位置に RegionSize だけ確保する。
package flashstore

/*
#include <stdint.h>

// ESP32-S3 ROM (esp32s3.rom.ld)
static int rom_spiflash_read(uint32_t addr, uint32_t *data, uint32_t len) {
	return ((int (*)(uint32_t, uint32_t *, uint32_t))0x40000a20)(addr, data, len);
}
static int rom_spiflash_write(uint32_t addr, const uint32_t *data, uint32_t len) {
	return ((int (*)(uint32_t, const uint32_t *, uint32_t))0x40000a14)(addr, data, len);
}
static int rom_spiflash_erase_sector(uint32_t sector) {
	return ((int (*)(uint32_t))0x400009fc)(sector);
}
static int rom_spiflash_unlock(void) {
	return ((int (*)(void))0x40000a2c)();
}
*/
import "C"

import (
	"errors"
	"runtime/interrupt"
	"unsafe"
)

// TinyGo のイメージヘッダはフラッシュサイズを 2MB と書くため、ROM のフラッシュ
// ドライバは 2MB を超えるアドレスの erase/write を拒否する。保存領域は 2MB 未満の
// 末尾に置く。TinyGo のアプリ本体は先頭側 (最大でも 1.2MB 程度) なので衝突しない。
const (
	// RegionOffset は保存領域のフラッシュ内オフセット (2MB - 64KB)。
	RegionOffset = 0x1F0000
	// RegionSize は保存領域の大きさ (4KB セクタ x 16)。
	RegionSize = 64 * 1024
	// SectorSize は消去単位。
	SectorSize = 4096
)

var (
	ErrOutOfRange = errors.New("flashstore: out of range")
	ErrAlign      = errors.New("flashstore: offset must be 4-byte aligned")
	ErrRead       = errors.New("flashstore: read failed")
	ErrWrite      = errors.New("flashstore: write failed")
	ErrErase      = errors.New("flashstore: erase failed")
	ErrUnlock     = errors.New("flashstore: unlock failed")
)

// Read は保存領域の off から len(p) バイト読む。
func Read(off uint32, p []byte) error {
	if off%4 != 0 {
		return ErrAlign
	}
	if int(off)+len(p) > RegionSize {
		return ErrOutOfRange
	}
	if len(p) == 0 {
		return nil
	}
	words := make([]uint32, (len(p)+3)/4)
	res := C.rom_spiflash_read(C.uint32_t(RegionOffset+off), (*C.uint32_t)(unsafe.Pointer(&words[0])), C.uint32_t(len(words)*4))
	if res != 0 {
		return ErrRead
	}
	copy(p, unsafe.Slice((*byte)(unsafe.Pointer(&words[0])), len(words)*4))
	return nil
}

// Erase は off から size バイト分を含むセクタを消去する (両端はセクタ境界に丸める)。
func Erase(off, size uint32) error {
	if off+size > RegionSize {
		return ErrOutOfRange
	}
	first := off / SectorSize
	last := (off + size + SectorSize - 1) / SectorSize
	state := interrupt.Disable()
	defer interrupt.Restore(state)
	if C.rom_spiflash_unlock() != 0 {
		return ErrUnlock
	}
	for s := first; s < last; s++ {
		if C.rom_spiflash_erase_sector(C.uint32_t((RegionOffset+s*SectorSize)/SectorSize)) != 0 {
			return ErrErase
		}
	}
	return nil
}

// Write は off に p を書く。事前に Erase しておくこと。長さは 4 バイト単位に
// 0xff で埋められる (フラッシュは 1 から 0 にしか変えられないため)。
func Write(off uint32, p []byte) error {
	if off%4 != 0 {
		return ErrAlign
	}
	if int(off)+len(p) > RegionSize {
		return ErrOutOfRange
	}
	if len(p) == 0 {
		return nil
	}
	words := make([]uint32, (len(p)+3)/4)
	buf := unsafe.Slice((*byte)(unsafe.Pointer(&words[0])), len(words)*4)
	for i := range buf {
		buf[i] = 0xff
	}
	copy(buf, p)
	state := interrupt.Disable()
	defer interrupt.Restore(state)
	if C.rom_spiflash_unlock() != 0 {
		return ErrUnlock
	}
	if C.rom_spiflash_write(C.uint32_t(RegionOffset+off), (*C.uint32_t)(unsafe.Pointer(&words[0])), C.uint32_t(len(buf))) != 0 {
		return ErrWrite
	}
	return nil
}
