package ui

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"math"
	"sync"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"perimeter/internal/guard"
)

// Tone is the colour of the icon.
type Tone int

const (
	Gray Tone = iota
	Green
	Amber
	Red
)

var tones = map[Tone]color.NRGBA{
	Gray:  {0x9c, 0xa3, 0xaf, 0xff},
	Green: {0x16, 0xa3, 0x4a, 0xff},
	Amber: {0xea, 0x8a, 0x00, 0xff},
	Red:   {0xdc, 0x26, 0x26, 0xff},
}

func ToneOf(s guard.Snapshot) Tone {
	if s.Paused() {
		return Gray
	}
	switch s.Status {
	case guard.OK:
		return Green
	case guard.Suspect:
		return Amber
	case guard.Blocked:
		return Red
	}
	return Gray
}

// coverage of a pixel by an anti-aliased rounded rectangle (r = corner radius).
func roundRectAlpha(x, y, w, h, r float64) float64 {
	cx := math.Max(math.Max(r-x, x-(w-r)), 0)
	cy := math.Max(math.Max(r-y, y-(h-r)), 0)
	d := math.Hypot(cx, cy) - r
	return math.Min(math.Max(0.5-d, 0), 1)
}

func fill(img *image.NRGBA, c color.NRGBA, alpha func(x, y float64) float64) {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			a := alpha(float64(x)+0.5, float64(y)+0.5)
			if a > 0 {
				cc := c
				cc.A = uint8(float64(c.A) * a)
				img.SetNRGBA(x, y, cc)
			}
		}
	}
}

// barColors: the alarm colours of the menu bar icon (system orange and red).
var barColors = map[Tone]color.NRGBA{Amber: {0xff, 0x9f, 0x0a, 0xff}, Red: {0xff, 0x45, 0x3a, 0xff}}

// BarIcon is the macOS menu bar icon, 16 pt drawn at 32 px: the logo's fence.
// Calm states are template images (black; macOS tints them to the menu bar),
// alarms are coloured. lit is how many of the 8 dashes still glow while apps
// are frozen: they go out one by one as the grace time runs out.
//
//	Green - full fence and the dot         (template)
//	Gray  - dim fence and a pause sign     (template)
//	Amber - lit dashes and the dot, orange (frozen)
//	Red   - dim fence and a cross, red     (closed)
func BarIcon(t Tone, lit int) (png []byte, template bool) {
	f := fence{c: 8, h: 5.5, r: 3.2, l: 2.12}
	const w = 0.8 // half the stroke
	c, template := color.NRGBA{0, 0, 0, 0xff}, true
	if bc, ok := barColors[t]; ok {
		c, template = bc, false
	}
	type mark struct {
		dist  func(x, y float64) float64
		half  float64
		alpha float64
	}
	dashes := func(from, to int, alpha float64) mark {
		return mark{func(x, y float64) float64 {
			d := math.Inf(1)
			for i := from; i < to; i++ {
				d = math.Min(d, f.dist(i, x, y))
			}
			return d
		}, w, alpha}
	}
	dot := mark{func(x, y float64) float64 { return math.Hypot(x-8, y-8) }, 1.9, 1}
	seg := func(alpha float64, s ...[4]float64) mark {
		return mark{func(x, y float64) float64 {
			d := math.Inf(1)
			for _, v := range s {
				d = math.Min(d, segDist(x, y, v[0], v[1], v[2], v[3]))
			}
			return d
		}, 0.75, alpha}
	}
	var marks []mark
	switch t {
	case Gray:
		marks = []mark{dashes(0, 8, 0.4), seg(0.75, [4]float64{6.8, 6, 6.8, 10}, [4]float64{9.2, 6, 9.2, 10})}
	case Amber:
		lit = max(0, min(8, lit))
		marks = []mark{dashes(0, lit, 1), dashes(lit, 8, 0.22), dot}
	case Red:
		marks = []mark{dashes(0, 8, 0.25), seg(1, [4]float64{6.4, 6.4, 9.6, 9.6}, [4]float64{9.6, 6.4, 6.4, 9.6})}
	default:
		marks = []mark{dashes(0, 8, 1), dot}
	}
	const n = 32
	unit := float64(n) / 16
	img := image.NewNRGBA(image.Rect(0, 0, n, n))
	for py := 0; py < n; py++ {
		for px := 0; px < n; px++ {
			x, y := (float64(px)+0.5)/unit, (float64(py)+0.5)/unit
			a := 0.0
			for _, m := range marks {
				a = math.Max(a, m.alpha*clamp01(0.5-(m.dist(x, y)-m.half)*unit))
			}
			if a > 0 {
				img.SetNRGBA(px, py, color.NRGBA{c.R, c.G, c.B, uint8(a*255 + 0.5)})
			}
		}
	}
	return pngBytes(img), template
}

// BarLit is how many fence dashes glow with this much grace time left.
func BarLit(left, grace time.Duration) int {
	if grace <= 0 {
		return 8
	}
	return int(math.Ceil(8 * float64(left) / float64(grace)))
}

var (
	faceMu sync.Mutex
	faces  = map[float64]font.Face{}
)

func face(size float64) font.Face {
	faceMu.Lock()
	defer faceMu.Unlock()
	if f, ok := faces[size]; ok {
		return f
	}
	ft, _ := opentype.Parse(gobold.TTF)
	f, _ := opentype.NewFace(ft, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
	faces[size] = f
	return f
}

// Badge is the tray icon on Windows and Linux: a coloured tile with the country code,
// so the country is visible at a glance even without a menu bar title.
func Badge(t Tone, cc string, n int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, n, n))
	fn := float64(n)
	fill(img, tones[t], func(x, y float64) float64 { return roundRectAlpha(x, y, fn, fn, fn*0.22) })
	if len(cc) != 2 {
		// No country yet: a white dot in the middle.
		fill(img, color.NRGBA{255, 255, 255, 255}, func(x, y float64) float64 {
			d := math.Hypot(x-fn/2, y-fn/2) - fn*0.16
			return math.Min(math.Max(0.5-d, 0), 1)
		})
		return img
	}
	f := face(fn * 0.62)
	adv := font.MeasureString(f, cc)
	m := f.Metrics()
	x := (fixed.I(n) - adv) / 2
	y := (fixed.I(n) + m.Ascent - m.Descent) / 2
	d := &font.Drawer{Dst: img, Src: image.NewUniform(color.White), Face: f, Dot: fixed.Point26_6{X: x, Y: y}}
	d.DrawString(cc)
	return img
}

func pngBytes(img image.Image) []byte {
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}

// ICO packs PNG images into a .ico file (supported since Windows Vista).
func ICO(imgs ...*image.NRGBA) []byte {
	var hdr, body bytes.Buffer
	binary.Write(&hdr, binary.LittleEndian, [3]uint16{0, 1, uint16(len(imgs))})
	offset := 6 + 16*len(imgs)
	for _, img := range imgs {
		p := pngBytes(img)
		w := img.Bounds().Dx()
		if w >= 256 {
			w = 0
		}
		binary.Write(&hdr, binary.LittleEndian, struct {
			W, H, Colors, Reserved uint8
			Planes, BPP            uint16
			Size, Offset           uint32
		}{uint8(w), uint8(w), 0, 0, 1, 32, uint32(len(p)), uint32(offset)})
		body.Write(p)
		offset += len(p)
	}
	return append(hdr.Bytes(), body.Bytes()...)
}

// AppIcon is the big icon for the .app bundle, the .exe and the README.
func AppIcon(n int) *image.NRGBA { return Logo(n, Green) }

// logoFence: the 46x46 square at 27..73, corner 13; a dash is 5.5% of the outline.
var logoFence = fence{c: 50, h: 23, r: 13, l: 8.892}

// Logo is the Perimeter mark, drawn on a 100x100 grid: a blue tile (80x80,
// corner 18) with a dashed perimeter - eight dashes, one on every corner and
// side centre - around a dot. Gray is the browser extension's "off" look.
func Logo(n int, t Tone) *image.NRGBA {
	top, bottom := color.NRGBA{0x2f, 0x80, 0xff, 0xff}, color.NRGBA{0x17, 0x64, 0xe6, 0xff}
	if t == Gray {
		top, bottom = color.NRGBA{0xa1, 0xa1, 0xaa, 0xff}, color.NRGBA{0x80, 0x80, 0x89, 0xff}
	}
	img := image.NewNRGBA(image.Rect(0, 0, n, n))
	unit := float64(n) / 100
	aa := 1 / unit // one pixel in grid units: the width of the anti-aliased edge
	for py := 0; py < n; py++ {
		for px := 0; px < n; px++ {
			x, y := (float64(px)+0.5)/unit, (float64(py)+0.5)/unit
			body := clamp01(0.5 - sdRoundRect(x, y, 50, 50, 40, 40, 18)/aa)
			// A soft shadow under the tile, as macOS icons have.
			shadow := 0.0
			if d := sdRoundRect(x, y-1.6, 50, 50, 40, 40, 18); d > 0 {
				shadow = 0.22 * clamp01(1-d/3.5)
			} else {
				shadow = 0.22
			}
			k := clamp01((y - 10) / 80)
			c := [3]float64{lerp(top.R, bottom.R, k), lerp(top.G, bottom.G, k), lerp(top.B, bottom.B, k)}
			mark := clamp01(0.5 - math.Min(logoFence.near(x, y)-3, math.Hypot(x-50, y-50)-7)/aa)
			for i := range c {
				c[i] += (255 - c[i]) * mark
			}
			a := body + shadow*(1-body)
			if a <= 0 {
				continue
			}
			img.SetNRGBA(px, py, color.NRGBA{
				uint8(c[0]*body/a + 0.5), uint8(c[1]*body/a + 0.5), uint8(c[2]*body/a + 0.5), uint8(a*255 + 0.5),
			})
		}
	}
	return img
}

// fence is the logo's dashed perimeter: a rounded square centred at (c, c)
// with half-size h and corner r, and eight dashes of length l - one on every
// side centre and corner, clockwise from the top.
type fence struct{ c, h, r, l float64 }

// dist is the distance from (x, y) to the centre line of dash i (0..7).
func (f fence) dist(i int, x, y float64) float64 {
	half := f.l / 2
	if i%2 == 0 { // a side: top, right, bottom, left
		dx, dy := [4]float64{0, 1, 0, -1}[i/2], [4]float64{-1, 0, 1, 0}[i/2]
		mx, my := f.c+dx*f.h, f.c+dy*f.h
		return segDist(x, y, mx-dy*half, my-dx*half, mx+dy*half, my+dx*half)
	}
	sx, sy := [4]float64{1, 1, -1, -1}[i/2], [4]float64{-1, 1, 1, -1}[i/2] // the corner's direction
	in := f.h - f.r
	return arcDist(x, y, f.c+sx*in, f.c+sy*in, f.r, math.Atan2(sy, sx), half/f.r)
}

// near is the distance to the nearest dash.
func (f fence) near(x, y float64) float64 {
	d := math.Inf(1)
	for i := 0; i < 8; i++ {
		d = math.Min(d, f.dist(i, x, y))
	}
	return d
}

func segDist(x, y, x1, y1, x2, y2 float64) float64 {
	dx, dy := x2-x1, y2-y1
	t := clamp01(((x-x1)*dx + (y-y1)*dy) / (dx*dx + dy*dy))
	return math.Hypot(x-(x1+t*dx), y-(y1+t*dy))
}

// arcDist: distance to the arc of radius r around (cx, cy) spanning mid±span.
func arcDist(x, y, cx, cy, r, mid, span float64) float64 {
	diff := math.Remainder(math.Atan2(y-cy, x-cx)-mid, 2*math.Pi)
	if math.Abs(diff) <= span {
		return math.Abs(math.Hypot(x-cx, y-cy) - r)
	}
	a1, a2 := mid-span, mid+span
	return math.Min(math.Hypot(x-(cx+r*math.Cos(a1)), y-(cy+r*math.Sin(a1))),
		math.Hypot(x-(cx+r*math.Cos(a2)), y-(cy+r*math.Sin(a2))))
}

// sdRoundRect is the signed distance to a rounded rectangle (negative inside).
func sdRoundRect(x, y, cx, cy, hx, hy, r float64) float64 {
	qx, qy := math.Abs(x-cx)-hx+r, math.Abs(y-cy)-hy+r
	return math.Hypot(math.Max(qx, 0), math.Max(qy, 0)) + math.Min(math.Max(qx, qy), 0) - r
}

func clamp01(v float64) float64 { return math.Min(math.Max(v, 0), 1) }

func lerp(a, b uint8, k float64) float64 { return float64(a) + (float64(b)-float64(a))*k }
