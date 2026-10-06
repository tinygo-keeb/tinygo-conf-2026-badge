# firmware

TinyGo Conference 2026 バッジ (ESP32-S3-DevKit ベース) の TinyGo ファームウェア。

## ビルド・書き込み

```sh
cd firmware
tinygo flash --target esp32s3-box-3 --size short ./examples/blink
tinygo monitor --target esp32s3-box-3
```

`Makefile` の `smoketest` に全 example のビルドコマンドを 1 行ずつそのまま書いてある
(`make smoketest` で全部ビルドして `out/` に出力する)。書き込みたいときは該当する行を
コピーして、`build -o ./out/xxx.bin` を `flash` に変えて実行する。

Wi-Fi の example は SSID とパスワードをビルド時に埋め込む。Makefile の行は
`YOUR_SSID` / `YOUR_PASSWORD` のプレースホルダなので、自分のものに書き換えるか、
環境変数から埋め込む `make flash-wifi-<name>` を使う。

```sh
make flash-wifi-server SSID=yourssid PASS=yourpassword
# 環境変数でも可 (WIFI_SSID / WIFI_PASS、または SSID / PASS)
export WIFI_SSID=yourssid WIFI_PASS=yourpassword
make flash-wifi-server
# 手動なら
CGO_CFLAGS_ALLOW=-fno-short-enums tinygo flash --target esp32s3-box-3 --size short \
  -ldflags="-X main.ssid=yourssid -X main.password=yourpassword" ./examples/wifi-server
```

BLE の example は TinyGo 0.43 以降なら Makefile の行のとおり通常のターゲットでビルドできる。
それより古い TinyGo では専用ターゲット (`targets/esp32s3-box-3-ble.json`) と
`-tags espradio` を指定する (firmware/ で実行。リンカスクリプトのパスは firmware/ からの相対)。
`-tags bledebug` を足すと bluetooth パッケージのデバッグ出力 (ATT/HCI の処理) がシリアルに出る。

```sh
CGO_CFLAGS_ALLOW=-fno-short-enums tinygo flash --target esp32s3-box-3 --size short ./examples/ble-sensor
# 古い TinyGo なら
CGO_CFLAGS_ALLOW=-fno-short-enums tinygo flash --target ./targets/esp32s3-box-3-ble.json \
  -tags espradio --size short ./examples/ble-sensor
```

## 構成

| パス | 内容 |
| --- | --- |
| `badge/` | ピン割り当てと各ペリフェラルの初期化ヘルパー |
| `ws2812s3/` | ESP32-S3 (240MHz) 用 WS2812B ドライバ |
| `i2s/` | ESP32-S3 の I2S0 + GDMA を使った 16bit ステレオ音声出力ドライバ |
| `wifi/` | espradio で Wi-Fi に接続するヘルパー (失敗時はリセットして再試行) |
| `flashstore/` | ROM 関数で SPI フラッシュの一部 (0x1F0000 から 64KB) を読み書きする設定保存用パッケージ |
| `targets/` | BLE 用のカスタムターゲット (アップストリーム TinyGo の esp32s3.ld を同梱) |
| `examples/blink` | WS2812B を虹色に点灯 |
| `examples/display` | ST7789 にカラーバーと文字を表示 |
| `examples/input` | SW1/SW2 とジョイスティックの状態をシリアル出力 |
| `examples/joyraw` | ジョイスティックの生値と可動範囲 (min/max) を計測してシリアル出力 |
| `examples/aht21b` | 温湿度センサーの値をシリアル出力 |
| `examples/i2cscan` | Grove / AHT21B の I2C バスをスキャン |
| `examples/dht20` | Grove につないだ DHT20 (AHT20 互換) の温湿度をシリアル出力 |
| `examples/wifi-httpget` | Wi-Fi に接続して http://httpbin.org/get を取得 (net/http) |
| `examples/wifi-server` | Wi-Fi に接続して AHT21B の温湿度を返す HTTP サーバー (httphi) |
| `examples/wifi-joystick` | ジョイスティックの XY とスイッチの状態をブラウザにリアルタイム表示 |
| `examples/ble-scanner` | BLE で周囲のデバイスをスキャンしてアドレス、RSSI、名前を表示 |
| `examples/ble-sensor` | BLE ペリフェラル。温湿度 (Environmental Sensing)、WS2812B 2 個の色書き込み、ボタン通知、LED の自動/手動モード。名前と LCD にチップ固有の ID を表示 |
| `examples/ble-sensor/webble.html` | 上記に Web Bluetooth で接続して操作するページ (Chrome / Edge で開く) |
| `examples/ir` | 赤外線受信 (NEC) と、ボタン押下で赤外線送信 |
| `examples/irlearn` | リモコンの信号を学習して送信するアプリ。一覧、登録、名前編集、削除。フラッシュに保存 |
| `examples/audio` | MAX98357 から音階・メロディ・ビープを鳴らす |
| `examples/audiotest` | I2S の動作確認用。診断出力を出したあと 1kHz の正弦波を鳴らし続ける |
| `examples/demo` | 上記をまとめた全機能デモ |
| `examples/rhythm` | 音ゲー。上から降ってくるノーツに合わせて 5 つのキー (ジョイスティックの左、上か下、右と SW2、SW1) を押す。BGM はその場で合成し、メロディの音の高さとレーンが対応している。コンボ数に応じて画面の演出が増える。EASY / NORMAL |
| `examples/selftest` | 基板上の全デバイスを一度に動作確認するセルフテスト。ボタン、ジョイスティック上下左右、AHT21B、赤外線の自己受信、I2S を自動判定し、すべて済むと ALL OK。LCD、LED、スピーカー、Grove は画面の表示と音で目視確認 |

## ピン割り当て

`tinygo-conf-2026.kicad_sch` から抽出。定数は `badge/badge.go` にある。

| 機能 | 信号 | GPIO | 備考 |
| --- | --- | --- | --- |
| LCD ST7789 (J3) | SCL | 12 | SPI0 (FSPI) |
| | SDA (MOSI) | 11 | |
| | CS | 10 | |
| | DC | 5 | |
| | RES | 4 | |
| | BLK | - | 3V3 直結 |
| I2S MAX98357 (U3) | BCLK | 45 | I2S0 (後述) |
| | LRC | 21 | |
| | DIN | 47 | |
| WS2812B x2 (D1, D2) | DIN | 16 | D1 -> D2 直列 |
| ジョイスティック (U2) | X | 7 | ADC1_CH6 |
| | Y | 6 | ADC1_CH5 |
| | BTN | 15 | 押下で GND、内部プルアップ |
| SW1 | | 13 | 押下で GND、内部プルアップ |
| SW2 | | 14 | 押下で GND、内部プルアップ |
| 赤外線 LED (D5) | | 17 | High で点灯、PWM 38kHz で変調 |
| 赤外線受信 (J2) | DATA | 18 | |
| Grove I2C (J1) | SDA | 8 | I2C0 |
| | SCL | 9 | |
| AHT21B (J4) | SDA | 41 | I2C1 |
| | SCL | 42 | |

拡張ヘッダ J5 (2x8):

```
 1: 3V3     2: 3V3
 3: GPIO1   4: GPIO2
 5: GPIO3   6: GPIO38
 7: GPIO39  8: GND
 9: GND    10: GPIO40
11: GPIO41 12: GPIO42   (AHT21B SDA/SCL と共用)
13: GPIO45 14: GPIO46   (GPIO45 は I2S BCLK と共用)
15: GPIO48 16: VCC (5V)
```

## 注意点

- **ターゲット**: `esp32s3-box-3` を使う。`esp32s3-generic` は `machine.CPUFrequency()`
  や SPI のデフォルトピンが定義されておらず、drivers の一部がビルドできない。
- **WS2812B**: `tinygo.org/x/drivers/ws2812` は xtensa では 80/160MHz のみ対応で、
  ESP32-S3 は 240MHz で動作するため使えない。`ws2812s3/` に 240MHz 用の
  タイミングでビットバンギング実装を置いている。
- **LCD のちらつき**: ディスプレイに直接描くと部分ごとの書き換えが見えてちらつく。
  `badge.NewFramebuffer(display)` で 1 画面分 (RGB565、約 115KB) のオフスクリーンバッファを
  作り、そこに描いてから `Display()` で一括転送する (`examples/demo` 参照)。tinyfont や
  tinydraw の描画先としてそのまま渡せる。
- **I2C 初期化**: `machine.I2C.Configure` は最後のバスクリアで完了ビットを無限に待つため、
  プルアップのない (浮いた) バスでは止まることがある。また Configure はピン設定を上書きするので
  事前に内部プルアップを有効にしても外れる。`badge.ConfigureI2C()` は同じ手順を内部プルアップ
  対応と有限待ちで自前実装したもので、Grove (`ConfigureGroveI2C`、内部プルアップ有効、100kHz) と
  AHT21B (`ConfigureSensorI2C`) はこれを使う。
- **I2C スキャン**: TinyGo の ESP32 I2C ドライバは読み出し時のアドレス NACK をエラーにせず、
  書き込みでは NACK 後に未送信データが TX FIFO に残って次のトランザクションがジェネラルコール
  になる (AHT21B はこれに ACK する)。そのため `badge.ProbeI2C()` はレジスタを直接操作して
  アドレスバイトのみを送り、失敗時は `badge.ResetI2C()` で FSM と FIFO をリセットしている。
- **ジョイスティックの可動範囲**: スティックはハット側で可動範囲を絞ってあり、ADC の生値は中心から
  ±16000..17000 カウント (全域 0..65535 の約半分) しか動かない。`badge.Joystick` は `Range` (既定 16000)
  で正規化するので、いっぱい倒すと 1000 に届く。範囲が変わったら `examples/joyraw` で計測して `Range` を直す。
  静止時のノイズは ±100 カウント程度なので `DeadZone` は 50 (= 800 カウント) にしてある。
- **USB シリアル**: `print` の出力は改行まで送られない。改行なしで長い処理をすると
  何も表示されないように見えるので、区切りには `println` を使う。
- **割り込み内での出力**: 赤外線受信 (irremote) などのコールバックは GPIO 割り込みの中で呼ばれる。
  その中で `println` を使うと USB シリアル出力が止まることがあるので、割り込み内ではデータを
  保存するだけにしてメインループで出力する (`examples/ir` 参照)。
- **フラッシュへの保存**: TinyGo の machine パッケージには ESP32-S3 用のフラッシュ API が無いので、
  `flashstore/` は ROM に固定アドレスで存在する `esp_rom_spiflash_read/write/erase_sector/unlock`
  を CGo から直接呼ぶ (アドレスは ESP-IDF v5.1.2 の esp32s3.rom.ld)。手順は TinyGo の
  ESP32-C3 用ドライバと同じで、割り込み禁止のまま ROM 関数を呼ぶ。TinyGo のイメージヘッダは
  フラッシュサイズを 2MB と書くので ROM ドライバは 2MB を超える erase/write を拒否する。そのため
  保存領域は 2MB 未満の末尾 (0x1F0000 から 64KB) に置く。読み出しもキャッシュ経由ではなく
  ROM 関数で行う。
- **I2S (MAX98357)**: TinyGo の machine パッケージは ESP32-S3 の I2S を未サポート。
  `i2s/` にレジスタ直叩きで実装している (ESP-IDF v5.1 の i2s_ll.h / gdma_ll.h / i2s_std.c の
  手順を移植)。ESP32-S3 の I2S は GDMA 経由でしか送れないため、DMA バッファをリング状につないで
  連続送信し、`Write` はそのバッファを順に埋める。リングは再生され続けるので、鳴らし終えたら
  `Silence()` (badge.ToneGenerator なら `Stop()`) で無音にすること。フォーマットは Philips 標準、
  16bit、2ch、MCLK = fs*256、BCLK = fs*32 (スロット幅 16。`Config.SlotBits` で 32 も可)。
  GDMA はチャネル 0 を使う。
- **Wi-Fi**: `tinygo.org/x/espradio` (TinyGo 0.41 以降、Espressif のバイナリブロブ + 純 Go の
  TCP/IP スタック lneto) を使う。espradio の C コードが `-fno-short-enums` を要求するので、
  ビルド時に環境変数 `CGO_CFLAGS_ALLOW=-fno-short-enums` が必要 (Makefile の該当行に前置してある)。
  SSID/パスワードは `-ldflags="-X main.ssid=... -X main.password=..."` で埋め込む。埋め込まないと
  example は起動時に `failure: ssid is empty` を繰り返す (このとき Wi-Fi のコードは
  デッドコードとして落ちるので、バイナリが極端に小さくなる)。HTTP サーバーは espradio 推奨の
  `httphi` (リクエストごとにヒープを使わない) を使う。`net/http` は接続ごとに約 10kB を
  ヒープに確保するため、長時間動かすと GC の断片化で止まることがある (espradio の README 参照)。
  espradio は無線の初期化を一度しかできず、接続に失敗したあと `NetConnect` を呼び直しても
  `already enabled` で失敗する。`wifi.Connect()` は失敗時に 5 秒待ってチップをソフトリセット
  (`badge.Reset()`) し、最初からやり直す。モニタを閉じた直後などは AP 側に前のセッションが残って
  `auth expired` になることがあるが、この再試行でつながる。
  espradio を import するだけで (使わなくても) 初期化データがリンクされ、フラッシュ約 170KB、
  RAM 約 160KB が増えるので、Wi-Fi のヘルパーは `badge` ではなく別パッケージ `wifi` に置いている。
- **BLE**: `tinygo.org/x/bluetooth` の espradio バックエンド (ビルドタグ `espradio`) を使う。
  TinyGo 0.43 以降は ESP32 ターゲットに `espradio` タグが標準で付き、ESP32-S3 の BT ROM
  シンボルもリンカスクリプトに含まれるので、通常のターゲットでビルドできる。
  それより古い TinyGo (0.42.0-dev など) ではどちらも無く、espradio の BLE ブロブ
  (libbtdm_app.a) が参照する ROM シンボル (`r_osi_funcs_p` など約 1000 個) でリンクに失敗する。
  その場合のために、アップストリーム TinyGo (dev ブランチ、2026-09-25 取得) の `esp32s3.ld` を
  `targets/esp32s3-ble.ld` として同梱し、`targets/esp32s3-box-3-ble.json` (esp32s3-box-3 を継承)
  から参照している。古い TinyGo ではこのカスタムターゲットと `-tags espradio` を指定する。
  TinyGo のターゲット JSON の `linkerscript` は cwd (firmware/) からの相対パスで解決される。
- **バッジの識別**: `badge.SerialNumber()` は eFuse の MAC アドレス (チップ固有) の下位 3 バイトを
  16 進 6 桁にした ID を返す。ble-sensor はアドバタイズ名を `TinyGo Conf 2026 #XXXXXX` にし、
  同じ ID を LCD に表示するので、ワークショップで複数のバッジが同時に動いていても見分けられる。
- **espradio の import 位置**: TinyGo はパッケージの init を import パスの辞書順に実行し、
  コンパイル時に評価できない init (espradio の C 呼び出し) に当たると、それ以降のパッケージの
  init を実行時に回す。このモジュール (`github.com/sago35/...`) は `github.com/soypat/lneto` や
  `net/http` より辞書順で前なので、このモジュール内のパッケージから espradio を import すると
  unicode などのテーブル初期化が実行時に回り、RAM が約 70KB 増える (実測: 115KB → 240KB)。
  そのため `wifi.Connect()` は espradio を import せず `netlink.Netlinker` を受け取り、
  `Esplink` の生成は example の main で行う。
- **MAX98357 の SD ピン**: 回路図では未接続。Adafruit 製モジュールは基板上の 1MΩ でプルアップされ
  Vin=5V なら SD が約 0.45V (ステレオ平均モード) になるが、これで動かない個体があった。
  SD を Vin に直結 (左チャネルのみ) したモジュールで動作確認済み。ファームウェアは L/R に同じ
  データを流すのでどちらのモードでも音は同じ。次の基板では SD のプルアップを基板側に持たせたい。
