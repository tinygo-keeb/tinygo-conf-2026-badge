# TinyGo Conf 2026 Badge

[English](README.en.md)

TinyGo Conference 2026 向けバッジのハードウェア設計と TinyGo ファームウェアをまとめたリポジトリです。ESP32-S3-DevKit を中心に、ディスプレイ、入力デバイス、音声、赤外線、温湿度センサーなどを搭載しています。

## 主な機能

| 機能 | ハードウェア | ファームウェアの例 |
| --- | --- | --- |
| 表示 | ST7789 ディスプレイ | [`display`](firmware/examples/display)、[`demo`](firmware/examples/demo) |
| 入力 | ジョイスティック、スイッチ × 2 | [`input`](firmware/examples/input)、[`rhythm`](firmware/examples/rhythm) |
| 光と音 | RGB LED × 2、MAX98357 アンプとスピーカー | [`blink`](firmware/examples/blink)、[`audio`](firmware/examples/audio) |
| センサー・拡張 | AHT21B 温湿度センサー、Grove 互換 I2C コネクタ | [`aht21b`](firmware/examples/aht21b)、[`i2cscan`](firmware/examples/i2cscan) |
| 赤外線 | 赤外線 LED・受信モジュール | [`ir`](firmware/examples/ir)、[`irlearn`](firmware/examples/irlearn) |
| 無線通信 | ESP32-S3 の Wi-Fi・BLE | [`wifi-server`](firmware/examples/wifi-server)、[`ble-sensor`](firmware/examples/ble-sensor) |

全体の動作確認には [`selftest`](firmware/examples/selftest) を使えます。

## 組み立て

必要な部品と写真付きの組み立て手順は [ビルドガイド](hardware/build/build.md) を参照してください。

## ドキュメントとファイル

| パス | 内容 |
| --- | --- |
| [hardware/README.md](hardware/README.md) | 基板の構成、KiCad のセットアップ、製造データ |
| [hardware/build/build.md](hardware/build/build.md) | 部品一覧と写真付きの組み立て手順 |
| [hardware/tinygo-conf-2026.kicad_pro](hardware/tinygo-conf-2026.kicad_pro) | KiCad プロジェクト。回路図と基板レイアウトは同じディレクトリにあります |
| [firmware/README.md](firmware/README.md) | ビルド・書き込み方法、サンプル一覧、ピン割り当て、実装上の注意点 |
| [firmware/Makefile](firmware/Makefile) | サンプルのビルド確認や Wi-Fi サンプルの書き込み |

## 使い始める

### ハードウェア

KiCad 9.0 以降を使用します。KiCad ライブラリの一部はサブモジュールなので、リポジトリのルートで取得してください。

```sh
git submodule update --init --recursive
```

[KiCad プロジェクト](hardware/tinygo-conf-2026.kicad_pro) を開き、バッジの組み立てには [組み立てガイド](hardware/build/build.md) を参照してください。

### ファームウェア

TinyGo を用意し、ESP32-S3-DevKit を接続して以下を実行します。

```sh
cd firmware
tinygo flash --target esp32s3-box-3 --size short ./examples/blink
tinygo monitor --target esp32s3-box-3
```

全サンプルのビルド確認は `firmware/` で `make smoketest` を実行します。Wi-Fi・BLE の設定や各サンプルの使い方は [ファームウェアのREADME](firmware/README.md) を参照してください。
