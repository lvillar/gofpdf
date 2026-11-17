package gofpdf_test

import (
	"bytes"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/lvillar/gofpdf"
)

// svgSeg is a path segment written by hand: the command and only the
// arguments that command uses.
type svgSeg struct {
	cmd  byte
	args []float64
}

func svgDoc(d string) []byte {
	return []byte(`<svg width="100" height="100"><path d="` + d + `"/></svg>`)
}

func describeSegments(segs []gofpdf.SVGBasicSegmentType) string {
	var sb strings.Builder
	for _, seg := range segs {
		fmt.Fprintf(&sb, "%c%v ", seg.Cmd, seg.Arg)
	}
	return sb.String()
}

// parseOnePath parses a document that holds a single path and returns its
// segments.
func parseOnePath(t *testing.T, doc []byte) []gofpdf.SVGBasicSegmentType {
	t.Helper()
	sig, err := gofpdf.SVGBasicParse(doc)
	if err != nil {
		t.Fatalf("SVGBasicParse: %v", err)
	}
	if len(sig.Segments) != 1 {
		t.Fatalf("got %d paths, want 1", len(sig.Segments))
	}
	return sig.Segments[0]
}

// checkSegments compares a parsed path against the segments expected by hand.
func checkSegments(t *testing.T, got []gofpdf.SVGBasicSegmentType, want []svgSeg) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d segments, want %d: %s", len(got), len(want), describeSegments(got))
	}
	for j, w := range want {
		if got[j].Cmd != w.cmd {
			t.Fatalf("segment %d: got command %q, want %q: %s", j, got[j].Cmd, w.cmd, describeSegments(got))
		}
		for k, arg := range w.args {
			if math.Abs(got[j].Arg[k]-arg) > 1e-4 {
				t.Errorf("segment %d (%c), argument %d: got %.5f, want %.5f", j, w.cmd, k, got[j].Arg[k], arg)
			}
		}
	}
}

// kappa is the distance of the control points of the cubic Bézier curve that
// approximates a quarter of a unit circle: 4/3 * tan(pi/8).
const kappa = 0.5522847498

func TestSVGBasicParse_paths(t *testing.T) {
	tests := []struct {
		name string
		d    string
		want []svgSeg
	}{
		{
			// Optimized SVG files drop every separator that is not needed:
			// a sign or a second decimal point starts a new number.
			name: "numbers without separators",
			d:    "M1.5.5-2,3l.5-.25",
			want: []svgSeg{
				{'M', []float64{1.5, 0.5}},
				{'L', []float64{-2, 3}},
				{'L', []float64{-1.5, 2.75}},
			},
		},
		{
			name: "exponent notation",
			d:    "M1e1 2E-1L-1.5e+1 0",
			want: []svgSeg{
				{'M', []float64{10, 0.2}},
				{'L', []float64{-15, 0}},
			},
		},
		{
			// The first control point is the reflection of the previous
			// curve's second control point (3,4) about its end (5,6).
			name: "smooth cubic after a cubic",
			d:    "M0 0C1 2 3 4 5 6S9 10 11 12",
			want: []svgSeg{
				{'M', []float64{0, 0}},
				{'C', []float64{1, 2, 3, 4, 5, 6}},
				{'C', []float64{7, 8, 9, 10, 11, 12}},
			},
		},
		{
			name: "relative smooth cubic",
			d:    "M0 0c1 2 3 4 5 6s4 4 6 6",
			want: []svgSeg{
				{'M', []float64{0, 0}},
				{'C', []float64{1, 2, 3, 4, 5, 6}},
				{'C', []float64{7, 8, 9, 10, 11, 12}},
			},
		},
		{
			// Without a previous curve the first control point is the
			// current point.
			name: "smooth cubic after a line",
			d:    "M0 0L1 1S2 3 4 5",
			want: []svgSeg{
				{'M', []float64{0, 0}},
				{'L', []float64{1, 1}},
				{'C', []float64{1, 1, 2, 3, 4, 5}},
			},
		},
		{
			name: "smooth cubics chained without repeating the command",
			d:    "M0 0C0 1 1 2 2 2S4 1 4 0 3-2 2-2",
			want: []svgSeg{
				{'M', []float64{0, 0}},
				{'C', []float64{0, 1, 1, 2, 2, 2}},
				{'C', []float64{3, 2, 4, 1, 4, 0}},
				{'C', []float64{4, -1, 3, -2, 2, -2}},
			},
		},
		{
			// The control point is the reflection of the previous one (1,2)
			// about the end of that curve (3,4).
			name: "smooth quadratic after a quadratic",
			d:    "M0 0Q1 2 3 4T7 8",
			want: []svgSeg{
				{'M', []float64{0, 0}},
				{'Q', []float64{1, 2, 3, 4}},
				{'Q', []float64{5, 6, 7, 8}},
			},
		},
		{
			name: "relative smooth quadratic after a line",
			d:    "M0 0l3 4t4 4",
			want: []svgSeg{
				{'M', []float64{0, 0}},
				{'L', []float64{3, 4}},
				{'Q', []float64{3, 4, 7, 8}},
			},
		},
		{
			// A quarter of the circle of radius 10 centered at the origin,
			// from (10,0) to (0,10).
			name: "arc of a quarter of a circle",
			d:    "M10 0A10 10 0 0 1 0 10",
			want: []svgSeg{
				{'M', []float64{10, 0}},
				{'C', []float64{10, 10 * kappa, 10 * kappa, 10, 0, 10}},
			},
		},
		{
			// The two flags of an arc are single digits and may come glued
			// to each other and to the next number.
			name: "relative arc with glued flags",
			d:    "M10 0a10 10 0 01-10 10",
			want: []svgSeg{
				{'M', []float64{10, 0}},
				{'C', []float64{10, 10 * kappa, 10 * kappa, 10, 0, 10}},
			},
		},
		{
			// Sweep flag 0 turns the other way: the same end points are
			// joined through the quarter of the circle centered at (10,10).
			name: "arc turning the other way",
			d:    "M10 0A10 10 0 0 0 0 10",
			want: []svgSeg{
				{'M', []float64{10, 0}},
				{'C', []float64{10 - 10*kappa, 0, 0, 10 - 10*kappa, 0, 10}},
			},
		},
		{
			// Half a circle of radius 5 centered at (5,0), through (5,-5): it
			// takes two curves, one for each quarter.
			name: "arc of half a circle",
			d:    "M0 0A5 5 0 0 1 10 0",
			want: []svgSeg{
				{'M', []float64{0, 0}},
				{'C', []float64{0, -5 * kappa, 5 - 5*kappa, -5, 5, -5}},
				{'C', []float64{5 + 5*kappa, -5, 10, -5 * kappa, 10, 0}},
			},
		},
		{
			// Radii too small to reach the end point are scaled up until
			// they do (here from 1 to 5).
			name: "arc with radii too small",
			d:    "M0 0A1 1 0 0 1 10 0",
			want: []svgSeg{
				{'M', []float64{0, 0}},
				{'C', []float64{0, -5 * kappa, 5 - 5*kappa, -5, 5, -5}},
				{'C', []float64{5 + 5*kappa, -5, 10, -5 * kappa, 10, 0}},
			},
		},
		{
			// The large-arc flag picks the three quarters of the circle
			// centered at (10,0), through (10,-10) and (20,0), instead of the
			// quarter of the circle centered at (0,10).
			name: "large arc",
			d:    "M0 0A10 10 0 1 1 10 10",
			want: []svgSeg{
				{'M', []float64{0, 0}},
				{'C', []float64{0, -10 * kappa, 10 - 10*kappa, -10, 10, -10}},
				{'C', []float64{10 + 10*kappa, -10, 20, -10 * kappa, 20, 0}},
				{'C', []float64{20, 10 * kappa, 10 + 10*kappa, 10, 10, 10}},
			},
		},
		{
			// An ellipse with radii 20 and 10 rotated 90 degrees is an
			// ellipse with radii 10 and 20.
			name: "rotated elliptical arc",
			d:    "M10 0A20 10 90 0 1 0 20",
			want: []svgSeg{
				{'M', []float64{10, 0}},
				{'C', []float64{10, 20 * kappa, 10 * kappa, 20, 0, 20}},
			},
		},
		{
			name: "arc with a zero radius is a straight line",
			d:    "M0 0A0 5 0 0 1 10 10",
			want: []svgSeg{
				{'M', []float64{0, 0}},
				{'L', []float64{10, 10}},
			},
		},
		{
			name: "arc that ends where it starts draws nothing",
			d:    "M3 4A5 5 0 0 1 3 4L6 8",
			want: []svgSeg{
				{'M', []float64{3, 4}},
				{'L', []float64{6, 8}},
			},
		},
		{
			// After closing a subpath the current point goes back to its
			// start, which is where a relative move is measured from.
			name: "relative move after closing a subpath",
			d:    "M10 10h5v5zm1 1h2",
			want: []svgSeg{
				{'M', []float64{10, 10}},
				{'H', []float64{15}},
				{'V', []float64{15}},
				{'Z', nil},
				{'M', []float64{11, 11}},
				{'H', []float64{13}},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			checkSegments(t, parseOnePath(t, svgDoc(tc.d)), tc.want)
		})
	}
}

func TestSVGBasicParse_invalidPaths(t *testing.T) {
	tests := []struct {
		name string
		d    string
	}{
		{"unknown command", "M0 0X5 5"},
		{"missing arguments", "M0 0L5"},
		{"number before any command", "5 5L1 1"},
		{"arc flag that is not 0 or 1", "M0 0A5 5 0 2 1 10 0"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := gofpdf.SVGBasicParse(svgDoc(tc.d)); err == nil {
				t.Errorf("path %q: expected an error", tc.d)
			}
		})
	}
}

// A document without width and height takes its size from the viewBox, and
// its coordinates are measured from the corner of the viewBox.
func TestSVGBasicParse_viewBoxWithoutSize(t *testing.T) {
	doc := []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="10 20 100 50">` +
		`<path d="M10 20L110 70H60V30z"/></svg>`)
	sig, err := gofpdf.SVGBasicParse(doc)
	if err != nil {
		t.Fatalf("SVGBasicParse: %v", err)
	}
	if sig.Wd != 100 || sig.Ht != 50 {
		t.Errorf("got size %g x %g, want 100 x 50", sig.Wd, sig.Ht)
	}
	if len(sig.Segments) != 1 {
		t.Fatalf("got %d paths, want 1", len(sig.Segments))
	}
	checkSegments(t, sig.Segments[0], []svgSeg{
		{'M', []float64{0, 0}},
		{'L', []float64{100, 50}},
		{'H', []float64{50}},
		{'V', []float64{10}},
		{'Z', nil},
	})
}

// With width, height and viewBox, the box is stretched to the size: the path
// is in the units of the viewBox, 100 x 50 from (10,20), and the image is
// 30 x 40, so the coordinates are scaled by 0.3 and 0.8.
func TestSVGBasicParse_sizeAndViewBox(t *testing.T) {
	doc := []byte(`<svg width="30" height="40" viewBox="10 20 100 50"><path d="M10 20L15 25H110V70"/></svg>`)
	sig, err := gofpdf.SVGBasicParse(doc)
	if err != nil {
		t.Fatalf("SVGBasicParse: %v", err)
	}
	if sig.Wd != 30 || sig.Ht != 40 {
		t.Errorf("got size %g x %g, want 30 x 40", sig.Wd, sig.Ht)
	}
	checkSegments(t, sig.Segments[0], []svgSeg{
		{'M', []float64{0, 0}},
		{'L', []float64{1.5, 4}},
		{'H', []float64{30}},
		{'V', []float64{40}},
	})
}

// The usual icon: drawn on a grid of 24 and shown at another size.
func TestSVGBasicParse_iconScaledByItsSize(t *testing.T) {
	doc := []byte(`<svg width="96" height="96" viewBox="0 0 24 24"><path d="M12 2C6 2 2 6 2 12"/></svg>`)
	sig, err := gofpdf.SVGBasicParse(doc)
	if err != nil {
		t.Fatalf("SVGBasicParse: %v", err)
	}
	checkSegments(t, sig.Segments[0], []svgSeg{
		{'M', []float64{48, 8}},
		{'C', []float64{24, 8, 8, 24, 8, 48}},
	})
}

func TestSVGBasicParse_sizeWithUnits(t *testing.T) {
	tests := []struct {
		name   string
		doc    string
		wd, ht float64
	}{
		{"pixels", `<svg width="24px" height="12px"><path d="M0 0L1 1"/></svg>`, 24, 12},
		// A size relative to what surrounds the image says nothing by itself:
		// the viewBox is used
		{"percentages", `<svg width="100%" height="100%" viewBox="0 0 64 32"><path d="M0 0L1 1"/></svg>`, 64, 32},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sig, err := gofpdf.SVGBasicParse([]byte(tc.doc))
			if err != nil {
				t.Fatalf("SVGBasicParse: %v", err)
			}
			if sig.Wd != tc.wd || sig.Ht != tc.ht {
				t.Errorf("got size %g x %g, want %g x %g", sig.Wd, sig.Ht, tc.wd, tc.ht)
			}
		})
	}
}

// Radii so small or so large that the arc cannot be computed leave a
// straight line to its end point.
func TestSVGBasicParse_arcWithExtremeRadii(t *testing.T) {
	for _, d := range []string{"M0 0A1e-200 1e-200 0 0 1 10 10", "M0 0A1e200 1e200 0 0 1 10 10"} {
		checkSegments(t, parseOnePath(t, svgDoc(d)), []svgSeg{
			{'M', []float64{0, 0}},
			{'L', []float64{10, 10}},
		})
	}
}

func TestSVGBasicParse_withoutSizeOrViewBox(t *testing.T) {
	if _, err := gofpdf.SVGBasicParse([]byte(`<svg><path d="M0 0L1 1"/></svg>`)); err == nil {
		t.Error("expected an error for a document without size")
	}
}

// pageContent renders the document and returns the content stream of its only
// page.
func pageContent(t *testing.T, pdf *gofpdf.Fpdf) string {
	t.Helper()
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		t.Fatalf("Output: %v", err)
	}
	out := buf.String()
	start := strings.Index(out, "stream\n")
	end := strings.Index(out, "\nendstream")
	if start < 0 || end < start {
		t.Fatalf("no content stream in the document")
	}
	return out[start+len("stream\n") : end]
}

// newPointPage returns a document with one 100 x 100 point page without
// compression, so that the drawing operators can be read as text.
func newPointPage() *gofpdf.Fpdf {
	pdf := gofpdf.NewCustom(&gofpdf.InitType{
		UnitStr: "pt",
		Size:    gofpdf.SizeType{Wd: 100, Ht: 100},
	})
	pdf.SetCompression(false)
	pdf.AddPage()
	return pdf
}

// SVGBasicDraw builds each path as a single PDF path, so that it can be
// filled: a square with a square hole is one path with two subpaths.
func TestSVGBasicDraw_fillsEachPathAsOne(t *testing.T) {
	sig, err := gofpdf.SVGBasicParse(svgDoc("M0 0H10V10H0zM2 2V8H8V2z"))
	if err != nil {
		t.Fatalf("SVGBasicParse: %v", err)
	}
	pdf := newPointPage()
	pdf.SetXY(20, 30)
	pdf.SVGBasicDraw(&sig, 2, "F")
	if pdf.Err() {
		t.Fatalf("SVGBasicDraw: %v", pdf.Error())
	}
	content := pageContent(t, pdf)
	// Origin (20,30) on a page 100 points high, scale 2: the point (x,y) of
	// the path lands at (20+2x, 100-30-2y).
	want := "20.00 70.00 m\n40.00 70.00 l\n40.00 50.00 l\n20.00 50.00 l\nh\n" +
		"24.00 66.00 m\n24.00 54.00 l\n36.00 54.00 l\n36.00 66.00 l\nh\nf\n"
	if !strings.Contains(content, want) {
		t.Errorf("content stream:\n%s\ndoes not contain the filled path:\n%s", content, want)
	}
}

func TestSVGBasicDraw_curves(t *testing.T) {
	sig, err := gofpdf.SVGBasicParse(svgDoc("M0 0C1 2 3 4 5 6Q7 8 9 10"))
	if err != nil {
		t.Fatalf("SVGBasicParse: %v", err)
	}
	pdf := newPointPage()
	pdf.SetXY(0, 0)
	pdf.SVGBasicDraw(&sig, 1, "D")
	if pdf.Err() {
		t.Fatalf("SVGBasicDraw: %v", pdf.Error())
	}
	content := pageContent(t, pdf)
	// The quadratic curve from (5,6) with control point (7,8) is written as
	// the equivalent cubic: its control points are one third of the way from
	// the control point to each end.
	want := "0.00 100.00 m\n" +
		"1.00000 98.00000 3.00000 96.00000 5.00000 94.00000 c\n" +
		"6.33333 92.66667 7.66667 91.33333 9.00000 90.00000 c\n" +
		"S\n"
	if !strings.Contains(content, want) {
		t.Errorf("content stream:\n%s\ndoes not contain the stroked path:\n%s", content, want)
	}
}

// Each path element is painted on its own, as SVG does.
func TestSVGBasicDraw_paintsEveryPath(t *testing.T) {
	doc := []byte(`<svg width="10" height="10"><path d="M0 0H1V1z"/><path d="M5 5H6V6z"/></svg>`)
	sig, err := gofpdf.SVGBasicParse(doc)
	if err != nil {
		t.Fatalf("SVGBasicParse: %v", err)
	}
	pdf := newPointPage()
	pdf.SetXY(0, 0)
	pdf.SVGBasicDraw(&sig, 1, "F")
	content := pageContent(t, pdf)
	want := "0.00 100.00 m\n1.00 100.00 l\n1.00 99.00 l\nh\nf\n" +
		"5.00 95.00 m\n6.00 95.00 l\n6.00 94.00 l\nh\nf\n"
	if !strings.Contains(content, want) {
		t.Errorf("content stream:\n%s\ndoes not contain both filled paths:\n%s", content, want)
	}
}

// The current position is the origin of the image and drawing does not move
// it, so that what follows can be placed from the same point.
func TestSVGBasicDraw_keepsPosition(t *testing.T) {
	sig, err := gofpdf.SVGBasicParse(svgDoc("M0 0L10 10C1 2 3 4 5 6z"))
	if err != nil {
		t.Fatalf("SVGBasicParse: %v", err)
	}
	pdf := newPointPage()
	pdf.SetXY(20, 30)
	pdf.SVGBasicDraw(&sig, 2, "F")
	if x, y := pdf.GetXY(); x != 20 || y != 30 {
		t.Errorf("got position (%g, %g), want (20, 30)", x, y)
	}
}

// A PDF path has to begin with a move. A path that draws before moving
// starts at the origin of the image, and one that keeps drawing after closing
// a subpath starts a new one where the closed one began.
func TestSVGBasicDraw_pathThatDrawsBeforeMoving(t *testing.T) {
	sig, err := gofpdf.SVGBasicParse(svgDoc("L10 0V10zL5 5"))
	if err != nil {
		t.Fatalf("SVGBasicParse: %v", err)
	}
	pdf := newPointPage()
	pdf.SetXY(20, 30)
	pdf.SVGBasicDraw(&sig, 1, "D")
	content := pageContent(t, pdf)
	want := "20.00 70.00 m\n30.00 70.00 l\n30.00 60.00 l\nh\n20.00 70.00 m\n25.00 65.00 l\nS\n"
	if !strings.Contains(content, want) {
		t.Errorf("content stream:\n%s\ndoes not contain:\n%s", content, want)
	}
}
