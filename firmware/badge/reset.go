package badge

import (
	"device/esp"
	"time"
)

// Reset はチップ全体をソフトウェアリセットする (RTC_CNTL の SW_SYS_RST)。
// 戻ってこない。USB シリアルの出力を送り切るために少し待ってから実行する。
//
// espradio は無線の初期化を一度しかできず、接続に失敗したあと NetConnect を
// 呼び直しても "already enabled" で失敗する。espradio の Enable のコメントに
// あるとおり、失敗したらデバイスをリセットしてやり直す必要がある。
func Reset() {
	time.Sleep(100 * time.Millisecond)
	esp.RTC_CNTL.SetOPTIONS0_SW_SYS_RST(1)
	for {
	}
}
