package gofpdf_test

import (
	"strings"
	"testing"

	"github.com/lvillar/gofpdf"
	"github.com/lvillar/gofpdf/internal/example"
)

// newKerningPage returns a document with one 100 x 100 point page without
// compression and DejaVu Sans Condensed at 10 points as the current font.
//
// The kerning expected by these tests was measured with a browser (HarfBuzz)
// and is expressed in thousandths of the font size. In this font "A" and "V"
// are 615 wide each, the space is 286 wide and the pairs "AV" and "VA" are
// tightened by 64.
func newKerningPage(t *testing.T) *gofpdf.Fpdf {
	t.Helper()
	pdf := gofpdf.NewCustom(&gofpdf.InitType{
		UnitStr:    "pt",
		Size:       gofpdf.SizeType{Wd: 100, Ht: 100},
		FontDirStr: example.FontDir(),
	})
	pdf.SetCompression(false)
	pdf.AddUTF8Font("dejavu", "", "DejaVuSansCondensed.ttf")
	pdf.AddPage()
	pdf.SetFont("dejavu", "", 10)
	pdf.SetCellMargin(0)
	if pdf.Err() {
		t.Fatalf("setting up the document: %v", pdf.Error())
	}
	return pdf
}

func TestSetKerning_stringWidth(t *testing.T) {
	tests := []struct {
		str  string
		want int // kerning of the whole string
	}{
		{"AV", -64},
		{"VA", -64},
		{"To", -170},
		{"LT", -138},
		{"P.", -156},
		{"r.", -92},
		{"ÁV", -64},     // glyphs outside ASCII are kerned too
		{"HH", 0},       // a pair without kerning
		{"AVA", -128},   // every pair counts: AV and VA
		{"AV AV", -128}, // the space does not join the pairs around it
		{"A", 0},
		{"", 0},
	}
	for _, tc := range tests {
		t.Run(tc.str, func(t *testing.T) {
			pdf := newKerningPage(t)
			plain := pdf.GetStringSymbolWidth(tc.str)
			pdf.SetKerning(true)
			kerned := pdf.GetStringSymbolWidth(tc.str)
			if kerned-plain != tc.want {
				t.Errorf("got kerning %d, want %d (width %d without kerning, %d with it)",
					kerned-plain, tc.want, plain, kerned)
			}
		})
	}
}

// Kerning changes where every piece of text ends, so documents that do not
// ask for it must keep their layout.
func TestSetKerning_offUnlessEnabled(t *testing.T) {
	pdf := newKerningPage(t)
	if got := pdf.GetStringSymbolWidth("AV"); got != 1230 {
		t.Errorf("by default: got width %d, want 1230", got)
	}
	pdf.SetKerning(true)
	pdf.SetKerning(false)
	if got := pdf.GetStringSymbolWidth("AV"); got != 1230 {
		t.Errorf("after disabling it: got width %d, want 1230", got)
	}
}

func TestSetKerning_stringWidthInUserUnits(t *testing.T) {
	pdf := newKerningPage(t)
	pdf.SetKerning(true)
	// (615 + 615 - 64) thousandths of 10 points
	if got := pdf.GetStringWidth("AV"); got < 11.6599 || got > 11.6601 {
		t.Errorf("got width %.4f, want 11.66", got)
	}
}

func TestSetKerning_text(t *testing.T) {
	pdf := newKerningPage(t)
	pdf.SetKerning(true)
	pdf.Text(10, 10, "AVA")
	content := pageContent(t, pdf)
	// A positive number in the array moves the next glyph to the left
	want := "BT 10.00 90.00 Td [(\x00A) 64 (\x00V) 64 (\x00A)] TJ ET"
	if !strings.Contains(content, want) {
		t.Errorf("content stream:\n%q\ndoes not contain:\n%q", content, want)
	}
}

func TestSetKerning_textWithoutKernedPairs(t *testing.T) {
	pdf := newKerningPage(t)
	pdf.SetKerning(true)
	pdf.Text(10, 10, "HH (1)")
	content := pageContent(t, pdf)
	want := "BT 10.00 90.00 Td [(\x00H\x00H\x00 \x00\\(\x001\x00\\))] TJ ET"
	if !strings.Contains(content, want) {
		t.Errorf("content stream:\n%q\ndoes not contain:\n%q", content, want)
	}
}

// The kerned text is what gets aligned: right-aligned in a cell 50 points
// wide, 11.66 points of text start at 38.34.
func TestSetKerning_cellAlignment(t *testing.T) {
	pdf := newKerningPage(t)
	pdf.SetKerning(true)
	pdf.SetXY(0, 0)
	pdf.CellFormat(50, 10, "AV", "", 0, "R", false, 0, "")
	content := pageContent(t, pdf)
	want := "BT 38.34 92.00 Td [(\x00A) 64 (\x00V)] TJ ET"
	if !strings.Contains(content, want) {
		t.Errorf("content stream:\n%q\ndoes not contain:\n%q", content, want)
	}
}

// Justified text spreads over its spaces what is left of the line once the
// kerned text is measured: 10000 - (4*615 + 286 - 2*64) = 7382 thousandths.
func TestSetKerning_justifiedCell(t *testing.T) {
	pdf := newKerningPage(t)
	pdf.SetKerning(true)
	pdf.SetXY(0, 0)
	pdf.CellFormat(100, 10, "AV AV", "", 0, "J", false, 0, "")
	content := pageContent(t, pdf)
	want := "[(\x00A) 64 (\x00V) -7382 (\x00 \x00A) 64 (\x00V)] TJ ET"
	if !strings.Contains(content, want) {
		t.Errorf("content stream:\n%q\ndoes not contain:\n%q", content, want)
	}
}

// Underlines are as long as the kerned text.
func TestSetKerning_underline(t *testing.T) {
	pdf := newKerningPage(t)
	pdf.SetKerning(true)
	pdf.SetFont("dejavu", "U", 10)
	pdf.Text(10, 10, "AV")
	content := pageContent(t, pdf)
	if !strings.Contains(content, " 11.66 ") {
		t.Errorf("content stream:\n%q\nhas no underline 11.66 points long", content)
	}
}

// A font without kerning information is used as it is.
func TestSetKerning_fontWithoutKerning(t *testing.T) {
	pdf := gofpdf.NewCustom(&gofpdf.InitType{
		UnitStr:    "pt",
		Size:       gofpdf.SizeType{Wd: 100, Ht: 100},
		FontDirStr: example.FontDir(),
	})
	pdf.SetCompression(false)
	pdf.AddUTF8Font("calligra", "", "calligra.ttf")
	pdf.AddPage()
	pdf.SetFont("calligra", "", 10)
	plain := pdf.GetStringSymbolWidth("AV")
	pdf.SetKerning(true)
	if got := pdf.GetStringSymbolWidth("AV"); got != plain {
		t.Errorf("got width %d, want %d", got, plain)
	}
	pdf.Text(10, 10, "AV")
	if pdf.Err() {
		t.Fatalf("Text: %v", pdf.Error())
	}
	content := pageContent(t, pdf)
	if want := "[(\x00A\x00V)] TJ"; !strings.Contains(content, want) {
		t.Errorf("content stream:\n%q\ndoes not contain:\n%q", content, want)
	}
}

// Kerning is read from the files of UTF-8 fonts: the core fonts and the fonts
// described by JSON files have none, and their text is written as always.
func TestSetKerning_fontThatIsNotUTF8(t *testing.T) {
	pdf := gofpdf.NewCustom(&gofpdf.InitType{
		UnitStr: "pt",
		Size:    gofpdf.SizeType{Wd: 100, Ht: 100},
	})
	pdf.SetCompression(false)
	pdf.AddPage()
	pdf.SetFont("Helvetica", "", 10)
	plain := pdf.GetStringSymbolWidth("AV")
	pdf.SetKerning(true)
	if got := pdf.GetStringSymbolWidth("AV"); got != plain {
		t.Errorf("got width %d, want %d", got, plain)
	}
	pdf.Text(10, 10, "AV")
	content := pageContent(t, pdf)
	if want := "BT 10.00 90.00 Td (AV) Tj ET"; !strings.Contains(content, want) {
		t.Errorf("content stream:\n%q\ndoes not contain:\n%q", content, want)
	}
}

// A line that starts with a blank has one more space to stretch. "H" is 677
// wide: 10000 - (4*677 + 2*286) leaves 6720 for the two spaces.
func TestSetKerning_justifiedCellStartingWithSpace(t *testing.T) {
	pdf := newKerningPage(t)
	pdf.SetKerning(true)
	pdf.SetXY(0, 0)
	pdf.CellFormat(100, 10, " HH HH", "", 0, "J", false, 0, "")
	content := pageContent(t, pdf)
	want := "[-3360 (\x00 \x00H\x00H) -3360 (\x00 \x00H\x00H)] TJ ET"
	if !strings.Contains(content, want) {
		t.Errorf("content stream:\n%q\ndoes not contain:\n%q", content, want)
	}
}

// Right-to-left text is written reversed, which would pair up the wrong
// characters: it is left without kerning.
func TestSetKerning_notAppliedToRightToLeftText(t *testing.T) {
	pdf := newKerningPage(t)
	pdf.SetKerning(true)
	pdf.RTL()
	if got := pdf.GetStringSymbolWidth("To"); got != 1099 {
		t.Errorf("got width %d, want 1099 (the width without kerning)", got)
	}
	pdf.Text(50, 10, "To")
	content := pageContent(t, pdf)
	if want := "[(\x00o\x00T)] TJ"; !strings.Contains(content, want) {
		t.Errorf("content stream:\n%q\ndoes not contain:\n%q", content, want)
	}
}
