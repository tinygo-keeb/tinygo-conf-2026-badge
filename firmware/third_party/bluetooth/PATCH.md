# tinygo.org/x/bluetooth v0.16.0 のローカル修正版

go.mod の `replace tinygo.org/x/bluetooth => ./third_party/bluetooth` で使う。
`_test.go` と examples/ を除いた v0.16.0 のコピーに、次の修正を加えている。

## att_hci.go: Find By Type Value Request (0x06) に応答する

HCI バックエンドの ATT サーバーは Find By Type Value Request を受け取っても
デバッグ表示するだけで応答を返さなかった。Windows (Chrome の Web Bluetooth など)
はサービス探索でこの要求を使うため、応答がないと ATT のトランザクション
タイムアウト (30 秒) 後に切断される。

修正では、要求のアトリビュート型が Primary Service (0x2800) のとき、値
(16bit または 128bit のサービス UUID、リトルエンディアン) に一致する
ローカルサービスの (開始ハンドル, 終了ハンドル) を Find By Type Value
Response (0x07) で返す。該当がなければ Error Response (Attribute Not Found)。

アップストリームに取り込まれたら、このディレクトリと replace を削除する。
