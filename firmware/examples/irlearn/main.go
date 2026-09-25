// リモコンの信号を学習して送信するアプリ。
//
// 画面には登録済みのリモコン信号が一覧で並ぶ。ジョイスティックの上下で選び、
//   - SW1: 選んだ信号を送信する
//   - SW2: メニューを開く (Delete / Rename / Back)。上下で選び SW1 で決定、SW2 で戻る
//
// 一覧の末尾の "+ Register" を選んで SW1 を押すと登録モードになる。
//   - 登録モード: 受信モジュールに向けてリモコンのボタンを押すと信号を記録する。
//     SW1 で決定、SW2 でキャンセル。決定後に名前を入力する
//   - 名前入力: 左右でカーソル移動、上下で文字変更。SW1 で決定、SW2 でキャンセル
//
// 信号は受信モジュールの生のパルス長 (マイクロ秒) をそのまま記録し、38kHz の
// キャリアで再生するので、NEC 以外 (AEHA、SONY など) のリモコンにも使える。
// 登録内容はフラッシュ (flashstore) に保存され、電源を切っても残る。
package main

import (
	"image/color"
	"machine"
	"time"

	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/freesans"

	"github.com/sago35/tinygo-conf-2026-badge/firmware/badge"
)

var (
	black  = color.RGBA{0, 0, 0, 255}
	white  = color.RGBA{255, 255, 255, 255}
	gray   = color.RGBA{140, 140, 140, 255}
	dark   = color.RGBA{40, 40, 40, 255}
	cyan   = color.RGBA{0, 200, 255, 255}
	green  = color.RGBA{0, 255, 0, 255}
	yellow = color.RGBA{255, 220, 0, 255}
	red    = color.RGBA{255, 60, 60, 255}
)

// 画面の状態
type screen int

const (
	screenList screen = iota
	screenMenu
	screenLearn
	screenName
)

// 名前入力で使える文字
const nameChars = " ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-"

type app struct {
	fb *badge.Framebuffer
	tx *badge.IRTransmitter

	remotes []Remote
	cursor  int // 一覧のカーソル (len(remotes) は "+ Register")
	scroll  int

	screen   screen
	menuIdx  int
	message  string // 画面下部に短時間出すメッセージ
	msgUntil time.Time

	// 登録モード
	learned []uint16

	// 名前入力
	nameBuf   [NameLen]byte
	nameCur   int
	nameIsNew bool // true なら登録の続き、false なら改名
}

func main() {
	time.Sleep(500 * time.Millisecond)
	display := badge.NewDisplay()
	a := &app{fb: badge.NewFramebuffer(display)}
	in := newInput()

	tx, err := badge.NewIRTransmitter()
	if err != nil {
		println("ir tx init:", err.Error())
		select {}
	}
	a.tx = tx
	badge.IR_DATA.Configure(machine.PinConfig{Mode: machine.PinInputPullup})

	a.remotes = loadRemotes()
	println("loaded", len(a.remotes), "remotes")
	a.draw()

	for {
		ev := in.poll()
		changed := false
		switch a.screen {
		case screenList:
			changed = a.handleList(ev)
		case screenMenu:
			changed = a.handleMenu(ev)
		case screenLearn:
			changed = a.handleLearn(ev)
		case screenName:
			changed = a.handleName(ev)
		}
		if a.message != "" && time.Now().After(a.msgUntil) {
			a.message = ""
			changed = true
		}
		if changed {
			a.draw()
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func (a *app) show(msg string) {
	a.message = msg
	a.msgUntil = time.Now().Add(1500 * time.Millisecond)
}

// ---- 一覧 ----

func (a *app) handleList(ev input) bool {
	items := len(a.remotes) + 1 // 末尾は "+ Register"
	switch {
	case ev.up:
		if a.cursor > 0 {
			a.cursor--
		}
	case ev.down:
		if a.cursor < items-1 {
			a.cursor++
		}
	case ev.sw1:
		if a.cursor == len(a.remotes) {
			a.screen = screenLearn
			a.learned = nil
			startCapture()
			return true
		}
		r := a.remotes[a.cursor]
		println("send:", r.Name, len(r.Pulses), "pulses")
		a.tx.SendRaw(r.Pulses)
		a.show("Sent: " + r.Name)
	case ev.sw2:
		if a.cursor < len(a.remotes) {
			a.screen = screenMenu
			a.menuIdx = 0
		}
	default:
		return false
	}
	return true
}

// ---- メニュー ----

var menuItems = []string{"Delete", "Rename", "Back"}

func (a *app) handleMenu(ev input) bool {
	switch {
	case ev.up:
		if a.menuIdx > 0 {
			a.menuIdx--
		}
	case ev.down:
		if a.menuIdx < len(menuItems)-1 {
			a.menuIdx++
		}
	case ev.sw2:
		a.screen = screenList
	case ev.sw1:
		switch a.menuIdx {
		case 0: // Delete
			name := a.remotes[a.cursor].Name
			a.remotes = append(a.remotes[:a.cursor], a.remotes[a.cursor+1:]...)
			if a.cursor > len(a.remotes) {
				a.cursor = len(a.remotes)
			}
			a.save()
			a.show("Deleted: " + name)
			a.screen = screenList
		case 1: // Rename
			a.startNameEdit(a.remotes[a.cursor].Name, false)
		default:
			a.screen = screenList
		}
	default:
		return false
	}
	return true
}

// ---- 登録 (受信待ち) ----

func (a *app) handleLearn(ev input) bool {
	changed := false
	if p := captured(); p != nil {
		a.learned = p
		println("captured", len(p), "pulses")
		changed = true
	}
	switch {
	case ev.sw2:
		stopCapture()
		a.learned = nil
		a.screen = screenList
		return true
	case ev.sw1:
		if a.learned == nil {
			return changed
		}
		stopCapture()
		a.startNameEdit("", true)
		return true
	}
	return changed
}

// ---- 名前入力 ----

func (a *app) startNameEdit(initial string, isNew bool) {
	for i := range a.nameBuf {
		a.nameBuf[i] = ' '
	}
	copy(a.nameBuf[:], initial)
	a.nameCur = 0
	a.nameIsNew = isNew
	a.screen = screenName
}

func (a *app) handleName(ev input) bool {
	switch {
	case ev.left:
		if a.nameCur > 0 {
			a.nameCur--
		}
	case ev.right:
		if a.nameCur < NameLen-1 {
			a.nameCur++
		}
	case ev.up, ev.down:
		idx := indexOf(nameChars, a.nameBuf[a.nameCur])
		if ev.up {
			idx = (idx + 1) % len(nameChars)
		} else {
			idx = (idx + len(nameChars) - 1) % len(nameChars)
		}
		a.nameBuf[a.nameCur] = nameChars[idx]
	case ev.sw2:
		a.learned = nil
		a.screen = screenList
	case ev.sw1:
		name := trimRight(a.nameBuf[:])
		if name == "" {
			name = "REMOTE"
		}
		if a.nameIsNew {
			if len(a.remotes) >= MaxRemotes {
				a.show("Full (max 24)")
			} else {
				a.remotes = append(a.remotes, Remote{Name: name, Pulses: a.learned})
				a.cursor = len(a.remotes) - 1
				a.show("Registered: " + name)
			}
			a.learned = nil
		} else {
			a.remotes[a.cursor].Name = name
			a.show("Renamed: " + name)
		}
		a.save()
		a.screen = screenList
	default:
		return false
	}
	return true
}

func (a *app) save() {
	if err := saveRemotes(a.remotes); err != nil {
		println("save error:", err.Error())
		a.show(err.Error())
	}
}

// ---- 描画 ----

const (
	lineH     = 18
	listTop   = 40
	listLines = 9
)

func (a *app) draw() {
	fb := a.fb
	w, h := fb.Size()
	fb.FillScreen(black)
	tinyfont.WriteLine(fb, &freesans.Bold12pt7b, 8, 24, "IR Remote", white)

	switch a.screen {
	case screenList, screenMenu:
		a.drawList()
		if a.screen == screenMenu {
			a.drawMenu()
		}
		tinyfont.WriteLine(fb, &freesans.Regular9pt7b, 8, h-8, "SW1: send   SW2: menu", gray)
	case screenLearn:
		tinyfont.WriteLine(fb, &freesans.Regular9pt7b, 8, 60, "Point the remote at the", white)
		tinyfont.WriteLine(fb, &freesans.Regular9pt7b, 8, 80, "sensor and press a key.", white)
		if a.learned == nil {
			tinyfont.WriteLine(fb, &freesans.Bold12pt7b, 8, 130, "waiting...", yellow)
		} else {
			tinyfont.WriteLine(fb, &freesans.Bold12pt7b, 8, 130, "captured "+itoa(len(a.learned))+" pulses", green)
			tinyfont.WriteLine(fb, &freesans.Regular9pt7b, 8, 155, "press again to re-capture", gray)
		}
		tinyfont.WriteLine(fb, &freesans.Regular9pt7b, 8, h-8, "SW1: OK   SW2: cancel", gray)
	case screenName:
		title := "Name (new)"
		if !a.nameIsNew {
			title = "Rename"
		}
		tinyfont.WriteLine(fb, &freesans.Regular9pt7b, 8, 60, title, white)
		// 1 文字ずつ等間隔に描き、カーソル位置に下線
		const cw = 17
		x0 := int16(10)
		for i, c := range a.nameBuf {
			x := x0 + int16(i)*cw
			col := white
			if i == a.nameCur {
				fb.FillRectangle(x-2, 100, cw-2, 3, cyan)
				col = cyan
			}
			tinyfont.WriteLine(fb, &freesans.Bold12pt7b, x, 95, string(c), col)
		}
		tinyfont.WriteLine(fb, &freesans.Regular9pt7b, 8, 140, "left/right: cursor", gray)
		tinyfont.WriteLine(fb, &freesans.Regular9pt7b, 8, 160, "up/down: character", gray)
		tinyfont.WriteLine(fb, &freesans.Regular9pt7b, 8, h-8, "SW1: OK   SW2: cancel", gray)
	}

	if a.message != "" {
		fb.FillRectangle(0, h-52, w, 24, dark)
		tinyfont.WriteLine(fb, &freesans.Regular9pt7b, 8, h-36, a.message, yellow)
	}
	fb.Display()
}

func (a *app) drawList() {
	fb := a.fb
	w, _ := fb.Size()
	items := len(a.remotes) + 1
	// カーソルが見えるようにスクロール
	if a.cursor < a.scroll {
		a.scroll = a.cursor
	}
	if a.cursor >= a.scroll+listLines {
		a.scroll = a.cursor - listLines + 1
	}
	for row := 0; row < listLines; row++ {
		i := a.scroll + row
		if i >= items {
			break
		}
		y := int16(listTop + row*lineH)
		label := "+ Register"
		col := cyan
		if i < len(a.remotes) {
			label = a.remotes[i].Name
			col = white
		}
		if i == a.cursor {
			fb.FillRectangle(4, y, w-8, lineH, dark)
			tinyfont.WriteLine(fb, &freesans.Regular9pt7b, 8, y+13, ">", yellow)
		}
		tinyfont.WriteLine(fb, &freesans.Regular9pt7b, 22, y+13, label, col)
	}
	if items > listLines {
		tinyfont.WriteLine(fb, &freesans.Regular9pt7b, w-40, listTop+13, itoa(a.cursor+1)+"/"+itoa(items), gray)
	}
}

func (a *app) drawMenu() {
	fb := a.fb
	w, _ := fb.Size()
	const mw, mh = 120, 3*lineH + 16
	x := (w - mw) / 2
	y := int16(90)
	fb.FillRectangle(x-2, y-2, mw+4, mh+4, gray)
	fb.FillRectangle(x, y, mw, mh, black)
	for i, item := range menuItems {
		yy := y + 8 + int16(i)*lineH
		col := white
		if i == a.menuIdx {
			fb.FillRectangle(x+4, yy, mw-8, lineH, dark)
			col = yellow
		}
		tinyfont.WriteLine(fb, &freesans.Regular9pt7b, x+12, yy+13, item, col)
	}
}

// ---- 入力 ----

// input は 1 回のポーリングで検出したイベント (押した瞬間、または上下左右の 1 ステップ)。
type input struct {
	sw1, sw2              bool
	up, down, left, right bool
}

type inputState struct {
	sw1, sw2   badge.Button
	joy        *badge.Joystick
	prev1      bool
	prev2      bool
	dirHeld    int // 0 なし、1 上、2 下、3 左、4 右
	nextRepeat time.Time
}

func newInput() *inputState {
	sw1, sw2 := badge.NewButtons()
	joy := badge.NewJoystick()
	joy.Calibrate(300 * time.Millisecond)
	return &inputState{sw1: sw1, sw2: sw2, joy: joy}
}

const (
	dirThreshold = 500
	repeatFirst  = 400 * time.Millisecond
	repeatNext   = 120 * time.Millisecond
)

func (s *inputState) poll() input {
	var ev input
	p1, p2 := s.sw1.Pressed(), s.sw2.Pressed()
	ev.sw1 = p1 && !s.prev1
	ev.sw2 = p2 && !s.prev2
	s.prev1, s.prev2 = p1, p2

	x, y, _ := s.joy.Read()
	dir := 0
	switch {
	case y > dirThreshold:
		dir = 1
	case y < -dirThreshold:
		dir = 2
	case x < -dirThreshold:
		dir = 3
	case x > dirThreshold:
		dir = 4
	}
	fire := false
	if dir != s.dirHeld {
		s.dirHeld = dir
		if dir != 0 {
			fire = true
			s.nextRepeat = time.Now().Add(repeatFirst)
		}
	} else if dir != 0 && time.Now().After(s.nextRepeat) {
		fire = true
		s.nextRepeat = time.Now().Add(repeatNext)
	}
	if fire {
		switch dir {
		case 1:
			ev.up = true
		case 2:
			ev.down = true
		case 3:
			ev.left = true
		case 4:
			ev.right = true
		}
	}
	return ev
}

// ---- 小物 ----

func indexOf(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return 0
}

func trimRight(b []byte) string {
	n := len(b)
	for n > 0 && (b[n-1] == ' ' || b[n-1] == 0) {
		n--
	}
	return string(b[:n])
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [12]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
