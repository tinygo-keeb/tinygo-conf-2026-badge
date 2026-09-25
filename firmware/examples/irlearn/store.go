package main

import (
	"encoding/binary"

	"github.com/sago35/tinygo-conf-2026-badge/firmware/flashstore"
)

// 登録したリモコン信号の保存。flashstore の領域に全件をまとめて書く。
//
//	ヘッダ 8 バイト: magic "IRL1", count (uint16), 予約 (uint16)
//	エントリ: name (NameLen バイト、0 埋め)、pulses 数 (uint16)、予約 (uint16)、
//	          pulses (uint16 x MaxEdges、マイクロ秒)
// 各エントリは固定長 (entrySize) なので、index から位置が決まる。

const (
	NameLen    = 12
	MaxRemotes = 24
	headerSize = 8
	entrySize  = NameLen + 2 + 2 + MaxEdges*2
)

var magic = [4]byte{'I', 'R', 'L', '1'}

// Remote は登録済みのリモコン信号 1 件。
type Remote struct {
	Name   string
	Pulses []uint16
}

func loadRemotes() []Remote {
	var hdr [headerSize]byte
	if err := flashstore.Read(0, hdr[:]); err != nil {
		println("flash read error:", err.Error())
		return nil
	}
	if [4]byte{hdr[0], hdr[1], hdr[2], hdr[3]} != magic {
		return nil
	}
	count := int(binary.LittleEndian.Uint16(hdr[4:]))
	if count > MaxRemotes {
		count = MaxRemotes
	}
	remotes := make([]Remote, 0, count)
	var e [entrySize]byte
	for i := 0; i < count; i++ {
		if err := flashstore.Read(uint32(headerSize+i*entrySize), e[:]); err != nil {
			println("flash read error:", err.Error())
			break
		}
		n := int(binary.LittleEndian.Uint16(e[NameLen:]))
		if n > MaxEdges {
			n = MaxEdges
		}
		pulses := make([]uint16, n)
		for k := range pulses {
			pulses[k] = binary.LittleEndian.Uint16(e[NameLen+4+2*k:])
		}
		remotes = append(remotes, Remote{Name: trimName(e[:NameLen]), Pulses: pulses})
	}
	return remotes
}

func saveRemotes(remotes []Remote) error {
	if len(remotes) > MaxRemotes {
		remotes = remotes[:MaxRemotes]
	}
	size := headerSize + len(remotes)*entrySize
	if err := flashstore.Erase(0, uint32(size)); err != nil {
		return err
	}
	var hdr [headerSize]byte
	copy(hdr[:], magic[:])
	binary.LittleEndian.PutUint16(hdr[4:], uint16(len(remotes)))
	if err := flashstore.Write(0, hdr[:]); err != nil {
		return err
	}
	var e [entrySize]byte
	for i, r := range remotes {
		for k := range e {
			e[k] = 0
		}
		copy(e[:NameLen], r.Name)
		n := len(r.Pulses)
		if n > MaxEdges {
			n = MaxEdges
		}
		binary.LittleEndian.PutUint16(e[NameLen:], uint16(n))
		for k := 0; k < n; k++ {
			binary.LittleEndian.PutUint16(e[NameLen+4+2*k:], r.Pulses[k])
		}
		if err := flashstore.Write(uint32(headerSize+i*entrySize), e[:]); err != nil {
			return err
		}
	}
	return nil
}

func trimName(b []byte) string {
	n := 0
	for n < len(b) && b[n] != 0 {
		n++
	}
	return string(b[:n])
}
