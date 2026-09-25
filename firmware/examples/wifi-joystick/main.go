// Wi-Fi に接続して HTTP サーバーを立ち上げ、ジョイスティックの XY 位置と
// スイッチ (SW1, SW2, ジョイスティックのボタン) の状態をブラウザにリアルタイムで表示する。
//
// 起動直後 (Wi-Fi 接続前) にジョイスティックのセンター位置を実測するので、
// 起動時はスティックに触らないこと。
//
//	/       XY 平面にプロットする HTML ページ (約 32ms 間隔で /input を取得)
//	/input  {"x":-12,"y":345,"joy":false,"sw1":true,"sw2":false,"dead":true,"sat":true}
//	        x, y は -1000..1000 (中立 0)。dead, sat はデッドゾーンと飽和の有効/無効
//	/config?dead=1&sat=0  デッドゾーンと飽和の有効/無効を切り替える (ページのチェックボックス)
//
// SSID とパスワードはビルド時に ldflags で埋め込む。
//
//	make flash-wifi-joystick SSID=yourssid PASS=yourpassword
package main

import (
	"strconv"
	"sync"
	"time"

	"github.com/soypat/lneto/http/httphi"
	"github.com/soypat/lneto/http/httpraw"
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

// デッドゾーンと飽和の既定値 (badge.NewJoystick の値)。オフにするときは 0 と 1000 にする。
const (
	defaultDeadZone   = 80
	defaultSaturation = 850
)

// 入力の最新値。読み取りは 1 つの goroutine だけが行い、ハンドラは mu を取って読む。
// joy のフィールド変更も mu の中で行う。
var (
	mu           sync.Mutex
	joy          *badge.Joystick
	joyX, joyY   int
	joyBtn       bool
	sw1On, sw2On bool
)

const indexHTML = `<!doctype html>
<html lang="ja"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Joystick</title>
<style>
body{font-family:sans-serif;margin:1em;background:#111;color:#eee}
canvas{background:#222;border:1px solid #555;touch-action:none}
.sw{display:inline-block;margin:.5em 1em .5em 0;padding:.4em 1em;border-radius:.4em;background:#333;color:#888}
.sw.on{background:#2a2;color:#fff}
#s{color:#888;font-size:.9em}
</style></head><body>
<h1>Joystick</h1>
<canvas id="c" width="320" height="320"></canvas>
<div>
<span id="sw1" class="sw">SW1</span><span id="sw2" class="sw">SW2</span><span id="joy" class="sw">JOY</span>
</div>
<p>
<label><input type="checkbox" id="dead" checked> dead zone</label>
<label><input type="checkbox" id="sat" checked> saturation</label>
</p>
<p id="v"></p><p id="s"></p>
<script>
const c=document.getElementById('c'),g=c.getContext('2d'),W=c.width,H=c.height;
const trail=[];const MAXTRAIL=60;
function draw(x,y){
  g.clearRect(0,0,W,H);
  g.strokeStyle='#444';g.lineWidth=1;
  g.beginPath();g.moveTo(W/2,0);g.lineTo(W/2,H);g.moveTo(0,H/2);g.lineTo(W,H/2);g.stroke();
  g.beginPath();g.arc(W/2,H/2,W/2-2,0,Math.PI*2);g.stroke();
  const px=v=>W/2+v*(W/2-8)/1000, py=v=>H/2-v*(H/2-8)/1000;
  for(let i=0;i<trail.length;i++){
    const a=(i+1)/trail.length;g.fillStyle='rgba(80,160,255,'+(a*0.6)+')';
    g.beginPath();g.arc(px(trail[i][0]),py(trail[i][1]),3,0,Math.PI*2);g.fill();
  }
  g.fillStyle='#4af';g.beginPath();g.arc(px(x),py(y),8,0,Math.PI*2);g.fill();
}
function setSw(id,on){document.getElementById(id).className='sw'+(on?' on':'');}
const dead=document.getElementById('dead'),sat=document.getElementById('sat');
let synced=false; // 最初の /input 応答で基板側の設定をチェックボックスに反映する
async function sendConfig(){
  try{await fetch('/config?dead='+(dead.checked?1:0)+'&sat='+(sat.checked?1:0));}
  catch(e){document.getElementById('s').textContent='config error: '+e;}
}
dead.onchange=sendConfig;sat.onchange=sendConfig;
const INTERVAL=32; // ms。応答を待ってから次を出すので、遅延時に要求が溜まらない
let fails=0;
async function update(){
  const t0=performance.now();
  try{
    const r=await fetch('/input');const j=await r.json();
    if(!synced){dead.checked=j.dead;sat.checked=j.sat;synced=true;}
    trail.push([j.x,j.y]);if(trail.length>MAXTRAIL)trail.shift();
    draw(j.x,j.y);
    setSw('sw1',j.sw1);setSw('sw2',j.sw2);setSw('joy',j.joy);
    document.getElementById('v').textContent='x='+j.x+' y='+j.y+'  ('+Math.round(performance.now()-t0)+'ms)';
    document.getElementById('s').textContent='';fails=0;
  }catch(e){fails++;document.getElementById('s').textContent='error ('+fails+'): '+e;}
  setTimeout(update,Math.max(0,INTERVAL-(performance.now()-t0)));
}
draw(0,0);update();
</script></body></html>`

func main() {
	time.Sleep(2 * time.Second)
	if ssid == "" {
		failure("ssid is empty: build with -ldflags=\"-X main.ssid=... -X main.password=...\"")
	}

	sw1, sw2 := badge.NewButtons()
	joy = badge.NewJoystick()
	println("calibrating joystick center (do not touch)...")
	joy.Calibrate(500 * time.Millisecond)
	println("center:", joy.CenterX, joy.CenterY)
	go func() {
		for {
			mu.Lock()
			x, y, btn := joy.Read()
			p1, p2 := sw1.Pressed(), sw2.Pressed()
			joyX, joyY, joyBtn = x, y, btn
			sw1On, sw2On = p1, p2
			mu.Unlock()
			time.Sleep(10 * time.Millisecond)
		}
	}()

	// 接続に失敗した場合はリセットして最初からやり直す (成功するまで戻らない)
	link := &link.Esplink{}
	netdev.UseNetdev(link)
	wifi.Connect(link, ssid, password)

	var mux httphi.MuxSlice
	mux.Handle("/", index)
	mux.Handle("/input", input)
	mux.Handle("/config", config)

	var router httphi.Router
	cfg := httphi.DefaultRouterConfig(4, 2048, mux.MaxPathValues())
	failIfErr("router configure", router.Configure(&mux, cfg))
	defer router.Shutdown()

	addr, err := link.Addr()
	failIfErr("link.Addr", err)
	print("HTTP server listening on http://", addr.String(), ":", port, "\n")
	failIfErr("ListenAndServe", link.ListenAndServe(&router, port))
}

func index(exch *httphi.Exchange) {
	exch.RespondString(httphi.StatusOK, "text/html; charset=utf-8", indexHTML)
}

// レスポンス組み立て用の固定バッファ。ハンドラごとに確保しない。
var (
	bufMu sync.Mutex
	buf   [96]byte
)

func input(exch *httphi.Exchange) {
	mu.Lock()
	x, y, btn, p1, p2 := joyX, joyY, joyBtn, sw1On, sw2On
	dead, sat := joy.DeadZone > 0, joy.Saturation < 1000
	mu.Unlock()

	bufMu.Lock()
	defer bufMu.Unlock()
	b := append(buf[:0], `{"x":`...)
	b = strconv.AppendInt(b, int64(x), 10)
	b = append(b, `,"y":`...)
	b = strconv.AppendInt(b, int64(y), 10)
	b = append(b, `,"joy":`...)
	b = strconv.AppendBool(b, btn)
	b = append(b, `,"sw1":`...)
	b = strconv.AppendBool(b, p1)
	b = append(b, `,"sw2":`...)
	b = strconv.AppendBool(b, p2)
	b = append(b, `,"dead":`...)
	b = strconv.AppendBool(b, dead)
	b = append(b, `,"sat":`...)
	b = strconv.AppendBool(b, sat)
	b = append(b, '}')
	exch.Respond(httphi.StatusOK, "application/json", b)
}

var formBuf [64]byte

// config は /config?dead=1&sat=0 を受けてデッドゾーンと飽和の有効/無効を切り替える。
func config(exch *httphi.Exchange) {
	bufMu.Lock()
	defer bufMu.Unlock()
	var form httpraw.Form
	form.Reset(formBuf[:], 2)
	if err := exch.RequestParseForm(&form, true, true); err != nil {
		exch.RespondString(httphi.StatusBadRequest, "text/plain", "bad query\n")
		return
	}
	mu.Lock()
	if v := form.Get("dead"); len(v) > 0 {
		if v[0] == '1' {
			joy.DeadZone = defaultDeadZone
		} else {
			joy.DeadZone = 0
		}
	}
	if v := form.Get("sat"); len(v) > 0 {
		if v[0] == '1' {
			joy.Saturation = defaultSaturation
		} else {
			joy.Saturation = 1000
		}
	}
	dead, sat := joy.DeadZone, joy.Saturation
	mu.Unlock()
	println("config: deadzone", dead, "saturation", sat)

	b := append(buf[:0], `{"deadzone":`...)
	b = strconv.AppendInt(b, int64(dead), 10)
	b = append(b, `,"saturation":`...)
	b = strconv.AppendInt(b, int64(sat), 10)
	b = append(b, '}')
	exch.Respond(httphi.StatusOK, "application/json", b)
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
