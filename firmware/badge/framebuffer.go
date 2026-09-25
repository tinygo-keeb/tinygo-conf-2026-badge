package badge

import (
	"image/color"
	"runtime"

	"tinygo.org/x/drivers/pixel"
	"tinygo.org/x/drivers/st7789"
)

// Framebuffer は LCD 1 画面分のオフスクリーンバッファ (RGB565、240x240 で約 115KB)。
// 描画はすべてメモリ上で行い、Display で 1 回の SPI 転送にまとめて送る。
// ディスプレイに直接描くと部分ごとの書き換えが見えてちらつくが、
// これを使うと画面全体が一度に更新される。
//
// tinyfont や tinydraw の描画先 (drivers.Displayer) としてそのまま渡せる。
type Framebuffer struct {
	img     pixel.Image[pixel.RGB565BE]
	display *st7789.Device
	width   int16
	height  int16
}

// NewFramebuffer は display と同じ大きさのフレームバッファを確保して返す。
func NewFramebuffer(display *st7789.Device) *Framebuffer {
	w, h := display.Size()
	return &Framebuffer{
		img:     pixel.NewImage[pixel.RGB565BE](int(w), int(h)),
		display: display,
		width:   w,
		height:  h,
	}
}

// Size は画面の大きさを返す。
func (f *Framebuffer) Size() (int16, int16) {
	return f.width, f.height
}

// SetPixel は 1 ピクセルを描く。範囲外は無視する。
func (f *Framebuffer) SetPixel(x, y int16, c color.RGBA) {
	if x < 0 || y < 0 || x >= f.width || y >= f.height {
		return
	}
	f.img.Set(int(x), int(y), pixel.NewColor[pixel.RGB565BE](c.R, c.G, c.B))
}

// FillRectangle は矩形を塗りつぶす。画面からはみ出す部分は切り捨てる。
func (f *Framebuffer) FillRectangle(x, y, width, height int16, c color.RGBA) {
	x0, y0, x1, y1 := x, y, x+width, y+height
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x1 > f.width {
		x1 = f.width
	}
	if y1 > f.height {
		y1 = f.height
	}
	if x0 >= x1 || y0 >= y1 {
		return
	}
	pc := pixel.NewColor[pixel.RGB565BE](c.R, c.G, c.B)
	for yy := int(y0); yy < int(y1); yy++ {
		for xx := int(x0); xx < int(x1); xx++ {
			f.img.Set(xx, yy, pc)
		}
	}
}

// FillScreen は画面全体を塗りつぶす。
func (f *Framebuffer) FillScreen(c color.RGBA) {
	f.img.FillSolidColor(pixel.NewColor[pixel.RGB565BE](c.R, c.G, c.B))
}

// DisplayChunkRows は Display が 1 回の SPI 転送で送る行数。
// 画面全体 (約 115KB) を一度に送ると 20ms 以上ほかの goroutine が動けず、
// espradio の BLE/Wi-Fi のタスク (goroutine として動く) が止まってリンクを
// 落とすことがあるため、この行数ごとに runtime.Gosched で他へ譲る。
const DisplayChunkRows = 16

// Display はバッファの内容を LCD に転送する。DisplayChunkRows 行ごとに
// 分けて送り、合間に他の goroutine へ実行を譲る。
func (f *Framebuffer) Display() error {
	raw := f.img.RawBuffer()
	rowBytes := int(f.width) * 2
	for y := int16(0); y < f.height; y += DisplayChunkRows {
		rows := int16(DisplayChunkRows)
		if y+rows > f.height {
			rows = f.height - y
		}
		start := int(y) * rowBytes
		end := start + int(rows)*rowBytes
		if err := f.display.DrawRGBBitmap8(0, y, raw[start:end], f.width, rows); err != nil {
			return err
		}
		runtime.Gosched()
	}
	return nil
}
