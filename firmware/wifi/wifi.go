// Package wifi は Wi-Fi 接続のヘルパー。
//
// 意図的に tinygo.org/x/espradio を import していない。TinyGo はパッケージの
// init を import パスの辞書順に実行し、コンパイル時に評価できない init
// (espradio の C 呼び出し) に当たると、それ以降のパッケージの init を実行時に
// 回す。このモジュール (github.com/sago35/...) は lneto や net/http より辞書順で
// 前に並ぶため、ここから espradio を import すると unicode などの初期化が
// 実行時に回り、RAM が約 70KB 増える。espradio は example の main から直接
// import し、ここには netlink.Netlinker として渡す。
package wifi

import (
	"time"

	nl "tinygo.org/x/drivers/netlink"

	"github.com/sago35/tinygo-conf-2026-badge/firmware/badge"
)

// RetryDelay は接続に失敗してからリセットするまでの待ち時間。
const RetryDelay = 5 * time.Second

// Connect は l で Wi-Fi に接続する。
// 接続に失敗した場合は RetryDelay 待ってからチップをリセットして最初からやり直す
// (espradio は無線の初期化を一度しかできず、失敗後に NetConnect を呼び直しても
// "already enabled" で失敗するため)。成功するまで戻らない。
//
// モニタの切断などでチップがリセットされた直後は、アクセスポイント側に前の
// セッションが残っていて "auth expired" になることがあるが、リセット後の再試行で
// つながる。
func Connect(l nl.Netlinker, ssid, password string) {
	println("connecting to WiFi:", ssid)
	err := l.NetConnect(&nl.ConnectParams{Ssid: ssid, Passphrase: password})
	if err != nil {
		println("WiFi connect failed:", err.Error(), "- resetting in", int(RetryDelay/time.Second), "s")
		time.Sleep(RetryDelay)
		badge.Reset()
	}
	println("WiFi connected")
}
