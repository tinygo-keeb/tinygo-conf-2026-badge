// Wi-Fi に接続して http://httpbin.org/get を定期的に取得し、
// ステータスとヘッダ、ボディをシリアルに出力する。
//
// SSID とパスワードはビルド時に ldflags で埋め込む。
//
//	make flash-wifi-httpget SSID=yourssid PASS=yourpassword
//
// net/http は接続ごとにヒープを大きく使うため、長時間動かす用途には
// 向かない (espradio の examples/README.md 参照)。
package main

import (
	"io"
	"net/http"
	"strings"
	"time"

	"tinygo.org/x/drivers/netdev"
	link "tinygo.org/x/espradio/netlink"

	"github.com/sago35/tinygo-conf-2026-badge/firmware/wifi"
)

var (
	ssid     string
	password string
)

const url = "http://httpbin.org/get?device=tinygo-conf-2026-badge"

func main() {
	time.Sleep(2 * time.Second)
	if ssid == "" {
		failure("ssid is empty: build with -ldflags=\"-X main.ssid=... -X main.password=...\"")
	}

	// 接続に失敗した場合はリセットして最初からやり直す (成功するまで戻らない)
	link := &link.Esplink{}
	netdev.UseNetdev(link)
	wifi.Connect(link, ssid, password)
	if addr, err := link.Addr(); err == nil {
		println("IP:", addr.String())
	}

	for i := 1; ; i++ {
		println("GET", url)
		resp, err := http.Get(url)
		if err != nil {
			println("request failed:", err.Error())
			time.Sleep(10 * time.Second)
			continue
		}
		println(resp.Proto, resp.Status)
		for k, v := range resp.Header {
			println(k+":", strings.Join(v, " "))
		}
		println()
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			println("read body failed:", err.Error())
		} else {
			println(string(body))
		}
		println("-------- request", i, "done --------")
		time.Sleep(10 * time.Second)
	}
}

func failure(msg string) {
	for {
		println("failure:", msg)
		time.Sleep(time.Second)
	}
}
