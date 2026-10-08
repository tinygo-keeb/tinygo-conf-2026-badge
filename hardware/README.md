# ハードウェア (tinygo-conf-2026)

[English](README.en.md)

TinyGo Conference 2026 向け開発ボード (devkit) の KiCad プロジェクトです。
ESP32-S3-DevKit を中心に、ディスプレイ・オーディオ・入力デバイスなどを 1 枚の基板にまとめています。

## 主な搭載部品・機能

- **ESP32-S3-DevKit** — メインボード (ピンソケット経由で搭載)
- **ST7789 ディスプレイ** — SPI 接続 (CS / DC / RES / SCL / DIN)、8 ピンコネクタ
- **MAX98357 (Adafruit)** — I2S オーディオアンプ (BCLK / LRC / DIN) + スピーカー
- **RGB LED × 2** — シリアル接続
- **アナログジョイスティック (ALPS RKJXV122400R)** — X / Y 軸 + プッシュボタン
- **プッシュスイッチ × 2**
- **赤外線 LED / 赤外線受信** — IR_LED / IR_DATA
- **I2C コネクタ (4 ピン)** — 外部モジュール接続用 (Grove 互換フットプリントあり)
- **AHT21B** — 温湿度センサー

## ファイル構成

| パス | 内容 |
| --- | --- |
| `tinygo-conf-2026.kicad_pro` | KiCad プロジェクトファイル |
| `tinygo-conf-2026.kicad_sch` | 回路図 |
| `tinygo-conf-2026.kicad_pcb` | 基板レイアウト |
| `tinygo-conf-2026-devkit/` | 製造用ガーバー・ドリルデータ |
| `lib/` | シンボル / フットプリントライブラリ (下記サブモジュール含む) |
| `lib/sglib.kicad_sym`, `lib/sglib.pretty/` | 自作ライブラリ (ジョイスティック、Grove コネクタなど) |
| `fp-lib-table`, `sym-lib-table` | プロジェクト用ライブラリテーブル |

## 必要環境

- KiCad 9.0 以降

## セットアップ

ライブラリの一部は git サブモジュールとして管理しています。リポジトリのルートで以下を実行してください。

```sh
git submodule update --init --recursive
```

その後 `hardware/tinygo-conf-2026.kicad_pro` を KiCad で開きます。

### サブモジュール一覧

| パス | 用途 | 取得元 |
| --- | --- | --- |
| `lib/espressif` | ESP32 系シンボル | espressif/kicad-libraries |
| `lib/kbd` | ESP32-S3-Devkit シンボルなど | foostan/kbd |
| `lib/MAX98357` | MAX98357 アンプ | besi/kicad-adafruit-MAX98357 |
| `lib/st7789` | ST7789 ディスプレイ | BennyLuca/Kicad_Components_Library |
| `lib/sparkfun` | SparkFun ライブラリ | sparkfun/SparkFun-KiCad-Libraries |

## 製造データ

`tinygo-conf-2026-devkit/` に 2 層基板のガーバーデータ (表裏の銅箔・レジスト・ペースト・シルク、外形、PTH/NPTH ドリル) 一式が出力済みです。

## 組み立て

部品一覧と写真付きの手順は [組み立てガイド](build/build.md) を参照してください。
