package main

// ホストの Go で動かすテスト。`go test` で役判定・停止ロジックを確認し、
// `go test -run TestPreview -preview` で画面イメージを PNG に書き出す。

import (
	"flag"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"tinygo.org/x/drivers/pixel"
)

var preview = flag.Bool("preview", false, "write preview PNGs")
var previewDir = flag.String("preview-dir", ".", "directory for preview PNGs")

// fakeScreen は液晶ドライバと同じ座標チェックをする仮想液晶（幅 w × 240）。
type fakeScreen struct {
	img  *image.RGBA
	w    int16
	errs int
}

func newFakeScreen(w int16) *fakeScreen {
	return &fakeScreen{img: image.NewRGBA(image.Rect(0, 0, int(w), screenH)), w: w}
}

func (s *fakeScreen) Size() (int16, int16) { return s.w, screenH }
func (s *fakeScreen) Display() error       { return nil }

func (s *fakeScreen) SetPixel(x, y int16, c color.RGBA) {
	if x < 0 || y < 0 || x >= s.w || y >= screenH {
		return
	}
	s.img.SetRGBA(int(x), int(y), c)
}

func (s *fakeScreen) FillRectangle(x, y, w, h int16, c color.RGBA) error {
	if x < 0 || y < 0 || w <= 0 || h <= 0 || x+w > s.w || y+h > screenH {
		s.errs++
		return os.ErrInvalid
	}
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			s.img.SetRGBA(int(xx), int(yy), c)
		}
	}
	return nil
}

func (s *fakeScreen) DrawBitmap(x, y int16, b pixel.Image[pixel.RGB565BE]) error {
	w, h := b.Size()
	if x < 0 || y < 0 || int(x)+w > int(s.w) || int(y)+h > screenH {
		s.errs++
		return os.ErrInvalid
	}
	for yy := 0; yy < h; yy++ {
		for xx := 0; xx < w; xx++ {
			s.img.SetRGBA(int(x)+xx, int(y)+yy, b.Get(xx, yy).RGBA())
		}
	}
	return nil
}

func TestJudge(t *testing.T) {
	tests := []struct {
		l, c, r uint8
		win     int
	}{
		{symSeven, symSeven, symSeven, 100},
		{symBar, symBar, symBar, 50},
		{symBell, symBell, symBell, 15},
		{symCherry, symCherry, symCherry, 10}, // 左チェリーの 2 は加算しない
		{symGrape, symGrape, symGrape, 8},
		{symCherry, symBell, symGrape, 2},
		{symGrape, symCherry, symCherry, 0},
		{symSeven, symSeven, symBar, 0},
	}
	for _, tt := range tests {
		if win, _ := judge(tt.l, tt.c, tt.r); win != tt.win {
			t.Errorf("judge(%d,%d,%d) = %d, want %d", tt.l, tt.c, tt.r, win, tt.win)
		}
	}
}

func TestInitialCenter(t *testing.T) {
	want := [3]uint8{symGrape, symBell, symGrape}
	for i := range strips {
		r := reel{strip: strips[i]}
		if got := r.center(); got != want[i] {
			t.Errorf("reel %d center = %d, want %d", i, got, want[i])
		}
	}
}

// どのタイミングで止めても、コマ境界に揃って最大1コマ（5フレーム）以内に止まる。
func TestStopAlignment(t *testing.T) {
	for frames := 0; frames < 200; frames++ {
		r := reel{strip: strips[0]}
		r.start()
		for i := 0; i < frames; i++ {
			r.update()
		}
		before := r.offset
		r.requestStop()
		n := 0
		for r.update() {
			n++
		}
		if r.offset%symH != 0 {
			t.Fatalf("frames=%d: offset %d not aligned", frames, r.offset)
		}
		if n > 5 {
			t.Fatalf("frames=%d: took %d frames to stop", frames, n)
		}
		if d := (r.offset - before + r.total()) % r.total(); d >= symH {
			t.Fatalf("frames=%d: slid %d px", frames, d)
		}
	}
}

// 1周の間のどこかで押せば 7 を中段に止められる（目押しできる）。
// 7 を狙える押下タイミングは 1周あたり 4 フレーム以上ある。
func TestAimSeven(t *testing.T) {
	for i := range strips {
		ok := 0
		probe := reel{strip: strips[i]}
		for frames := 1; frames <= probe.total()/spinSpeed; frames++ {
			r := reel{strip: strips[i]}
			r.start()
			for f := 0; f < frames; f++ {
				r.update()
			}
			r.requestStop()
			for r.update() {
			}
			if r.center() == symSeven {
				ok++
			}
		}
		if ok < 4 {
			t.Errorf("reel %d: only %d frames hit seven", i, ok)
		}
	}
}

// 押しっぱなしは最初のフレームだけ入力になる。
func TestEdges(t *testing.T) {
	var prev [3]bool
	if p := edges([3]bool{true, false, false}, &prev); p != [3]bool{true, false, false} {
		t.Fatalf("first press: %v", p)
	}
	for i := 0; i < 5; i++ {
		if p := edges([3]bool{true, false, false}, &prev); p != [3]bool{} {
			t.Fatalf("held: %v", p)
		}
	}
	edges([3]bool{}, &prev)
	if p := edges([3]bool{true, false, true}, &prev); p != [3]bool{true, false, true} {
		t.Fatalf("re-press: %v", p)
	}
}

var layouts = map[string]layout{"wide": layoutWide, "square": layoutSquare}

func TestGameFlow(t *testing.T) {
	for name, lay := range layouts {
		t.Run(name, func(t *testing.T) { testGameFlow(t, lay) })
	}
}

func testGameFlow(t *testing.T, lay layout) {
	scr := newFakeScreen(lay.screenW)
	g := newGame(scr, lay)
	t0 := time.Unix(0, 0)
	now := t0
	tick := func(p [3]bool) {
		now = now.Add(frameTime)
		g.step(p, now)
	}
	all := [3]bool{true, true, true}
	none := [3]bool{}

	tick(all)
	if g.state != stateSpinning || g.credit != startCredit-bet {
		t.Fatalf("after start: state=%d credit=%d", g.state, g.credit)
	}
	// ロックアウト中は止まらない
	tick(all)
	for i := range g.reels {
		if g.reels[i].stopping {
			t.Fatal("stop accepted during lockout")
		}
	}
	for now.Sub(g.spinStart) < stopLockout {
		tick(none)
	}
	tick([3]bool{false, true, false})
	if !g.reels[1].stopping || g.reels[0].stopping || g.reels[2].stopping {
		t.Fatal("button B should stop only the middle reel")
	}
	for i := 0; i < 10; i++ {
		tick(none)
	}
	if g.reels[1].spinning || !g.reels[0].spinning {
		t.Fatal("middle reel should be stopped, left still spinning")
	}
	tick([3]bool{true, false, true})
	for g.state == stateSpinning {
		tick(none)
	}
	win, _ := judge(g.reels[0].center(), g.reels[1].center(), g.reels[2].center())
	if g.credit != startCredit-bet+win {
		t.Fatalf("credit = %d, want %d", g.credit, startCredit-bet+win)
	}

	// ハズレで残り 3 未満になれば GAME OVER、ボタンで 50 に戻る
	g.credit = bet
	tick(none)
	tick(all)
	now = now.Add(stopLockout)
	for i := range g.reels {
		g.reels[i].offset = 0 // 中段 ブドウ・ベル・ブドウ（ハズレ）
	}
	tick(all)
	if g.state != stateGameOver || g.credit != 0 {
		t.Fatalf("expected game over: state=%d credit=%d", g.state, g.credit)
	}
	tick(none)
	tick(all)
	if g.state != stateIdle || g.credit != startCredit {
		t.Fatalf("after reset: state=%d credit=%d", g.state, g.credit)
	}
	if scr.errs != 0 {
		t.Fatalf("%d draw calls outside the screen", scr.errs)
	}
}

// 1 ボタンのときは、押すたびに左のリールから順に止まる。
func TestOneButton(t *testing.T) {
	scr := newFakeScreen(layoutSquare.screenW)
	g := newGame(scr, layoutSquare)
	g.oneButton = true
	now := time.Unix(0, 0)
	var prev [3]bool
	tick := func(down [3]bool) {
		now = now.Add(frameTime)
		g.step(edges(down, &prev), now)
	}
	a := [3]bool{true}
	none := [3]bool{}

	tick(a)
	if g.state != stateSpinning || g.credit != startCredit-bet {
		t.Fatalf("after start: state=%d credit=%d", g.state, g.credit)
	}
	// 開始ボタンをロックアウト後まで押し続けてもリールは止まらない。
	for now.Sub(g.spinStart) <= stopLockout {
		tick(a)
		if g.nextReel() != 0 {
			t.Fatal("holding start stopped a reel")
		}
	}
	tick(none)
	for want := 0; want < 3; want++ {
		if g.nextReel() != want {
			t.Fatalf("next reel = %d, want %d", g.nextReel(), want)
		}
		tick(a)
		for i := range g.reels {
			if stopped := g.reels[i].stopping || !g.reels[i].spinning; stopped != (i <= want) {
				t.Fatalf("after press %d: reel %d stopped=%v", want+1, i, stopped)
			}
		}
		tick(a)
		wantNext := want + 1
		if wantNext == len(g.reels) {
			wantNext = -1
		}
		if g.nextReel() != wantNext {
			t.Fatal("holding SW1 stopped another reel")
		}
		tick(none)
	}
	for g.state == stateSpinning {
		tick(none)
	}
	win, _ := judge(g.reels[0].center(), g.reels[1].center(), g.reels[2].center())
	if g.credit != startCredit-bet+win {
		t.Fatalf("credit = %d, want %d", g.credit, startCredit-bet+win)
	}
	g.state, g.credit = stateGameOver, 0
	tick(a)
	if g.state != stateIdle || g.credit != startCredit {
		t.Fatal("SW1 did not reset credits after GAME OVER")
	}
	tick(a)
	if g.state != stateIdle {
		t.Fatal("holding restart began another game")
	}
	tick(none)
	tick(a)
	if g.state != stateSpinning || g.credit != startCredit-bet {
		t.Fatal("pressing SW1 again did not start another game")
	}
	if scr.errs != 0 {
		t.Fatalf("%d draw calls outside the screen", scr.errs)
	}
}

// 画面のはみ出しや重なりがないか確かめるため、画面ごとに PNG を書き出す。
// 320×240 は preview_*.png、240×240 は preview_square_*.png。
func TestPreview(t *testing.T) {
	if !*preview {
		t.Skip("use -preview to write PNGs")
	}
	previewLayout(t, layoutWide, "preview_")
	previewLayout(t, layoutSquare, "preview_square_")

	// 1 ボタンで左のリールを止めたところ（中が次、右は順番待ち）
	scr := newFakeScreen(layoutSquare.screenW)
	g := newGame(scr, layoutSquare)
	g.oneButton = true
	now := time.Unix(0, 0)
	g.step([3]bool{true}, now)
	now = now.Add(stopLockout)
	g.step([3]bool{true}, now)
	for i := 0; i < 6; i++ {
		now = now.Add(frameTime)
		g.step([3]bool{}, now)
	}
	writePNG(t, "preview_square_onebutton.png", scr.img)
}

func previewLayout(t *testing.T, lay layout, prefix string) {
	scr := newFakeScreen(lay.screenW)
	g := newGame(scr, lay)
	writePNG(t, prefix+"idle.png", scr.img)

	// 回転中（スタートして少し回したところ）
	now := time.Unix(0, 0)
	g.step([3]bool{true}, now)
	for i := 0; i < 7; i++ {
		now = now.Add(frameTime)
		g.step([3]bool{}, now)
	}
	writePNG(t, prefix+"spin.png", scr.img)

	// 7 を揃えたところ
	for i := range g.reels {
		g.reels[i].offset = symH
	}
	now = now.Add(stopLockout)
	g.step([3]bool{true, true, true}, now)
	for g.state == stateSpinning {
		now = now.Add(frameTime)
		g.step([3]bool{}, now)
	}
	writePNG(t, prefix+"win.png", scr.img)

	// GAME OVER
	g.credit = bet
	g.step([3]bool{true}, now)
	now = now.Add(stopLockout)
	for i := range g.reels {
		g.reels[i].offset = 0
	}
	g.step([3]bool{true, true, true}, now)
	writePNG(t, prefix+"gameover.png", scr.img)
	if scr.errs != 0 {
		t.Fatalf("%d draw calls outside the screen", scr.errs)
	}
}

func writePNG(t *testing.T, name string, img image.Image) {
	if err := os.MkdirAll(*previewDir, 0755); err != nil {
		t.Fatal(err)
	}
	name = filepath.Join(*previewDir, name)
	f, err := os.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	// 確認しやすいよう 2 倍に拡大する
	b := img.Bounds()
	big := image.NewRGBA(image.Rect(0, 0, b.Dx()*2, b.Dy()*2))
	for y := 0; y < b.Dy()*2; y++ {
		for x := 0; x < b.Dx()*2; x++ {
			big.Set(x, y, img.At(x/2, y/2))
		}
	}
	if err := png.Encode(f, big); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %s", name)
}
