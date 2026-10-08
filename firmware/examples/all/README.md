# all

`firmware/examples` の20本を、1つのファームウェアから選択して実行します。
`slotgame` は3リールの目押しスロットです。button_1（SW1）で開始し、押すたびに
左 → 中央 → 右の順にリールを止めます（[操作説明](../slotgame/README.md)）。

## 操作

| 画面 | 操作 | 動作 |
| --- | --- | --- |
| TOP | 起動時 | `img/logo.jpg` を240×240のLCDに表示 |
| TOP | `button_1` / SW1 | プログラム選択画面を開く |
| 選択画面 | Joystick 上・下 | 選択を移動。長く倒すと連続移動し、一覧の端で折り返す |
| 選択画面 | SW1 またはJoystick押し込み | 選択中のプログラムを起動 |
| 選択画面・起動待ち | Joystick 左 | TOPへ戻る |
| プログラム実行中 | Joystick 左 | 約150msの入力確認後、本体をリセットしてプログラム選択画面へ戻る |

起動時はJoystickを中立にしてください。プログラムを選んだ後も、ボタンと
Joystickを離してから実行を開始します。各プログラムのSW1/SW2などの操作は
元のexampleと同じです。シリアル出力だけのexampleでは、LCDに実行中の名前と
選択画面へ戻る操作を表示します。戻ると直前に実行していたプログラムが選択され、
Joystickとボタンを離してから次の操作を受け付けます。

実行中の左操作を選択画面へ戻る操作として使うため、次の違いがあります。

- `rhythm`: 左端のレーンをJoystick押し込みで操作します。画面のラベルは `JOY` です。
- `irlearn`: 名前編集のカーソルを戻す操作はJoystick押し込み、進める操作は右です。
- `selftest`: 左方向を最後に検査してください。左に倒すと検査後に選択画面へ戻ります。
- `input` / `joyraw` / `demo` / `audio` / `wifi-joystick`: 左方向を使う操作でも、
  左へ倒し続けると選択画面へ戻ります。

実行中のプログラムから戻る際は、音声DMA、赤外線割り込み、HTTPサーバーや
BLE/Wi-Fiの処理を残さないため、本体をリセットします。戻り先と選択位置はRTCの
レジスタに一時的に保存し、再起動後に選択画面を開きます。通常の電源投入時は
ロゴ画面になります。`irlearn` の保存済み
データは残り、`rhythm` のベストスコアなどRAMだけの状態は初期化されます。
Wi-Fi接続失敗時も既存の `wifi.Connect` によるリセットでTOPへ戻ります。

## ビルド・書き込み

`firmware/` で実行します。全機能を含むため `CGO_CFLAGS_ALLOW` が必要です。

```sh
make build-all
make flash-all
tinygo monitor --target esp32s3-box-3
```

Wi-Fiを利用するときはSSIDとパスワードを指定します。3つのWi-Fiプログラムが
同じ設定を使います。未指定でもTOPや他のプログラムを利用でき、Wi-Fiを選ぶと
設定が必要であることをLCDに表示します。

```sh
make build-all SSID=yourssid PASS=yourpassword
make flash-all SSID=yourssid PASS=yourpassword
# WIFI_SSID / WIFI_PASS 環境変数でも指定できます。
```

手動でビルドする場合:

```sh
mkdir -p out
CGO_CFLAGS_ALLOW=-fno-short-enums tinygo build -o ./out/all.bin \
  --target esp32s3-box-3 --size short \
  -ldflags="-X main.ssid=yourssid -X main.password=yourpassword" ./examples/all
```

BLE用のROMシンボルを含まない古いTinyGoでは、既存のカスタムターゲットを使います。

```sh
CGO_CFLAGS_ALLOW=-fno-short-enums tinygo build -o ./out/all.bin \
  --target ./targets/esp32s3-box-3-ble.json -tags espradio --size short \
  -ldflags="-X main.ssid=yourssid -X main.password=yourpassword" ./examples/all
```

## 元のexample・ロゴを変更したとき

元のexampleはそれぞれ単独でビルドできます。`example_*.go` と `programs.go` は
元のソースを名前空間を分けて取り込んだ生成ファイルです。変更後は次を実行します。
新しいexampleも、隣のディレクトリに `main` 関数を置けば一覧に追加されます。
`*_test.go` は統合せず、`//go:embed` で指定された画像は `assets/<example>/` に
コピーします。画像の指定はワイルドカードを含まない相対ファイルパスにしてください。

```sh
go generate ./examples/all
```

左操作と競合する入力の変更は `generate.go` の `adapt` に記述しています。
対象コードが変更された場合、生成処理はエラーを出すので、この定義も更新してください。

`img/logo.jpg` は元画像です。生成処理では拡張子によらずJPEG/PNGの内容を判別し、
縦横比を保って240×240のRGB565に縮小した `img/logo.rgb565` を作ります。
ファームウェアにはこの約115KBの変換済み画像を埋め込みます。画像のデコードは
PCで行い、LCDへの転送は16行ずつなので、大きな元画像を本体のRAMに展開しません。

確認コマンド:

```sh
make check-all
make build-all SSID=buildcheck PASS=buildcheck
```
