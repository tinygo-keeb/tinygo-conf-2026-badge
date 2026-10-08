# slotgame

[sat0ken/tinygo-slotgame](https://github.com/sat0ken/tinygo-slotgame) を
TinyGo Conference 2026 バッジ（ESP32-S3、ST7789 240×240）に移植した
3リールの目押しスロットです。

## 操作

| 操作 | 動作 |
| --- | --- |
| 停止中にbutton_1（SW1） | ゲーム開始（3クレジット消費） |
| 回転中にSW1を押す | 左 → 中央 → 右の順に、1回につき1リールを止める |
| GAME OVER中にSW1 | 50クレジットに戻す。もう一度押すとゲーム開始 |
| `all` から起動した場合、Joystick左 | プログラム選択画面へ戻る |

ゲームの操作はすべてbutton_1（SW1）です。押すたびに一度離してください。
長押しでは次のリールは止まりません。リール下の `SW1` ラベルは、次に止める
リールが赤、順番待ちが暗い赤、停止したリールが灰色になります。
開始直後の300msは停止できません。

リール配列は固定で乱数は使いません。押してから次のコマ境界まで、最大1コマ
滑って停止します。中段の赤い線に図柄がそろうと、次の配当になります。

| 中段の役 | 配当 |
| --- | --- |
| 7 × 3 | 100 |
| BAR × 3 | 50 |
| 茶色のGopher × 3 | 15 |
| チェリー × 3 | 10 |
| 青いGopher × 3 | 8 |
| 左リールがチェリー | 2 |

最初に当てはまる役だけを払い出し、残りが3クレジット未満ならGAME OVERです。
クレジットはRAMに保持するため、電源を切ったり `all` の選択画面へ戻ったりすると
初期化されます。元のゲームと同様に効果音はありません。

## ビルド・書き込み

`firmware/` で実行します。

```sh
make build-slotgame
make flash-slotgame
# 手動で書き込む場合
tinygo flash --target esp32s3-box-3 --size short ./examples/slotgame
```

`make flash-all` で書き込む統合版にも `slotgame` が含まれます。

## テスト・画面確認

```sh
make check-slotgame
go test ./examples/slotgame -run TestPreview -preview -preview-dir /tmp/slotgame-preview
```

`slot.go` はハードウェアに依存せず、配当・停止位置・ボタンの押下・ゲーム進行・
描画範囲をホストのGoで検証できます。描画用RAMは図柄5枚とリール1本分の約56KBで、
全画面のフレームバッファは確保しません。

## 移植元・画像のクレジット

ゲーム本体、テスト、Gopherスプライトの移植元は
[tinygo-slotgame の e61ecf541d6ef2643899a1041f0d8602801aa867](https://github.com/sat0ken/tinygo-slotgame/tree/e61ecf541d6ef2643899a1041f0d8602801aa867)
です。LCD初期化・ボタン入力を `badge` の実装に置き換え、1ボタン操作を有効にし、
操作表示をSW1に変更しています。スプライトは移植元の生成済みデータを使っています。

The Go gopher was designed by [Renée French](https://reneefrench.blogspot.com/),
licensed under [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/).
