// gensprite は PNG を図柄用の小さなスプライトに変換する（ホストの Go で実行）。
//
//	go run ./tools/gensprite -h 44 -o img/gopher_blue.rgba img/gopher_svg.png
//
// 透明部分を切り落とし、高さ -h に面積平均で縮小して書き出す。
// 出力形式は [幅, 高さ] の 2 バイトに続いて、各ピクセルの R, G, B, A（非乗算）。
package main

import (
	"flag"
	"image"
	"image/png"
	"log"
	"os"
)

func main() {
	height := flag.Int("h", 44, "output height in pixels")
	out := flag.String("o", "", "output file")
	flag.Parse()
	if flag.NArg() != 1 || *out == "" {
		log.Fatal("usage: gensprite -h 44 -o out.rgba in.png")
	}

	f, err := os.Open(flag.Arg(0))
	if err != nil {
		log.Fatal(err)
	}
	src, err := png.Decode(f)
	f.Close()
	if err != nil {
		log.Fatal(err)
	}

	bb := opaqueBounds(src)
	dh := *height
	dw := (bb.Dx()*dh + bb.Dy()/2) / bb.Dy()
	if dw > 255 || dh > 255 {
		log.Fatal("sprite too large")
	}

	buf := []byte{byte(dw), byte(dh)}
	for y := 0; y < dh; y++ {
		sy0 := bb.Min.Y + y*bb.Dy()/dh
		sy1 := bb.Min.Y + (y+1)*bb.Dy()/dh
		for x := 0; x < dw; x++ {
			sx0 := bb.Min.X + x*bb.Dx()/dw
			sx1 := bb.Min.X + (x+1)*bb.Dx()/dw
			// 乗算済みアルファで平均し、縁の色が濁らないようにする
			var r, g, b, a, n uint64
			for sy := sy0; sy < max(sy1, sy0+1); sy++ {
				for sx := sx0; sx < max(sx1, sx0+1); sx++ {
					cr, cg, cb, ca := src.At(sx, sy).RGBA()
					r, g, b, a = r+uint64(cr), g+uint64(cg), b+uint64(cb), a+uint64(ca)
					n++
				}
			}
			if a == 0 {
				buf = append(buf, 0, 0, 0, 0)
				continue
			}
			buf = append(buf,
				byte(r*255/a), byte(g*255/a), byte(b*255/a), byte(a/n>>8))
		}
	}
	if err := os.WriteFile(*out, buf, 0o644); err != nil {
		log.Fatal(err)
	}
	log.Printf("%s: %dx%d", *out, dw, dh)
}

// opaqueBounds は不透明なピクセルを囲む最小の矩形を返す。
func opaqueBounds(img image.Image) image.Rectangle {
	b := img.Bounds()
	r := image.Rectangle{Min: b.Max, Max: b.Min}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a > 0x0800 {
				r = r.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	if r.Empty() {
		return b
	}
	return r
}
