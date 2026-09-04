package see

import (
	"image"
	"image/draw"
)

// resizeLongEdge downscales so max(width,height) == maxEdge. No-op if already smaller.
func resizeLongEdge(img image.Image, maxEdge int) *image.RGBA {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	src := toRGBA(img)
	if maxEdge < 1 || w < 1 || h < 1 {
		return src
	}
	long := w
	if h > w {
		long = h
	}
	if long <= maxEdge {
		return src
	}
	nw, nh := w*maxEdge/long, h*maxEdge/long
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}
	return resizeNearest(src, nw, nh)
}

func toRGBA(img image.Image) *image.RGBA {
	if r, ok := img.(*image.RGBA); ok && r.Bounds().Min == image.Pt(0, 0) {
		return r
	}
	b := img.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), img, b.Min, draw.Src)
	return dst
}

func resizeNearest(src *image.RGBA, nw, nh int) *image.RGBA {
	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	for y := 0; y < nh; y++ {
		sy := sb.Min.Y + y*sh/nh
		for x := 0; x < nw; x++ {
			sx := sb.Min.X + x*sw/nw
			dst.SetRGBA(x, y, src.RGBAAt(sx, sy))
		}
	}
	return dst
}

func downscaleFile(path string, maxEdge int) (width, height int, err error) {
	img, err := readRGBA(path)
	if err != nil {
		return 0, 0, err
	}
	src := img.Bounds()
	out := resizeLongEdge(img, maxEdge)
	b := out.Bounds()
	if b.Dx() != src.Dx() || b.Dy() != src.Dy() {
		if err := writePNG(path, out); err != nil {
			return 0, 0, err
		}
	}
	return b.Dx(), b.Dy(), nil
}
