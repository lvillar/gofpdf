/*
 * Copyright (c) 2014 Kurt Jung (Gmail: kurt.w.jung)
 *
 * Permission to use, copy, modify, and distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

package gofpdf

import (
	"encoding/xml"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

// SVGBasicSegmentType describes a single curve or position segment
type SVGBasicSegmentType struct {
	Cmd byte // See http://www.w3.org/TR/SVG/paths.html for path command structure
	Arg [6]float64
}

// pathLexer reads the commands and numbers of SVG path data. Separators are
// optional wherever the text is not ambiguous without them ("1.5.5-2" holds
// the numbers 1.5, .5 and -2), which is how optimized SVG files are written.
type pathLexer struct {
	str string
	pos int
}

// skipSeparators moves past white space and commas and reports whether any
// data is left.
func (lex *pathLexer) skipSeparators() bool {
	for lex.pos < len(lex.str) {
		switch lex.str[lex.pos] {
		case ' ', '\t', '\n', '\r', '\f', ',':
			lex.pos++
		default:
			return true
		}
	}
	return false
}

// number reads the number at the current position.
func (lex *pathLexer) number() (val float64, err error) {
	if !lex.skipSeparators() {
		return 0, fmt.Errorf("expecting additional numeric arguments in SVG path")
	}
	digits := func() (count int) {
		for lex.pos < len(lex.str) && lex.str[lex.pos] >= '0' && lex.str[lex.pos] <= '9' {
			lex.pos++
			count++
		}
		return
	}
	start := lex.pos
	if c := lex.str[lex.pos]; c == '-' || c == '+' {
		lex.pos++
	}
	count := digits()
	if lex.pos < len(lex.str) && lex.str[lex.pos] == '.' {
		lex.pos++
		count += digits()
	}
	if count == 0 {
		lex.pos = start
		return 0, fmt.Errorf("expecting number in SVG path at position %d, got %q", start, lex.str[start])
	}
	if lex.pos < len(lex.str) && (lex.str[lex.pos] == 'e' || lex.str[lex.pos] == 'E') {
		// An exponent needs at least one digit; otherwise the letter is not
		// part of the number.
		mark := lex.pos
		lex.pos++
		if lex.pos < len(lex.str) && (lex.str[lex.pos] == '-' || lex.str[lex.pos] == '+') {
			lex.pos++
		}
		if digits() == 0 {
			lex.pos = mark
		}
	}
	return strconv.ParseFloat(lex.str[start:lex.pos], 64)
}

// flag reads one of the two flags of an arc: a single 0 or 1 that needs no
// separator after it.
func (lex *pathLexer) flag() (set bool, err error) {
	if !lex.skipSeparators() {
		return false, fmt.Errorf("expecting additional arc flags in SVG path")
	}
	c := lex.str[lex.pos]
	if c != '0' && c != '1' {
		return false, fmt.Errorf("expecting arc flag (0 or 1) in SVG path at position %d, got %q", lex.pos, c)
	}
	lex.pos++
	return c == '1', nil
}

// pathArgCounts holds the number of arguments that each path command takes.
var pathArgCounts = map[byte]int{
	'M': 2, 'L': 2, 'H': 1, 'V': 1, 'C': 6, 'S': 4, 'Q': 4, 'T': 2, 'A': 7, 'Z': 0,
}

// pathParse converts SVG path data into segments with absolute coordinates
// that use only the commands M, L, H, V, C, Q and Z. Smooth curves (S and T)
// are returned as the plain curves they stand for (C and Q), and elliptical
// arcs (A) as the cubic Bézier curves that approximate them.
func pathParse(pathStr string) (segs []SVGBasicSegmentType, err error) {
	var (
		cmd            byte    // command in effect, zero before the first one
		x, y           float64 // current point
		startX, startY float64 // start of the current subpath
		ctrlX, ctrlY   float64 // last control point of the previous curve
		prev           byte    // kind of the previous segment: 'C', 'Q' or zero
		arg            [7]float64
	)
	add := func(c byte, args ...float64) {
		seg := SVGBasicSegmentType{Cmd: c}
		copy(seg.Arg[:], args)
		segs = append(segs, seg)
	}
	lex := pathLexer{str: pathStr}
	for lex.skipSeparators() {
		if c := lex.str[lex.pos]; (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
			if _, ok := pathArgCounts[c&^0x20]; !ok {
				return nil, fmt.Errorf("expecting SVG path command at position %d, got %q", lex.pos, c)
			}
			cmd = c
			lex.pos++
			if cmd == 'Z' || cmd == 'z' {
				add('Z')
				x, y = startX, startY
				prev = 0
				continue
			}
		} else if cmd == 0 || cmd == 'Z' || cmd == 'z' {
			return nil, fmt.Errorf("expecting SVG path command at position %d, got %q", lex.pos, c)
		}
		relative := cmd >= 'a'
		upper := cmd &^ 0x20
		for j := 0; j < pathArgCounts[upper]; j++ {
			if upper == 'A' && (j == 3 || j == 4) {
				var set bool
				if set, err = lex.flag(); err != nil {
					return nil, err
				}
				arg[j] = 0
				if set {
					arg[j] = 1
				}
			} else if arg[j], err = lex.number(); err != nil {
				return nil, err
			}
		}
		// Offsets that turn the arguments of a relative command into
		// absolute coordinates
		var offX, offY float64
		if relative {
			offX, offY = x, y
		}
		kind := byte(0)
		switch upper {
		case 'M':
			x, y = arg[0]+offX, arg[1]+offY
			startX, startY = x, y
			add('M', x, y)
			// Further coordinate pairs after a moveto are linetos
			cmd = cmd - 'M' + 'L'
		case 'L':
			x, y = arg[0]+offX, arg[1]+offY
			add('L', x, y)
		case 'H':
			x = arg[0] + offX
			add('H', x)
		case 'V':
			y = arg[0] + offY
			add('V', y)
		case 'C':
			ctrlX, ctrlY = arg[2]+offX, arg[3]+offY
			add('C', arg[0]+offX, arg[1]+offY, ctrlX, ctrlY, arg[4]+offX, arg[5]+offY)
			x, y = arg[4]+offX, arg[5]+offY
			kind = 'C'
		case 'S':
			// The first control point mirrors the last one of the previous
			// cubic curve; without one it is the current point.
			cx0, cy0 := x, y
			if prev == 'C' {
				cx0, cy0 = 2*x-ctrlX, 2*y-ctrlY
			}
			ctrlX, ctrlY = arg[0]+offX, arg[1]+offY
			add('C', cx0, cy0, ctrlX, ctrlY, arg[2]+offX, arg[3]+offY)
			x, y = arg[2]+offX, arg[3]+offY
			kind = 'C'
		case 'Q':
			ctrlX, ctrlY = arg[0]+offX, arg[1]+offY
			add('Q', ctrlX, ctrlY, arg[2]+offX, arg[3]+offY)
			x, y = arg[2]+offX, arg[3]+offY
			kind = 'Q'
		case 'T':
			// The control point mirrors the one of the previous quadratic
			// curve; without one it is the current point.
			if prev == 'Q' {
				ctrlX, ctrlY = 2*x-ctrlX, 2*y-ctrlY
			} else {
				ctrlX, ctrlY = x, y
			}
			x, y = arg[0]+offX, arg[1]+offY
			add('Q', ctrlX, ctrlY, x, y)
			kind = 'Q'
		case 'A':
			endX, endY := arg[5]+offX, arg[6]+offY
			if endX != x || endY != y { // an arc that goes nowhere is left out
				if arg[0] == 0 || arg[1] == 0 { // without a radius it is a line
					add('L', endX, endY)
				} else {
					for _, c := range arcToCubics(x, y, arg[0], arg[1], arg[2], arg[3] != 0, arg[4] != 0, endX, endY) {
						add('C', c[:]...)
					}
				}
				x, y = endX, endY
			}
		}
		prev = kind
	}
	return
}

// arcToCubics returns the cubic Bézier curves (two control points and the end
// point of each one) that approximate the elliptical arc of an SVG path that
// goes from (x1, y1) to (x2, y2). The arguments are those of the arc command:
// the radii, the rotation of the ellipse in degrees and the two flags that
// pick one of the four possible arcs. The conversion follows the
// implementation notes of the SVG specification
// (https://www.w3.org/TR/SVG/implnote.html#ArcImplementationNotes).
func arcToCubics(x1, y1, rx, ry, degRotate float64, largeArc, sweep bool, x2, y2 float64) (curves [][6]float64) {
	rx, ry = math.Abs(rx), math.Abs(ry)
	sinPhi, cosPhi := math.Sincos(degRotate * math.Pi / 180)
	// Start point in the frame of the ellipse, measured from the middle of
	// the chord
	hx, hy := (x1-x2)/2, (y1-y2)/2
	x1p, y1p := cosPhi*hx+sinPhi*hy, -sinPhi*hx+cosPhi*hy
	// Radii too small to join both ends are scaled up until they do
	if lambda := x1p*x1p/(rx*rx) + y1p*y1p/(ry*ry); lambda > 1 {
		rx *= math.Sqrt(lambda)
		ry *= math.Sqrt(lambda)
	}
	// Center of the ellipse
	num := rx*rx*ry*ry - rx*rx*y1p*y1p - ry*ry*x1p*x1p
	den := rx*rx*y1p*y1p + ry*ry*x1p*x1p
	coef := math.Sqrt(math.Max(0, num/den))
	if largeArc == sweep {
		coef = -coef
	}
	cxp, cyp := coef*rx*y1p/ry, -coef*ry*x1p/rx
	cx := cosPhi*cxp - sinPhi*cyp + (x1+x2)/2
	cy := sinPhi*cxp + cosPhi*cyp + (y1+y2)/2
	// Angles of both ends and the angle swept between them
	theta := math.Atan2((y1p-cyp)/ry, (x1p-cxp)/rx)
	delta := math.Atan2((-y1p-cyp)/ry, (-x1p-cxp)/rx) - theta
	if sweep && delta < 0 {
		delta += 2 * math.Pi
	} else if !sweep && delta > 0 {
		delta -= 2 * math.Pi
	}
	// One curve for each quarter of a turn, or part of it
	count := int(math.Ceil(math.Abs(delta)/(math.Pi/2) - 1e-9))
	if count < 1 {
		count = 1
	}
	step := delta / float64(count)
	handle := 4.0 / 3.0 * math.Tan(step/4)
	// point returns the point of the ellipse at the specified angle and the
	// direction in which the arc leaves it
	point := func(angle float64) (px, py, dx, dy float64) {
		sin, cos := math.Sincos(angle)
		px = cx + rx*cos*cosPhi - ry*sin*sinPhi
		py = cy + rx*cos*sinPhi + ry*sin*cosPhi
		dx = -rx*sin*cosPhi - ry*cos*sinPhi
		dy = -rx*sin*sinPhi + ry*cos*cosPhi
		return
	}
	px, py, dx, dy := point(theta)
	for j := 1; j <= count; j++ {
		nx, ny, ndx, ndy := point(theta + float64(j)*step)
		if j == count {
			nx, ny = x2, y2 // the arc ends exactly where it was asked to
		}
		curves = append(curves, [6]float64{
			px + handle*dx, py + handle*dy,
			nx - handle*ndx, ny - handle*ndy,
			nx, ny,
		})
		px, py, dx, dy = nx, ny, ndx, ndy
	}
	return
}

// SVGBasicType aggregates the information needed to describe a multi-segment
// basic vector image
type SVGBasicType struct {
	Wd, Ht   float64
	Segments [][]SVGBasicSegmentType
}

// SVGBasicParse parses a simple scalable vector graphics (SVG) buffer into a
// descriptor. Only a small subset of the SVG standard is supported: the path
// elements that are direct children of the root element, without styles or
// transformations. All the path commands are understood. The returned path
// data includes only the commands 'M' (absolute moveto: x, y), 'L' (absolute
// lineto: x, y), 'H' (absolute horizontal lineto: x), 'V' (absolute vertical
// lineto: y), 'C' (absolute cubic Bézier curve: cx0, cy0, cx1, cy1, x1, y1),
// 'Q' (absolute quadratic Bézier curve: x0, y0, x1, y1) and 'Z' (closepath):
// relative coordinates are made absolute, smooth curves ('S' and 'T') are
// returned as 'C' and 'Q', and elliptical arcs ('A') as the 'C' curves that
// approximate them.
//
// The size of the image is taken from the width and height attributes of the
// root element, which must be plain numbers. When they are missing it is taken
// from the viewBox attribute, and the path coordinates are then measured from
// the top left corner of the viewBox.
func SVGBasicParse(buf []byte) (sig SVGBasicType, err error) {
	type pathType struct {
		D string `xml:"d,attr"`
	}
	type srcType struct {
		Wd      float64    `xml:"width,attr"`
		Ht      float64    `xml:"height,attr"`
		ViewBox string     `xml:"viewBox,attr"`
		Paths   []pathType `xml:"path"`
	}
	var src srcType
	err = xml.Unmarshal(buf, &src)
	if err == nil {
		var originX, originY float64
		if src.Wd > 0 && src.Ht > 0 {
			sig.Wd, sig.Ht = src.Wd, src.Ht
		} else if box, ok := viewBoxParse(src.ViewBox); ok {
			originX, originY, sig.Wd, sig.Ht = box[0], box[1], box[2], box[3]
		} else {
			err = fmt.Errorf("unacceptable values for basic SVG extent: %.2f x %.2f",
				src.Wd, src.Ht)
		}
		var segs []SVGBasicSegmentType
		for _, path := range src.Paths {
			if err == nil {
				segs, err = pathParse(path.D)
				if err == nil {
					translatePath(segs, -originX, -originY)
					sig.Segments = append(sig.Segments, segs)
				}
			}
		}
	}
	return
}

// viewBoxParse returns the four numbers of a viewBox attribute (left, top,
// width and height). ok is false if the attribute does not hold a box with a
// positive size.
func viewBoxParse(str string) (box [4]float64, ok bool) {
	fields := strings.FieldsFunc(str, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
	})
	if len(fields) != len(box) {
		return
	}
	for j, field := range fields {
		val, err := strconv.ParseFloat(field, 64)
		if err != nil {
			return
		}
		box[j] = val
	}
	ok = box[2] > 0 && box[3] > 0
	return
}

// translatePath moves the absolute segments of a path by the specified
// distances.
func translatePath(segs []SVGBasicSegmentType, dx, dy float64) {
	if dx == 0 && dy == 0 {
		return
	}
	for j := range segs {
		seg := &segs[j]
		switch seg.Cmd {
		case 'H':
			seg.Arg[0] += dx
		case 'V':
			seg.Arg[0] += dy
		case 'M', 'L':
			seg.Arg[0] += dx
			seg.Arg[1] += dy
		case 'Q', 'C':
			for k := 0; k < len(seg.Arg); k += 2 {
				seg.Arg[k] += dx
				seg.Arg[k+1] += dy
			}
		}
	}
}

// SVGBasicFileParse parses a simple scalable vector graphics (SVG) file into a
// basic descriptor. The SVGBasicWrite() example demonstrates this method.
func SVGBasicFileParse(svgFileStr string) (sig SVGBasicType, err error) {
	var buf []byte
	buf, err = os.ReadFile(svgFileStr)
	if err == nil {
		sig, err = SVGBasicParse(buf)
	}
	return
}
