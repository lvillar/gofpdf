package gofpdf_test

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"github.com/lvillar/gofpdf"
)

// newHelveticaPage returns a document with one 100 x 100 point page without
// compression and Helvetica at 10 points as the current font. In this font
// "A" and "V" are 667 thousandths of the font size wide (6.67 points) and the
// space is 278.
func newHelveticaPage() *gofpdf.Fpdf {
	pdf := newPointPage()
	pdf.SetFont("Helvetica", "", 10)
	pdf.SetCellMargin(0)
	pdf.SetMargins(0, 0, 0)
	pdf.SetAutoPageBreak(false, 0)
	return pdf
}

// pageContents renders the document and returns the content stream of each
// of its pages. It expects a document without embedded fonts or images, in
// which the pages are the only streams.
func pageContents(t *testing.T, pdf *gofpdf.Fpdf) (pages []string) {
	t.Helper()
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		t.Fatalf("Output: %v", err)
	}
	for _, part := range strings.Split(buf.String(), "\nstream\n")[1:] {
		if end := strings.Index(part, "\nendstream"); end >= 0 {
			pages = append(pages, part[:end])
		}
	}
	return
}

func near(got, want float64) bool {
	return math.Abs(got-want) < 1e-6
}

func TestSetCharSpacing_writesTheSpacing(t *testing.T) {
	pdf := newHelveticaPage()
	pdf.SetCharSpacing(0.5)
	pdf.Text(10, 10, "AV")
	content := pageContent(t, pdf)
	want := "0.500 Tc\nBT 10.00 90.00 Td (AV) Tj ET"
	if !strings.Contains(content, want) {
		t.Errorf("content stream:\n%s\ndoes not contain:\n%s", content, want)
	}
}

// The unit is the one of the document: 1 mm is 2.835 points.
func TestSetCharSpacing_usesTheUnitOfTheDocument(t *testing.T) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetCompression(false)
	pdf.AddPage()
	pdf.SetCharSpacing(-1)
	content := pageContent(t, pdf)
	if want := "-2.835 Tc\n"; !strings.Contains(content, want) {
		t.Errorf("content stream:\n%s\ndoes not contain:\n%s", content, want)
	}
}

// The spacing is added after every character, the last one included, which
// is how the viewer advances and how a browser measures letter-spacing.
func TestSetCharSpacing_stringWidth(t *testing.T) {
	pdf := newHelveticaPage()
	if got := pdf.GetStringWidth("AV"); !near(got, 13.34) {
		t.Fatalf("without spacing: got width %g, want 13.34", got)
	}
	pdf.SetCharSpacing(0.5)
	if got := pdf.GetStringWidth("AV"); !near(got, 14.34) {
		t.Errorf("got width %g, want 14.34", got)
	}
	if got := pdf.GetStringSymbolWidth("AV"); got != 1434 {
		t.Errorf("got width %d in glyph units, want 1434", got)
	}
	pdf.SetCharSpacing(-0.25)
	if got := pdf.GetStringWidth("AV"); !near(got, 12.84) {
		t.Errorf("negative spacing: got width %g, want 12.84", got)
	}
	pdf.SetCharSpacing(0)
	if got := pdf.GetStringWidth("AV"); !near(got, 13.34) {
		t.Errorf("after setting it back to zero: got width %g, want 13.34", got)
	}
}

// A character of a UTF-8 font can take several bytes: the spacing is counted
// by character.
func TestSetCharSpacing_stringWidthUTF8(t *testing.T) {
	pdf := newKerningPage(t)
	pdf.SetCharSpacing(0.5)
	// "Á" and "V" are 615 thousandths of 10 points wide each
	if got := pdf.GetStringWidth("ÁV"); !near(got, 13.3) {
		t.Errorf("got width %g, want 13.3", got)
	}
}

// Each page starts with the default spacing of the PDF format, so the one in
// effect is written again.
func TestSetCharSpacing_keptOnNewPages(t *testing.T) {
	pdf := newHelveticaPage()
	pdf.SetCharSpacing(0.5)
	pdf.AddPage()
	pdf.Text(10, 10, "AV")
	pages := pageContents(t, pdf)
	if len(pages) != 2 {
		t.Fatalf("got %d pages, want 2", len(pages))
	}
	if want := "0.500 Tc\n"; !strings.Contains(pages[1], want) {
		t.Errorf("second page:\n%s\ndoes not contain:\n%s", pages[1], want)
	}
}

func TestSetCharSpacing_beforeTheFirstPage(t *testing.T) {
	pdf := gofpdf.NewCustom(&gofpdf.InitType{
		UnitStr: "pt",
		Size:    gofpdf.SizeType{Wd: 100, Ht: 100},
	})
	pdf.SetCompression(false)
	pdf.SetCharSpacing(0.5)
	pdf.AddPage()
	if pdf.Err() {
		t.Fatalf("AddPage: %v", pdf.Error())
	}
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		t.Fatalf("Output: %v", err)
	}
	out := buf.String()
	first := strings.Index(out, "stream\n")
	if strings.Contains(out[:first], " Tc") {
		t.Errorf("the spacing was written outside of a page:\n%s", out[:first])
	}
	if want := "0.500 Tc\n"; !strings.Contains(out[first:], want) {
		t.Errorf("the first page does not contain:\n%s", want)
	}
}

// Pages without spacing are not touched.
func TestSetCharSpacing_notWrittenUnlessSet(t *testing.T) {
	pdf := newHelveticaPage()
	pdf.Text(10, 10, "AV")
	pdf.AddPage()
	for j, page := range pageContents(t, pdf) {
		if strings.Contains(page, "Tc") {
			t.Errorf("page %d sets a character spacing:\n%s", j+1, page)
		}
	}
}

// Right-aligned in a cell 50 points wide, 14.34 points of text start at
// 35.66.
func TestSetCharSpacing_cellAlignment(t *testing.T) {
	pdf := newHelveticaPage()
	pdf.SetCharSpacing(0.5)
	pdf.SetXY(0, 0)
	pdf.CellFormat(50, 10, "AV", "", 0, "R", false, 0, "")
	content := pageContent(t, pdf)
	if want := "BT 35.66 92.00 Td (AV)Tj ET"; !strings.Contains(content, want) {
		t.Errorf("content stream:\n%s\ndoes not contain:\n%s", content, want)
	}
}

// "AAAA AAAA" is 56.14 points wide and fits in 60 points. With 1 point after
// each of its 9 characters it is 65.14 wide and takes two lines.
func TestSetCharSpacing_lineBreaking(t *testing.T) {
	const txt = "AAAA AAAA"
	t.Run("SplitText", func(t *testing.T) {
		pdf := newHelveticaPage()
		if got := pdf.SplitText(txt, 60); len(got) != 1 {
			t.Fatalf("without spacing: got %q, want one line", got)
		}
		pdf.SetCharSpacing(1)
		got := pdf.SplitText(txt, 60)
		if len(got) != 2 || got[0] != "AAAA" || got[1] != "AAAA" {
			t.Errorf("got %q, want [AAAA AAAA]", got)
		}
	})
	t.Run("SplitLines", func(t *testing.T) {
		pdf := newHelveticaPage()
		if got := pdf.SplitLines([]byte(txt), 60); len(got) != 1 {
			t.Fatalf("without spacing: got %q, want one line", got)
		}
		pdf.SetCharSpacing(1)
		got := pdf.SplitLines([]byte(txt), 60)
		if len(got) != 2 || string(got[0]) != "AAAA" || string(got[1]) != "AAAA" {
			t.Errorf("got %q, want [AAAA AAAA]", got)
		}
	})
	t.Run("MultiCell", func(t *testing.T) {
		pdf := newHelveticaPage()
		pdf.SetXY(0, 0)
		pdf.MultiCell(60, 10, txt, "", "L", false)
		if got := pdf.GetY(); got != 10 {
			t.Fatalf("without spacing: the text ends at %g, want 10 (one line)", got)
		}
		pdf.SetCharSpacing(1)
		pdf.SetXY(0, 0)
		pdf.MultiCell(60, 10, txt, "", "L", false)
		if got := pdf.GetY(); got != 20 {
			t.Errorf("the text ends at %g, want 20 (two lines)", got)
		}
	})
	t.Run("Write", func(t *testing.T) {
		pdf := newHelveticaPage()
		pdf.SetRightMargin(40) // lines 60 points wide
		pdf.SetXY(0, 0)
		pdf.Write(10, txt)
		if got := pdf.GetY(); got != 0 {
			t.Fatalf("without spacing: the text ends at %g, want 0 (first line)", got)
		}
		pdf.SetCharSpacing(1)
		pdf.SetXY(0, 0)
		pdf.Write(10, txt)
		if got := pdf.GetY(); got != 10 {
			t.Errorf("the text ends at %g, want 10 (second line)", got)
		}
	})
}

// "AVAVAVAV" is 49.2 points wide in DejaVu Sans Condensed at 10 points, and
// 44.72 once its seven pairs are kerned: only then it fits in 46 points.
func TestSetKerning_lineBreaking(t *testing.T) {
	const txt = "AVAVAVAV"
	t.Run("SplitText", func(t *testing.T) {
		pdf := newKerningPage(t)
		got := pdf.SplitText(txt, 46)
		if len(got) != 2 || got[0] != "AVAVAVA" || got[1] != "V" {
			t.Fatalf("without kerning: got %q, want [AVAVAVA V]", got)
		}
		pdf.SetKerning(true)
		if got := pdf.SplitText(txt, 46); len(got) != 1 || got[0] != txt {
			t.Errorf("got %q, want one line", got)
		}
	})
	t.Run("MultiCell", func(t *testing.T) {
		pdf := newKerningPage(t)
		pdf.SetXY(0, 0)
		pdf.MultiCell(46, 10, txt, "", "L", false)
		if got := pdf.GetY(); got != 20 {
			t.Fatalf("without kerning: the text ends at %g, want 20 (two lines)", got)
		}
		pdf.SetKerning(true)
		pdf.SetXY(0, 0)
		pdf.MultiCell(46, 10, txt, "", "L", false)
		if got := pdf.GetY(); got != 10 {
			t.Errorf("the text ends at %g, want 10 (one line)", got)
		}
	})
	t.Run("Write", func(t *testing.T) {
		pdf := newKerningPage(t)
		pdf.SetMargins(0, 0, 54) // lines 46 points wide
		pdf.SetXY(0, 0)
		pdf.Write(10, txt)
		if got := pdf.GetY(); got != 10 {
			t.Fatalf("without kerning: the text ends at %g, want 10 (second line)", got)
		}
		pdf.SetKerning(true)
		pdf.SetXY(0, 0)
		pdf.Write(10, txt)
		if got := pdf.GetY(); got != 0 {
			t.Errorf("the text ends at %g, want 0 (first line)", got)
		}
	})
}

// A template is drawn with the spacing it was built with, not with the one
// of the page that uses it.
func TestSetCharSpacing_notInheritedByTemplates(t *testing.T) {
	pdf := newHelveticaPage()
	tpl := pdf.CreateTemplate(func(tpl *gofpdf.Tpl) {
		tpl.SetFont("Helvetica", "", 10)
		tpl.Text(10, 10, "LETTERHEAD")
	})
	pdf.UseTemplate(tpl)
	pdf.SetCharSpacing(3)
	pdf.UseTemplate(tpl)
	if pdf.Err() {
		t.Fatalf("UseTemplate: %v", pdf.Error())
	}
	content := pageContent(t, pdf)
	uses := strings.Split(content, " cm\n")
	if len(uses) != 3 {
		t.Fatalf("the template is used %d times, want 2:\n%s", len(uses)-1, content)
	}
	if strings.HasPrefix(uses[1], "0 Tc\n") {
		t.Errorf("without spacing nothing has to be reset:\n%s", content)
	}
	if !strings.HasPrefix(uses[2], "0 Tc\n/TPL") {
		t.Errorf("the spacing is not reset before the template is drawn:\n%s", content)
	}
}

func TestSetCharSpacing_notInheritedByImportedPages(t *testing.T) {
	pdf := newHelveticaPage()
	pdf.SetCharSpacing(3)
	pdf.UseImportedTemplate("/GOFPDITPL1", 1, 1, 0, 0)
	content := pageContent(t, pdf)
	if want := " 0 Tc q "; !strings.Contains(content, want) {
		t.Errorf("the spacing is not reset before the imported page is drawn:\n%s", content)
	}
}

// The header and the footer are not part of the text that is being written:
// they are drawn without its spacing, which is back in effect after the
// header of the new page.
func TestSetCharSpacing_headerAndFooter(t *testing.T) {
	pdf := newHelveticaPage()
	pdf.SetHeaderFunc(func() { pdf.Text(10, 10, "HEAD") })
	pdf.SetFooterFunc(func() { pdf.Text(10, 90, "FOOT") })
	pdf.SetCharSpacing(2)
	pdf.Text(10, 50, "BODY1")
	pdf.AddPage()
	if got := pdf.GetStringWidth("AV"); !near(got, 17.34) {
		t.Errorf("after the page break: got width %g, want 17.34 (the spacing is kept)", got)
	}
	pdf.Text(10, 50, "BODY2")
	pages := pageContents(t, pdf)
	if len(pages) != 2 {
		t.Fatalf("got %d pages, want 2", len(pages))
	}
	order := func(page string, parts ...string) bool {
		pos := 0
		for _, part := range parts {
			next := strings.Index(page[pos:], part)
			if next < 0 {
				return false
			}
			pos += next + len(part)
		}
		return true
	}
	if !order(pages[0], "2.000 Tc", "(BODY1)", "0 Tc", "(FOOT)") {
		t.Errorf("first page: the footer is not written without spacing:\n%s", pages[0])
	}
	if !order(pages[1], "(HEAD)", "2.000 Tc", "(BODY2)", "0 Tc", "(FOOT)") {
		t.Errorf("second page: expected the header, the spacing, the text and the footer without spacing:\n%s", pages[1])
	}
	if strings.Index(pages[1], "Tc") < strings.Index(pages[1], "(HEAD)") {
		t.Errorf("second page: the header is written with spacing:\n%s", pages[1])
	}
}

// A header that sets a spacing of its own does not change the one of the
// text.
func TestSetCharSpacing_headerWithItsOwnSpacing(t *testing.T) {
	pdf := newHelveticaPage()
	pdf.SetHeaderFunc(func() {
		pdf.SetCharSpacing(1)
		pdf.Text(10, 10, "HEAD")
	})
	pdf.AddPage()
	if got := pdf.GetStringWidth("AV"); !near(got, 13.34) {
		t.Errorf("got width %g, want 13.34 (no spacing)", got)
	}
	pdf.Text(10, 50, "BODY")
	pages := pageContents(t, pdf)
	if !strings.Contains(pages[1], "(HEAD) Tj ET\n0.000 Tc\n") {
		t.Errorf("the spacing of the header is still in effect after it:\n%s", pages[1])
	}
}

// When a transformation or a clipping area ends the viewer goes back to the
// spacing it had when it began, so the one set meanwhile is written again.
func TestSetCharSpacing_changedInsideTransformOrClip(t *testing.T) {
	t.Run("transformation", func(t *testing.T) {
		pdf := newHelveticaPage()
		pdf.SetCharSpacing(2)
		pdf.TransformBegin()
		pdf.SetCharSpacing(0)
		pdf.TransformEnd()
		content := pageContent(t, pdf)
		if want := "Q\n0.000 Tc\n"; !strings.Contains(content, want) {
			t.Errorf("content stream:\n%s\ndoes not contain:\n%s", content, want)
		}
	})
	t.Run("clipping", func(t *testing.T) {
		pdf := newHelveticaPage()
		pdf.ClipRect(0, 0, 50, 50, false)
		pdf.SetCharSpacing(1.5)
		pdf.ClipEnd()
		content := pageContent(t, pdf)
		if want := "Q\n1.500 Tc\n"; !strings.Contains(content, want) {
			t.Errorf("content stream:\n%s\ndoes not contain:\n%s", content, want)
		}
	})
	t.Run("not changed", func(t *testing.T) {
		pdf := newHelveticaPage()
		pdf.SetCharSpacing(2)
		pdf.TransformBegin()
		pdf.Text(10, 10, "AV")
		pdf.TransformEnd()
		content := pageContent(t, pdf)
		if strings.Count(content, "Tc") != 1 {
			t.Errorf("the spacing is written more than once:\n%s", content)
		}
	})
}

// A spacing too small to count character by character still adds up: 0.004
// points are 0.4 thousandths of 10 points, 40 in a hundred characters. A
// hundred "A" are 66700 wide and 66740 with the spacing, which is more than
// the 66720 of the line.
func TestSetCharSpacing_smallSpacingAddsUpWhenBreakingLines(t *testing.T) {
	pdf := gofpdf.NewCustom(&gofpdf.InitType{
		UnitStr: "pt",
		Size:    gofpdf.SizeType{Wd: 1000, Ht: 100},
	})
	pdf.AddPage()
	pdf.SetFont("Helvetica", "", 10)
	pdf.SetCellMargin(0)
	txt := strings.Repeat("A", 100)
	if got := pdf.SplitText(txt, 667.2); len(got) != 1 {
		t.Fatalf("without spacing: got %d lines, want 1", len(got))
	}
	pdf.SetCharSpacing(0.004)
	got := pdf.SplitText(txt, 667.2)
	if len(got) != 2 || len(got[0]) != 99 || len(got[1]) != 1 {
		t.Errorf("got %d lines, want one of 99 characters and one of 1", len(got))
	}
}
