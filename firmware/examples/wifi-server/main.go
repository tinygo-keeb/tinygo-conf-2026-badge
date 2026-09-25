// Wi-Fi に接続して HTTP サーバーを立ち上げ、基板上の AHT21B の温湿度を返す。
//
//	/         温湿度を表示する HTML ページ (2 秒ごとに /api を取得して更新)
//	/api      {"temperature":25.3,"humidity":48.2} 形式の JSON
//	/metrics  Prometheus 形式のテキスト
//
// SSID とパスワードはビルド時に ldflags で埋め込む。
//
//	make flash-wifi-server SSID=yourssid PASS=yourpassword
//
// HTTP サーバーには espradio 推奨の httphi (リクエストごとにヒープを使わない) を使う。
package main

import (
	"strconv"
	"sync"
	"time"

	"github.com/soypat/lneto/http/httphi"
	"tinygo.org/x/drivers/netdev"
	link "tinygo.org/x/espradio/netlink"

	"github.com/sago35/tinygo-conf-2026-badge/firmware/badge"
	"github.com/sago35/tinygo-conf-2026-badge/firmware/wifi"
)

var (
	ssid     string
	password string
)

const port uint16 = 80

// センサーの最新値。読み取りは 1 つの goroutine だけが行い、
// HTTP ハンドラは mu を取ってコピーを読む。
var (
	mu       sync.Mutex
	deciTemp int32 // 0.1 度単位
	deciHum  int32 // 0.1% 単位
	readErr  error
	readAt   time.Time
)

const indexHTML = `<!doctype html>
<html lang="ja"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>TinyGo Conf 2026 badge</title>
<style>
body{font-family:sans-serif;margin:2em;background:#111;color:#eee}
.v{font-size:3em;font-weight:bold}
.u{font-size:1em;color:#999}
</style></head><body>
<h1>TinyGo Conf 2026 badge</h1>
<p>AHT21B</p>
<div><span id="t" class="v">--</span><span class="u"> &deg;C</span></div>
<div><span id="h" class="v">--</span><span class="u"> %RH</span></div>
<p id="s" class="u"></p>
<script>
async function update(){
  try{
    const r=await fetch('/api');const j=await r.json();
    document.getElementById('t').textContent=j.temperature.toFixed(1);
    document.getElementById('h').textContent=j.humidity.toFixed(1);
    document.getElementById('s').textContent='updated '+new Date().toLocaleTimeString();
  }catch(e){document.getElementById('s').textContent='error: '+e;}
}
update();setInterval(update,2000);
</script></body></html>`

func main() {
	time.Sleep(2 * time.Second)
	if ssid == "" {
		failure("ssid is empty: build with -ldflags=\"-X main.ssid=... -X main.password=...\"")
	}

	sensor := badge.NewSensor()
	go func() {
		for {
			err := sensor.Read()
			mu.Lock()
			readErr = err
			if err == nil {
				deciTemp = sensor.DeciCelsius()
				deciHum = sensor.DeciRelHumidity()
				readAt = time.Now()
			}
			mu.Unlock()
			time.Sleep(2 * time.Second)
		}
	}()

	// 接続に失敗した場合はリセットして最初からやり直す (成功するまで戻らない)
	link := &link.Esplink{}
	netdev.UseNetdev(link)
	wifi.Connect(link, ssid, password)

	var mux httphi.MuxSlice
	mux.Handle("/", logRequest(index))
	mux.Handle("/api", logRequest(api))
	mux.Handle("/metrics", logRequest(metrics))

	var router httphi.Router
	cfg := httphi.DefaultRouterConfig(4, 2048, mux.MaxPathValues())
	failIfErr("router configure", router.Configure(&mux, cfg))
	defer router.Shutdown()

	addr, err := link.Addr()
	failIfErr("link.Addr", err)
	print("HTTP server listening on http://", addr.String(), ":", port, "\n")
	failIfErr("ListenAndServe", link.ListenAndServe(&router, port))
}

func logRequest(h httphi.HandlerFunc) httphi.HandlerFunc {
	return func(exch *httphi.Exchange) {
		println(exch.RequestMethod().String(), exch.MuxPattern())
		h(exch)
	}
}

func index(exch *httphi.Exchange) {
	exch.RespondString(httphi.StatusOK, "text/html; charset=utf-8", indexHTML)
}

// レスポンス組み立て用の固定バッファ。ハンドラごとに確保しない。
var (
	bufMu sync.Mutex
	buf   [160]byte
)

func snapshot() (t, h int32, err error, age time.Duration) {
	mu.Lock()
	defer mu.Unlock()
	return deciTemp, deciHum, readErr, time.Since(readAt)
}

func api(exch *httphi.Exchange) {
	t, h, err, _ := snapshot()
	bufMu.Lock()
	defer bufMu.Unlock()
	if err != nil {
		exch.RespondString(httphi.StatusInternalServerError, "application/json", `{"error":"sensor read failed"}`)
		return
	}
	b := append(buf[:0], `{"temperature":`...)
	b = appendDeci(b, t)
	b = append(b, `,"humidity":`...)
	b = appendDeci(b, h)
	b = append(b, '}')
	exch.Respond(httphi.StatusOK, "application/json", b)
}

func metrics(exch *httphi.Exchange) {
	t, h, err, _ := snapshot()
	bufMu.Lock()
	defer bufMu.Unlock()
	if err != nil {
		exch.RespondString(httphi.StatusInternalServerError, "text/plain", "sensor read failed\n")
		return
	}
	b := append(buf[:0], "badge_temperature_celsius "...)
	b = appendDeci(b, t)
	b = append(b, "\nbadge_humidity_percent "...)
	b = appendDeci(b, h)
	b = append(b, '\n')
	exch.Respond(httphi.StatusOK, "text/plain; version=0.0.4", b)
}

// appendDeci は 0.1 単位の整数を "12.3" 形式で追記する。
func appendDeci(b []byte, v int32) []byte {
	if v < 0 {
		b = append(b, '-')
		v = -v
	}
	b = strconv.AppendInt(b, int64(v/10), 10)
	b = append(b, '.')
	return strconv.AppendInt(b, int64(v%10), 10)
}

func failIfErr(action string, err error) {
	for err != nil {
		println("fail", action+":", err.Error())
		time.Sleep(time.Second)
	}
}

func failure(msg string) {
	for {
		println("failure:", msg)
		time.Sleep(time.Second)
	}
}
