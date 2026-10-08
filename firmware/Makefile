# 全 example をビルドする (コンパイル確認用)。生成物は out/ に置く。
#
# 書き込みたいときは該当する行をコピーして、`build -o ./out/xxx.bin` を `flash` に
# 変えてシェルで実行する。例:
#   tinygo flash --target esp32s3-box-3 --size short ./examples/blink
#
# Wi-Fi / BLE (espradio) を使うものは CGO_CFLAGS_ALLOW=-fno-short-enums が必要。
# Wi-Fi の YOUR_SSID / YOUR_PASSWORD は書き込むときに自分のものに書き換える
# (下の flash-wifi-* を使うと環境変数から埋め込める)。
# BLE は TinyGo 0.43 以降が前提。古い TinyGo での手順は README を参照。
smoketest: FORCE
	mkdir -p out
	# ---- 基本 ----
	# 全 example の統合ランチャー。ロゴ、ジョイスティックの選択画面、実行中は左で選択画面に戻る
	CGO_CFLAGS_ALLOW=-fno-short-enums tinygo build -o ./out/all.bin --target esp32s3-box-3 --size short -ldflags="-X main.ssid=YOUR_SSID -X main.password=YOUR_PASSWORD" ./examples/all
	# WS2812B を虹色に光らせながら、シリアルにカウンタを出力する
	tinygo build -o ./out/blink.bin         --target esp32s3-box-3 --size short ./examples/blink
	# ST7789 にカラーバーと文字を表示する
	tinygo build -o ./out/display.bin       --target esp32s3-box-3 --size short ./examples/display
	# ボタン (SW1/SW2) とジョイスティックの状態をシリアルに出力する
	tinygo build -o ./out/input.bin         --target esp32s3-box-3 --size short ./examples/input
	# ジョイスティックの生値と可動範囲 (min/max) を計測してシリアルに出力する
	tinygo build -o ./out/joyraw.bin        --target esp32s3-box-3 --size short ./examples/joyraw
	# バッジの全機能デモ
	tinygo build -o ./out/demo.bin          --target esp32s3-box-3 --size short ./examples/demo
	# 基板上の全デバイス (LCD, LED, ボタン, ジョイスティック, AHT21B, Grove, 赤外線, 音) を一度に動作確認する
	tinygo build -o ./out/selftest.bin      --target esp32s3-box-3 --size short ./examples/selftest
	# 音ゲー。降ってくるノーツに合わせてジョイスティックの左、上か下、右と SW2、SW1 を押す
	tinygo build -o ./out/rhythm.bin        --target esp32s3-box-3 --size short ./examples/rhythm
	# 3 リールの目押しスロット。SW1 で開始し、左/中央/右の順に止める
	tinygo build -o ./out/slotgame.bin      --target esp32s3-box-3 --size short ./examples/slotgame
	# ---- センサー (I2C) ----
	# 基板上の AHT21B 温湿度センサーの値をシリアルに出力する
	tinygo build -o ./out/aht21b.bin        --target esp32s3-box-3 --size short ./examples/aht21b
	# Grove コネクタにつないだ DHT20 温湿度センサーの値をシリアルに出力する
	tinygo build -o ./out/dht20.bin         --target esp32s3-box-3 --size short ./examples/dht20
	# Grove (I2C0) と AHT21B (I2C1) の両方の I2C バスをスキャンする
	tinygo build -o ./out/i2cscan.bin       --target esp32s3-box-3 --size short ./examples/i2cscan
	# ---- 音 (I2S, MAX98357) ----
	# SW1 でメロディ、SW2 で和音、ジョイスティックで音程を変えて鳴らす
	tinygo build -o ./out/audio.bin         --target esp32s3-box-3 --size short ./examples/audio
	# I2S の動作確認用 (レジスタダンプ、1kHz サイン波を鳴らし続ける)
	tinygo build -o ./out/audiotest.bin     --target esp32s3-box-3 --size short ./examples/audiotest
	# ---- 赤外線 ----
	# 赤外線の送受信 (NEC フォーマット)
	tinygo build -o ./out/ir.bin            --target esp32s3-box-3 --size short ./examples/ir
	# リモコンの信号を学習してフラッシュに保存し、送信するアプリ
	tinygo build -o ./out/irlearn.bin       --target esp32s3-box-3 --size short ./examples/irlearn
	# ---- Wi-Fi (espradio) ----
	# http://httpbin.org/get を定期的に取得してシリアルに出力する
	CGO_CFLAGS_ALLOW=-fno-short-enums tinygo build -o ./out/wifi-httpget.bin  --target esp32s3-box-3 --size short -ldflags="-X main.ssid=YOUR_SSID -X main.password=YOUR_PASSWORD" ./examples/wifi-httpget
	# HTTP サーバー。基板上の AHT21B の温湿度を返す
	CGO_CFLAGS_ALLOW=-fno-short-enums tinygo build -o ./out/wifi-server.bin   --target esp32s3-box-3 --size short -ldflags="-X main.ssid=YOUR_SSID -X main.password=YOUR_PASSWORD" ./examples/wifi-server
	# HTTP サーバー。ジョイスティックとスイッチの状態をブラウザにリアルタイム表示する
	CGO_CFLAGS_ALLOW=-fno-short-enums tinygo build -o ./out/wifi-joystick.bin --target esp32s3-box-3 --size short -ldflags="-X main.ssid=YOUR_SSID -X main.password=YOUR_PASSWORD" ./examples/wifi-joystick
	# ---- BLE (espradio) ----
	# 周囲の BLE デバイスをスキャンしてシリアルに出力する
	CGO_CFLAGS_ALLOW=-fno-short-enums tinygo build -o ./out/ble-scanner.bin   --target esp32s3-box-3 --size short ./examples/ble-scanner
	# BLE ペリフェラル。温湿度の通知と LED 制御 (webble.html から操作)
	CGO_CFLAGS_ALLOW=-fno-short-enums tinygo build -o ./out/ble-sensor.bin    --target esp32s3-box-3 --size short ./examples/ble-sensor

FORCE:

monitor:
	tinygo monitor --target esp32s3-box-3

# Wi-Fi の example は SSID とパスワードをビルド時に埋め込む必要があるので、
# 環境変数から埋め込んで書き込むターゲットを用意している。次のどちらでも指定できる。
#   make flash-wifi-server SSID=yourssid PASS=yourpassword
#   export WIFI_SSID=yourssid WIFI_PASS=yourpassword; make flash-wifi-server
SSID ?= $(WIFI_SSID)
PASS ?= $(WIFI_PASS)

.PHONY: build-all flash-all check-all build-slotgame flash-slotgame check-slotgame

build-all:
	mkdir -p out
	CGO_CFLAGS_ALLOW=-fno-short-enums tinygo build -o ./out/all.bin --target esp32s3-box-3 --size short -ldflags="-X main.ssid=$(SSID) -X main.password=$(PASS)" ./examples/all

flash-all:
	CGO_CFLAGS_ALLOW=-fno-short-enums tinygo flash --target esp32s3-box-3 --size short -ldflags="-X main.ssid=$(SSID) -X main.password=$(PASS)" ./examples/all

check-all:
	GO111MODULE=off go test ./examples/all/navigation.go ./examples/all/navigation_test.go
	cd examples/all && GO111MODULE=off go run generate.go -check

build-slotgame:
	mkdir -p out
	tinygo build -o ./out/slotgame.bin --target esp32s3-box-3 --size short ./examples/slotgame

flash-slotgame:
	tinygo flash --target esp32s3-box-3 --size short ./examples/slotgame

check-slotgame:
	go test ./examples/slotgame

flash-wifi-httpget: check-ssid
	CGO_CFLAGS_ALLOW=-fno-short-enums tinygo flash --target esp32s3-box-3 --size short -ldflags="-X main.ssid=$(SSID) -X main.password=$(PASS)" ./examples/wifi-httpget

flash-wifi-server: check-ssid
	CGO_CFLAGS_ALLOW=-fno-short-enums tinygo flash --target esp32s3-box-3 --size short -ldflags="-X main.ssid=$(SSID) -X main.password=$(PASS)" ./examples/wifi-server

flash-wifi-joystick: check-ssid
	CGO_CFLAGS_ALLOW=-fno-short-enums tinygo flash --target esp32s3-box-3 --size short -ldflags="-X main.ssid=$(SSID) -X main.password=$(PASS)" ./examples/wifi-joystick

check-ssid:
	@test -n "$(SSID)" || { echo "SSID is empty: set SSID/PASS or WIFI_SSID/WIFI_PASS"; exit 1; }
