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

// SVGBasicWrite renders the paths encoded in the basic SVG image specified by
// sb. The scale value is used to convert the coordinates in the path to the
// unit of measure specified in New(). The current position (as set with a call
// to SetXY()) is used as the origin of the image. The current line cap style
// (as set with SetLineCapStyle()), line width (as set with SetLineWidth()),
// and draw color (as set with SetDrawColor()) are used in drawing the image
// paths.
func (f *Fpdf) SVGBasicWrite(sb *SVGBasicType, scale float64) {
	originX, originY := f.GetXY()
	var x, y, newX, newY float64
	var cx0, cy0, cx1, cy1 float64
	var path []SVGBasicSegmentType
	var seg SVGBasicSegmentType
	var startX, startY float64
	sval := func(origin float64, arg int) float64 {
		return origin + scale*seg.Arg[arg]
	}
	xval := func(arg int) float64 {
		return sval(originX, arg)
	}
	yval := func(arg int) float64 {
		return sval(originY, arg)
	}
	val := func(arg int) (float64, float64) {
		return xval(arg), yval(arg + 1)
	}
	for j := 0; j < len(sb.Segments) && f.Ok(); j++ {
		path = sb.Segments[j]
		for k := 0; k < len(path) && f.Ok(); k++ {
			seg = path[k]
			switch seg.Cmd {
			case 'M':
				x, y = val(0)
				startX, startY = x, y
				f.SetXY(x, y)
			case 'L':
				newX, newY = val(0)
				f.Line(x, y, newX, newY)
				x, y = newX, newY
			case 'C':
				cx0, cy0 = val(0)
				cx1, cy1 = val(2)
				newX, newY = val(4)
				f.CurveCubic(x, y, cx0, cy0, newX, newY, cx1, cy1, "D")
				x, y = newX, newY
			case 'Q':
				cx0, cy0 = val(0)
				newX, newY = val(2)
				f.Curve(x, y, cx0, cy0, newX, newY, "D")
				x, y = newX, newY
			case 'H':
				newX = xval(0)
				f.Line(x, y, newX, y)
				x = newX
			case 'V':
				newY = yval(0)
				f.Line(x, y, x, newY)
				y = newY
			case 'Z':
				f.Line(x, y, startX, startY)
				x, y = startX, startY
			default:
				f.SetErrorf("Unexpected path command '%c'", seg.Cmd)
			}
		}
	}
}

// SVGBasicDraw paints the paths encoded in the basic SVG image specified by
// sb. Unlike SVGBasicWrite(), which strokes every segment on its own, this
// method builds each path of the image as a single PDF path, so that it can be
// filled: this is the method to use for logos and other filled shapes.
//
// The scale value is used to convert the coordinates in the path to the unit
// of measure specified in New(). The current position (as set with a call to
// SetXY()) is used as the origin of the image, and it is left unchanged.
//
// styleStr can be "F" for filled, "D" for outlined only, or "DF" or "FD" for
// outlined and filled, with the same meaning and the same additional values
// as in DrawPath(): "F*", for instance, fills with the even-odd rule. Filling
// uses the current fill color (as set with SetFillColor()); outlining uses the
// current draw color, line width and cap and join styles.
func (f *Fpdf) SVGBasicDraw(sb *SVGBasicType, scale float64, styleStr string) {
	originX, originY := f.GetXY()
	xval := func(v float64) float64 {
		return originX + scale*v
	}
	yval := func(v float64) float64 {
		return originY + scale*v
	}
	for j := 0; j < len(sb.Segments) && f.Ok(); j++ {
		// Current point and start of the current subpath, in path units
		var x, y, startX, startY float64
		path := sb.Segments[j]
		for k := 0; k < len(path) && f.Ok(); k++ {
			arg := path[k].Arg
			switch path[k].Cmd {
			case 'M':
				x, y = arg[0], arg[1]
				startX, startY = x, y
				f.MoveTo(xval(x), yval(y))
			case 'L':
				x, y = arg[0], arg[1]
				f.LineTo(xval(x), yval(y))
			case 'H':
				x = arg[0]
				f.LineTo(xval(x), yval(y))
			case 'V':
				y = arg[0]
				f.LineTo(xval(x), yval(y))
			case 'C':
				f.CurveBezierCubicTo(xval(arg[0]), yval(arg[1]), xval(arg[2]), yval(arg[3]),
					xval(arg[4]), yval(arg[5]))
				x, y = arg[4], arg[5]
			case 'Q':
				// PDF has no quadratic curves: this is the same curve as a
				// cubic one
				f.CurveBezierCubicTo(
					xval(x+2*(arg[0]-x)/3), yval(y+2*(arg[1]-y)/3),
					xval(arg[2]+2*(arg[0]-arg[2])/3), yval(arg[3]+2*(arg[1]-arg[3])/3),
					xval(arg[2]), yval(arg[3]))
				x, y = arg[2], arg[3]
			case 'Z':
				f.ClosePath()
				x, y = startX, startY
			default:
				f.SetErrorf("Unexpected path command '%c'", path[k].Cmd)
			}
		}
		if len(path) > 0 && f.Ok() {
			f.DrawPath(styleStr)
		}
	}
	f.SetXY(originX, originY)
}
