package see

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"strconv"
)

var (
	annotateRed   = color.RGBA{R: 255, A: 255}
	annotateWhite = color.RGBA{R: 255, G: 255, B: 255, A: 255}
)

// 5x7 digits; '#' is a lit pixel. Scaled 2x when drawn.
var digitGlyphs = [10][7]string{
	{
		".###.",
		"#...#",
		"#...#",
		"#...#",
		"#...#",
		"#...#",
		".###.",
	},
	{
		"..#..",
		".##..",
		"..#..",
		"..#..",
		"..#..",
		"..#..",
		".###.",
	},
	{
		".###.",
		"#...#",
		"....#",
		"..##.",
		".#...",
		"#....",
		"#####",
	},
	{
		".###.",
		"#...#",
		"....#",
		"..##.",
		"....#",
		"#...#",
		".###.",
	},
	{
		"#...#",
		"#...#",
		"#...#",
		"#####",
		"....#",
		"....#",
		"....#",
	},
	{
		"#####",
		"#....",
		"#....",
		"####.",
		"....#",
		"#...#",
		".###.",
	},
	{
		".###.",
		"#....",
		"#....",
		"####.",
		"#...#",
		"#...#",
		".###.",
	},
	{
		"#####",
		"....#",
		"...#.",
		"..#..",
		".#...",
		".#...",
		".#...",
	},
	{
		".###.",
		"#...#",
		"#...#",
		".###.",
		"#...#",
		"#...#",
		".###.",
	},
	{
		".###.",
		"#...#",
		"#...#",
		".####",
		"....#",
		"....#",
		".###.",
	},
}

func readRGBA(path string) (*image.RGBA, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return nil, err
	}
	if r, ok := img.(*image.RGBA); ok {
		return r, nil
	}
	b := img.Bounds()
	dst := image.NewRGBA(b)
	draw.Draw(dst, b, img, b.Min, draw.Src)
	return dst, nil
}

func writePNG(path string, img image.Image) error {
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	err = png.Encode(f, img)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func mapCoord(v, origin, src, dst int) int {
	if src == 0 {
		return v - origin
	}
	return (v - origin) * dst / src
}

func annotate(path string, windows []Window, originX, originY, srcW, srcH int) error {
	img, err := readRGBA(path)
	if err != nil {
		return err
	}
	b := img.Bounds()
	dstW, dstH := b.Dx(), b.Dy()
	if srcW <= 0 {
		srcW = dstW
	}
	if srcH <= 0 {
		srcH = dstH
	}
	for _, w := range windows {
		if !w.Mapped || w.Size[0] <= 0 || w.Size[1] <= 0 {
			continue
		}
		x0 := mapCoord(w.At[0], originX, srcW, dstW)
		y0 := mapCoord(w.At[1], originY, srcH, dstH)
		x1 := mapCoord(w.At[0]+w.Size[0], originX, srcW, dstW)
		y1 := mapCoord(w.At[1]+w.Size[1], originY, srcH, dstH)
		if x1 < b.Min.X || y1 < b.Min.Y || x0 >= b.Max.X || y0 >= b.Max.Y {
			continue
		}
		strokeRect(img, x0, y0, x1, y1, annotateRed)
		drawID(img, x0+2, y0+2, w.ID)
	}
	return writePNG(path, img)
}

func strokeRect(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	if x1 < x0 {
		x0, x1 = x1, x0
	}
	if y1 < y0 {
		y0, y1 = y1, y0
	}
	if x1 == x0 {
		x1 = x0 + 1
	}
	if y1 == y0 {
		y1 = y0 + 1
	}
	for t := 0; t < 2; t++ {
		hline(img, x0, x1, y0+t, c)
		hline(img, x0, x1, y1-1-t, c)
		vline(img, y0, y1, x0+t, c)
		vline(img, y0, y1, x1-1-t, c)
	}
}

func hline(img *image.RGBA, x0, x1, y int, c color.RGBA) {
	b := img.Bounds()
	if y < b.Min.Y || y >= b.Max.Y {
		return
	}
	if x0 < b.Min.X {
		x0 = b.Min.X
	}
	if x1 > b.Max.X {
		x1 = b.Max.X
	}
	for x := x0; x < x1; x++ {
		img.SetRGBA(x, y, c)
	}
}

func vline(img *image.RGBA, y0, y1, x int, c color.RGBA) {
	b := img.Bounds()
	if x < b.Min.X || x >= b.Max.X {
		return
	}
	if y0 < b.Min.Y {
		y0 = b.Min.Y
	}
	if y1 > b.Max.Y {
		y1 = b.Max.Y
	}
	for y := y0; y < y1; y++ {
		img.SetRGBA(x, y, c)
	}
}

func drawID(img *image.RGBA, x, y, id int) {
	s := strconv.Itoa(id)
	const (
		gw, gh, scale, pad, gap = 5, 7, 2, 2, 1
	)
	tw := pad*2 + len(s)*gw*scale + (len(s)-1)*gap*scale
	th := pad*2 + gh*scale
	fillRect(img, x, y, x+tw, y+th, annotateRed)
	cx := x + pad
	cy := y + pad
	for i := 0; i < len(s); i++ {
		d := int(s[i] - '0')
		if d < 0 || d > 9 {
			continue
		}
		drawGlyph(img, cx, cy, d, annotateWhite)
		cx += (gw + gap) * scale
	}
}

func fillRect(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	b := img.Bounds()
	if x0 < b.Min.X {
		x0 = b.Min.X
	}
	if y0 < b.Min.Y {
		y0 = b.Min.Y
	}
	if x1 > b.Max.X {
		x1 = b.Max.X
	}
	if y1 > b.Max.Y {
		y1 = b.Max.Y
	}
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			img.SetRGBA(x, y, c)
		}
	}
}

func drawGlyph(img *image.RGBA, x, y, digit int, c color.RGBA) {
	g := digitGlyphs[digit]
	const scale = 2
	b := img.Bounds()
	for row := 0; row < 7; row++ {
		line := g[row]
		for col := 0; col < len(line) && col < 5; col++ {
			if line[col] != '#' {
				continue
			}
			for dy := 0; dy < scale; dy++ {
				py := y + row*scale + dy
				if py < b.Min.Y || py >= b.Max.Y {
					continue
				}
				for dx := 0; dx < scale; dx++ {
					px := x + col*scale + dx
					if px < b.Min.X || px >= b.Max.X {
						continue
					}
					img.SetRGBA(px, py, c)
				}
			}
		}
	}
}
