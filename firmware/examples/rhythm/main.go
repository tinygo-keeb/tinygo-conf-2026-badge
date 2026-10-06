// 音ゲー。上から降ってくるノーツが判定ラインに重なるタイミングで、
// 対応するボタンを押す。
//
//	レーン (左から)  L    D/U    R    2    1
//	                ←   ↓か↑   →   SW2  SW1
//
// ジョイスティックの上と下は同じレーン (どちらに倒してもよい)。
// メロディの音の高さ (ド レ ミ ソ ラ) とレーンが対応しているので、
// 曲を聞いていると次に来るレーンが分かる。
//
//   - タイトル画面で SW1 を押すと EASY、SW2 を押すと NORMAL で始まる
//   - 判定は PERFECT (±50ms)、GOOD (±110ms)、それより遅れると MISS。
//     ジョイスティックのレーンは倒すのに時間がかかるので、それぞれ 30ms 広い
//   - ヒットすると LED がレーンの色に光る
//   - GOOD 以上が続く (コンボ) と、コンボ数に応じて画面の演出が増える (effects.go)
//   - 曲が終わると結果 (判定ごとの数、最大コンボ、ランク) を表示する
//
// 起動直後にジョイスティックのセンター位置を測るので、そのあいだは触らないこと。
//
// 音は I2S のリングバッファに空きができるたびに audioLoop (goroutine) が補充し、
// メインループは描画の合間に処理を譲る (Framebuffer.Display と time.Sleep)。
// ゲームの時刻はスピーカーから音が出ている位置 (synth.clock) に合わせているので、
// 描画が遅れても曲と譜面はずれない。
// 入力は pollLoop (goroutine) が 2ms ごとに読んで押した時刻を記録する。
// 描画のフレーム (33ms) 単位で読むと判定が最大 1 フレーム遅れるため。
package main

import (
	"image/color"
	"time"

	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/freesans"

	"github.com/sago35/tinygo-conf-2026-badge/firmware/badge"
	"github.com/sago35/tinygo-conf-2026-badge/firmware/i2s"
	"github.com/sago35/tinygo-conf-2026-badge/firmware/ws2812s3"
)

var (
	black    = color.RGBA{0, 0, 0, 255}
	white    = color.RGBA{255, 255, 255, 255}
	gray     = color.RGBA{48, 48, 48, 255}
	darkGray = color.RGBA{24, 24, 24, 255}
	red      = color.RGBA{255, 40, 40, 255}
	yellow   = color.RGBA{255, 220, 0, 255}
	cyan     = color.RGBA{0, 255, 255, 255}
	green    = color.RGBA{0, 255, 0, 255}
)

// レーンの色と、画面下に出すラベル。
var laneColors = [laneCount]color.RGBA{
	{255, 64, 160, 255},  // ←
	{64, 160, 255, 255},  // ↓↑
	{255, 160, 32, 255},  // →
	{200, 120, 255, 255}, // SW2
	{255, 255, 64, 255},  // SW1
}

var laneLabels = [laneCount]string{"L", "D/U", "R", "2", "1"}

var (
	font      = &freesans.Regular9pt7b
	fontBold  = &freesans.Bold12pt7b
	fontLarge = &freesans.Bold18pt7b
)

// 画面のレイアウト。
const (
	laneW     = badge.LCD_WIDTH / laneCount // 48
	judgeY    = 200                         // 判定ラインの y
	noteH     = 10
	topMargin = 24 // スコア表示の下からノーツが現れる
)

// 判定の幅。
const (
	perfectWindow = 50 * time.Millisecond
	goodWindow    = 110 * time.Millisecond
)

// fallTime はノーツが画面上端から判定ラインまで落ちてくる時間 (難易度別)。
var fallTime = [...]time.Duration{easy: 1600 * time.Millisecond, normal: 1200 * time.Millisecond}

// ジョイスティックを倒したとみなす量 (-1000..1000 のうち)。
// 浅めにして、倒し始めてから判定されるまでの遅れを小さくしている。
// 戻すときは joyRelease を下回ったら離したとみなす (境目でのばたつき防止)。
const (
	joyPress   = 300
	joyRelease = 250
)

// ジョイスティックのレーンは、倒すのにボタンより時間がかかるぶん判定の幅を広げる。
const joyWindowBonus = 30 * time.Millisecond

// joyLatency はジョイスティックの押下時刻から差し引く遅れ。結果を出すときに
// シリアルに出る「timing」の stick と button の差を見て調整する (正なら遅れ)。
const joyLatency = 0 * time.Millisecond

// 判定の種類。表示とカウントに使う。
const (
	judgeNone = iota
	judgePerfect
	judgeGood
	judgeMiss
)

var judgeText = [...]string{"", "PERFECT", "GOOD", "MISS"}
var judgeColor = [...]color.RGBA{black, cyan, green, red}

// screen は描画先のフレームバッファ。
var screen *badge.Framebuffer

type game struct {
	leds ledFlash
	syn  *synth
	joy  *badge.Joystick
	sw1  badge.Button
	sw2  badge.Button

	audio bool // 音が出せるか (I2S の初期化に成功したか)

	// pollLoop が更新する入力の状態
	held   [laneCount]bool // 押している間 true
	events []press         // 押した瞬間の記録 (メインループが takeEvents で取り出す)

	best [2]int // 難易度ごとの最高スコア (電源を切ると消える)
}

func main() {
	// ジョイスティックのセンター位置は起動直後に測る (触らないこと)
	joy := badge.NewJoystick()
	joy.Calibrate(300 * time.Millisecond)

	screen = badge.NewFramebuffer(badge.NewDisplay())
	g := &game{
		syn:  newSynth(),
		joy:  joy,
		leds: newLEDFlash(),
	}
	g.sw1, g.sw2 = badge.NewButtons()

	dev := i2s.New()
	err := dev.Configure(i2s.Config{
		BCLK:         badge.I2S_BCLK,
		LRC:          badge.I2S_LRC,
		DIN:          badge.I2S_DIN,
		SampleRate:   sampleRate,
		BufferFrames: bufFrames,
		BufferCount:  bufCount,
	})
	if err != nil {
		println("audio init:", err.Error())
	} else {
		g.audio = true
		go audioLoop(dev, g.syn)
	}
	g.events = make([]press, 0, maxEvents)
	go g.pollLoop()

	for {
		d := g.title()
		g.result(d, g.play(d))
	}
}

// input は 6 レーンの押下状態を返す。
func (g *game) input() (pressed [laneCount]bool) {
	x, y, _ := g.joy.Read()
	pressed[laneLeft] = tilted(-x, g.held[laneLeft])
	pressed[laneUpDown] = tilted(y, g.held[laneUpDown]) || tilted(-y, g.held[laneUpDown])
	pressed[laneRight] = tilted(x, g.held[laneRight])
	pressed[laneSW1] = g.sw1.Pressed()
	pressed[laneSW2] = g.sw2.Pressed()
	return pressed
}

// tilted は軸の値 v が倒した状態か返す。held (直前に倒していた) ならしきい値を下げる。
func tilted(v int, held bool) bool {
	if held {
		return v > joyRelease
	}
	return v > joyPress
}

// isStick はジョイスティックのレーンなら true。
func isStick(lane int) bool {
	return lane == laneLeft || lane == laneUpDown || lane == laneRight
}

// press はレーンを押した瞬間の記録。
type press struct {
	lane int
	at   time.Time
}

const maxEvents = 32

// pollLoop は 2ms ごとに入力を読み、押した瞬間をその時刻とともに記録する。
func (g *game) pollLoop() {
	for {
		p := g.input()
		now := time.Now()
		for i := range p {
			if p[i] && !g.held[i] && len(g.events) < maxEvents {
				g.events = append(g.events, press{lane: i, at: now})
			}
		}
		g.held = p
		time.Sleep(2 * time.Millisecond)
	}
}

// takeEvents は記録された押下を dst にコピーして返し、記録を空にする。
func (g *game) takeEvents(dst []press) []press {
	dst = append(dst[:0], g.events...)
	g.events = g.events[:0]
	return dst
}

// pressed は記録された押下の中に lane があれば true (タイトルと結果画面用)。
func pressed(evs []press, lane int) bool {
	for _, e := range evs {
		if e.lane == lane {
			return true
		}
	}
	return false
}

// frame は約 30fps になるように待つ。待つあいだは audioLoop が動く。
func frame(start time.Time) {
	if d := 33*time.Millisecond - time.Since(start); d > 0 {
		time.Sleep(d)
	} else {
		time.Sleep(time.Millisecond)
	}
}

// ---- タイトル ----

func (g *game) title() difficulty {
	var evs []press
	g.takeEvents(nil) // 前の画面で押したぶんは捨てる
	for {
		start := time.Now()
		evs = g.takeEvents(evs)
		if pressed(evs, laneSW1) {
			g.syn.beep(1047)
			return easy
		}
		if pressed(evs, laneSW2) {
			g.syn.beep(1319)
			return normal
		}

		fb := screen
		fb.FillScreen(black)
		centerText(fontLarge, 60, "TinyGo", white)
		centerText(fontLarge, 96, "BEAT", yellow)
		// レーンの色見本
		for i := 0; i < laneCount; i++ {
			x := int16(i*laneW) + 4
			fb.FillRectangle(x, 116, laneW-8, noteH, laneColors[i])
			centerTextAt(font, int16(i*laneW), laneW, 144, laneLabels[i], laneColors[i])
		}
		centerText(font, 180, "SW1: EASY   best "+itoa(g.best[easy]), white)
		centerText(font, 204, "SW2: NORMAL best "+itoa(g.best[normal]), white)
		centerText(font, 230, "#"+badge.SerialNumber(), gray)
		fb.Display()
		g.leds.update()
		frame(start)
	}
}

// ---- プレイ ----

type stats struct {
	counts    [4]int // judgePerfect, judgeGood, judgeMiss
	total     int
	score     int
	combo     int
	maxCombo  int
	lastJudge int
	judgeAt   time.Duration

	// ヒットのずれ (押した時刻 - ノーツの時刻) の合計と数。
	// [0] がジョイスティック、[1] がボタン。調整用にシリアルへ出す。
	offsetSum [2]time.Duration
	offsetN   [2]int
}

// timing はヒットしたときのずれを記録する。
func (s *stats) timing(lane int, dt time.Duration) {
	k := 1
	if isStick(lane) {
		k = 0
	}
	s.offsetSum[k] += dt
	s.offsetN[k]++
}

// avgOffsetMs はずれの平均 (ms)。正なら遅れ、負なら早い。
func (s *stats) avgOffsetMs(k int) int {
	if s.offsetN[k] == 0 {
		return 0
	}
	return int(s.offsetSum[k] / time.Duration(s.offsetN[k]) / time.Millisecond)
}

func (s *stats) add(j int, at time.Duration) {
	s.counts[j]++
	s.lastJudge = j
	s.judgeAt = at
	switch j {
	case judgePerfect:
		s.combo++
		s.score += 100 + s.combo
	case judgeGood:
		s.combo++
		s.score += 50 + s.combo/2
	case judgeMiss:
		s.combo = 0
	}
	if s.combo > s.maxCombo {
		s.maxCombo = s.combo
	}
}

func (g *game) play(d difficulty) stats {
	steps, notes := buildSong(d)
	end := songLength(steps)
	fall := fallTime[d]
	st := stats{total: len(notes)}
	var flashAt [laneCount]time.Duration // レーンごとのヒットの時刻 (エフェクト用)
	for i := range flashAt {
		flashAt[i] = -time.Hour
	}

	if g.audio {
		g.syn.start(steps)
		for !g.syn.started() {
			time.Sleep(time.Millisecond) // audioLoop が開始時刻を決めるのを待つ
		}
	} else {
		g.syn.startSilent() // 音が出ない場合も時刻だけは進める
	}
	fx := newEffects()
	var evs []press
	g.takeEvents(nil)
	head := 0 // これより前のノーツは判定済み

	for {
		start := time.Now()
		now := g.syn.clock()
		if now > end {
			break
		}
		evs = g.takeEvents(evs)

		// 押したレーンで、判定の幅に入っている一番早いノーツを取る。
		// 判定には押した瞬間の時刻を使う
		for _, ev := range evs {
			lane := ev.lane
			at := g.syn.at(ev.at)
			good, perfect := goodWindow, perfectWindow
			if isStick(lane) {
				at -= joyLatency
				good += joyWindowBonus
				perfect += joyWindowBonus
			}
			for i := head; i < len(notes) && notes[i].t <= at+good; i++ {
				n := &notes[i]
				if n.lane != lane || n.state != notePending {
					continue
				}
				dt := at - n.t
				signed := dt
				if dt < 0 {
					dt = -dt
				}
				if dt > good {
					continue
				}
				n.state = noteHit
				st.timing(lane, signed)
				j := judgeGood
				if dt <= perfect {
					j = judgePerfect
				}
				st.add(j, now)
				fx.hit(lane, st.combo, now)
				flashAt[lane] = now
				g.leds.flash(laneColors[lane])
				break
			}
		}

		// 判定ラインを過ぎたノーツは MISS
		for i := head; i < len(notes) && notes[i].t < now-goodWindow; i++ {
			if notes[i].state == notePending {
				notes[i].state = noteMissed
				prev := st.combo
				st.add(judgeMiss, now)
				fx.miss(prev, now)
			}
		}
		for head < len(notes) && notes[head].state != notePending {
			head++
		}

		fx.update()
		g.drawPlay(notes, head, now, fall, g.held, flashAt, &st, fx)
		g.leds.update()
		frame(start)
	}
	g.syn.stop()
	if st.score > g.best[d] {
		g.best[d] = st.score
	}
	return st
}

func (g *game) drawPlay(notes []note, head int, now, fall time.Duration, held [laneCount]bool, flashAt [laneCount]time.Duration, st *stats, fx *effects) {
	fb := screen
	fb.FillScreen(black)
	fx.drawBack(fb, now, st.combo)

	// レーンの背景。押しているレーンは明るく、ヒット直後はレーンの色で光らせる
	for i := 0; i < laneCount; i++ {
		x := int16(i * laneW)
		if held[i] {
			fb.FillRectangle(x, topMargin, laneW, judgeY-topMargin, darkGray)
		}
		if now-flashAt[i] < 120*time.Millisecond {
			fb.FillRectangle(x, judgeY-30, laneW, 30, dim(laneColors[i]))
		}
		fb.FillRectangle(x, topMargin, 1, badge.LCD_HEIGHT-topMargin, gray)
	}
	fx.judgeLine(fb, now, st.combo)

	// ノーツ。判定ラインに来る時刻から位置を決める
	for i := head; i < len(notes) && notes[i].t < now+fall; i++ {
		n := notes[i]
		if n.state != notePending {
			continue
		}
		y := judgeY - int16(int64(n.t-now)*int64(judgeY-topMargin)/int64(fall)) - noteH/2
		if y+noteH < topMargin {
			continue
		}
		fb.FillRectangle(int16(n.lane*laneW)+3, y, laneW-6, noteH, laneColors[n.lane])
	}
	fx.drawFront(fb, now, st.combo)

	// レーンのラベル
	for i := 0; i < laneCount; i++ {
		c := laneColors[i]
		if !held[i] {
			c = dim(c)
		}
		centerTextAt(font, int16(i*laneW), laneW, 226, laneLabels[i], c)
	}

	// スコアとコンボ
	fb.FillRectangle(0, 0, badge.LCD_WIDTH, topMargin, black)
	tinyfont.WriteLine(fb, font, 4, 18, itoa(st.score), white)
	if st.combo >= 2 {
		s := itoa(st.combo) + " combo"
		_, w := tinyfont.LineWidth(font, s)
		c := yellow
		if st.combo >= comboTier2 {
			c = hue(int(now/time.Millisecond) % 1536)
		}
		tinyfont.WriteLine(fb, font, badge.LCD_WIDTH-4-int16(w), 18, s, c)
	}
	if len(notes) > 0 && now < notes[0].t-fall {
		// 前奏のあいだ (最初のノーツが見えるまで) は案内を出す
		centerText(fontBold, 110, "READY", white)
	}

	// 判定の表示 (0.4 秒)
	if st.lastJudge != judgeNone && now-st.judgeAt < 400*time.Millisecond {
		centerText(fontBold, 120, judgeText[st.lastJudge], judgeColor[st.lastJudge])
	}
	fb.Display()
}

// ---- 結果 ----

func (g *game) result(d difficulty, st stats) {
	rate := 0
	if st.total > 0 {
		rate = (st.counts[judgePerfect]*100 + st.counts[judgeGood]*50) / st.total
	}
	rank, rankColor := "C", white
	switch {
	case rate >= 95:
		rank, rankColor = "S", yellow
	case rate >= 85:
		rank, rankColor = "A", cyan
	case rate >= 70:
		rank, rankColor = "B", green
	}
	fc := st.counts[judgeMiss] == 0
	println("result:", []string{"EASY", "NORMAL"}[d], "score", st.score,
		"perfect", st.counts[judgePerfect], "good", st.counts[judgeGood], "miss", st.counts[judgeMiss],
		"maxcombo", st.maxCombo, "rate", rate, "rank", rank)
	println("timing (ms, + is late): stick", st.avgOffsetMs(0), "n", st.offsetN[0],
		"button", st.avgOffsetMs(1), "n", st.offsetN[1])

	shown := time.Now()
	var evs []press
	for {
		start := time.Now()
		evs = g.takeEvents(evs)
		// 押しっぱなしのまま次に進まないよう、少し待ってから受け付ける
		if time.Since(shown) > time.Second && (pressed(evs, laneSW1) || pressed(evs, laneSW2)) {
			g.syn.beep(1047)
			return
		}

		fb := screen
		fb.FillScreen(black)
		centerText(fontBold, 28, "RESULT", white)
		centerText(fontLarge, 80, rank, rankColor)
		y := int16(112)
		row := func(label string, v int, c color.RGBA) {
			tinyfont.WriteLine(fb, font, 40, y, label, c)
			s := itoa(v)
			_, w := tinyfont.LineWidth(font, s)
			tinyfont.WriteLine(fb, font, 200-int16(w), y, s, c)
			y += 20
		}
		row("PERFECT", st.counts[judgePerfect], cyan)
		row("GOOD", st.counts[judgeGood], green)
		row("MISS", st.counts[judgeMiss], red)
		row("MAX COMBO", st.maxCombo, yellow)
		row("SCORE", st.score, white)
		if fc {
			centerText(fontBold, 232, "FULL COMBO!", yellow)
		} else {
			centerText(font, 230, "SW1/SW2: title", gray)
		}
		fb.Display()
		g.leds.update()
		frame(start)
	}
}

// ---- LED ----

// ledFlash はヒットしたときに LED をレーンの色で光らせ、徐々に消す。
type ledFlash struct {
	dev   ws2812s3.Device
	c     color.RGBA
	level int // 0..255
	buf   [badge.WS2812_COUNT]color.RGBA
}

func newLEDFlash() ledFlash {
	leds := badge.NewLEDs()
	leds.SetBrightness(48)
	leds.WriteColors(make([]color.RGBA, badge.WS2812_COUNT))
	return ledFlash{dev: leds}
}

func (l *ledFlash) flash(c color.RGBA) {
	l.c = c
	l.level = 255
}

// update は 1 フレームごとに呼ぶ。消えている間は何も送らない。
func (l *ledFlash) update() {
	if l.level <= 0 {
		return
	}
	l.level -= 40
	if l.level < 0 {
		l.level = 0
	}
	for i := range l.buf {
		l.buf[i] = scale(l.c, l.level)
	}
	l.dev.WriteColors(l.buf[:])
}

// ---- 描画の補助 ----

func centerText(f tinyfont.Fonter, y int16, s string, c color.RGBA) {
	centerTextAt(f, 0, badge.LCD_WIDTH, y, s, c)
}

// centerTextAt は x から幅 w の範囲の中央に文字列を描く。
func centerTextAt(f tinyfont.Fonter, x, w, y int16, s string, c color.RGBA) {
	_, tw := tinyfont.LineWidth(f, s)
	tinyfont.WriteLine(screen, f, x+(w-int16(tw))/2, y, s, c)
}

func dim(c color.RGBA) color.RGBA {
	return scale(c, 96)
}

func scale(c color.RGBA, level int) color.RGBA {
	return color.RGBA{
		R: uint8(int(c.R) * level / 255),
		G: uint8(int(c.G) * level / 255),
		B: uint8(int(c.B) * level / 255),
		A: 255,
	}
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
