# firmware

TinyGo Conference 2026 バッジ (ESP32-S3-DevKit ベース) の TinyGo ファームウェア。

## ビルド・書き込み

```sh
cd firmware
tinygo flash --target esp32s3-box-3 --size short ./examples/blink
tinygo monitor --target esp32s3-box-3
```

`make build` で全 example のコンパイル確認、`make flash-<name>` で書き込みができる。

## 構成

| パス | 内容 |
| --- | --- |
| `badge/` | ピン割り当てと各ペリフェラルの初期化ヘルパー |
| `ws2812s3/` | ESP32-S3 (240MHz) 用 WS2812B ドライバ |
| `i2s/` | ESP32-S3 の I2S0 + GDMA を使った 16bit ステレオ音声出力ドライバ |
| `examples/blink` | WS2812B を虹色に点灯 |
| `examples/display` | ST7789 にカラーバーと文字を表示 |
| `examples/input` | SW1/SW2 とジョイスティックの状態をシリアル出力 |
| `examples/aht21b` | 温湿度センサーの値をシリアル出力 |
| `examples/i2cscan` | Grove / AHT21B の I2C バスをスキャン |
| `examples/ir` | 赤外線受信 (NEC) と、ボタン押下で赤外線送信 |
| `examples/audio` | MAX98357 から音階・メロディ・ビープを鳴らす |
| `examples/audiotest` | I2S の動作確認用。診断出力を出したあと 1kHz の正弦波を鳴らし続ける |
| `examples/demo` | 上記をまとめた全機能デモ |

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
- **I2C 初期化**: `machine.I2C.Configure` は最後のバスクリアで完了ビットを無限に待つため、
  プルアップのない (浮いた) バスでは止まることがある。また Configure はピン設定を上書きするので
  事前に内部プルアップを有効にしても外れる。`badge.ConfigureI2C()` は同じ手順を内部プルアップ
  対応と有限待ちで自前実装したもので、Grove (`ConfigureGroveI2C`、内部プルアップ有効、100kHz) と
  AHT21B (`ConfigureSensorI2C`) はこれを使う。
- **I2C スキャン**: TinyGo の ESP32 I2C ドライバは読み出し時のアドレス NACK をエラーにせず、
  書き込みでは NACK 後に未送信データが TX FIFO に残って次のトランザクションがジェネラルコール
  になる (AHT21B はこれに ACK する)。そのため `badge.ProbeI2C()` はレジスタを直接操作して
  アドレスバイトのみを送り、失敗時は `badge.ResetI2C()` で FSM と FIFO をリセットしている。
- **USB シリアル**: `print` の出力は改行まで送られない。改行なしで長い処理をすると
  何も表示されないように見えるので、区切りには `println` を使う。
- **割り込み内での出力**: 赤外線受信 (irremote) などのコールバックは GPIO 割り込みの中で呼ばれる。
  その中で `println` を使うと USB シリアル出力が止まることがあるので、割り込み内ではデータを
  保存するだけにしてメインループで出力する (`examples/ir` 参照)。
- **I2S (MAX98357)**: TinyGo の machine パッケージは ESP32-S3 の I2S を未サポート。
  `i2s/` にレジスタ直叩きで実装している (ESP-IDF v5.1 の i2s_ll.h / gdma_ll.h / i2s_std.c の
  手順を移植)。ESP32-S3 の I2S は GDMA 経由でしか送れないため、DMA バッファをリング状につないで
  連続送信し、`Write` はそのバッファを順に埋める。リングは再生され続けるので、鳴らし終えたら
  `Silence()` (badge.ToneGenerator なら `Stop()`) で無音にすること。フォーマットは Philips 標準、
  16bit、2ch、MCLK = fs*256、BCLK = fs*32 (スロット幅 16。`Config.SlotBits` で 32 も可)。
  GDMA はチャネル 0 を使う。
- **MAX98357 の SD ピン**: 回路図では未接続。Adafruit 製モジュールは基板上の 1MΩ でプルアップされ
  Vin=5V なら SD が約 0.45V (ステレオ平均モード) になるが、これで動かない個体があった。
  SD を Vin に直結 (左チャネルのみ) したモジュールで動作確認済み。ファームウェアは L/R に同じ
  データを流すのでどちらのモードでも音は同じ。次の基板では SD のプルアップを基板側に持たせたい。
