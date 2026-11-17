package gofpdf_test

import (
	"github.com/lvillar/gofpdf"
	"github.com/lvillar/gofpdf/internal/example"
)

// ExampleFpdf_SetKerning demonstrates pair kerning and letter spacing: the
// same text is written as the library does by default, with the kerning
// stored in the font and with the letters spaced out.
func ExampleFpdf_SetKerning() {
	const txt = "Type AVAILABLE Today. Wavy Yacht, To."
	pdf := gofpdf.New("P", "mm", "A4", example.FontDir())
	pdf.AddUTF8Font("dejavu", "", "DejaVuSansCondensed.ttf")
	pdf.AddPage()
	// line writes a label and, below it, the sample text with the specified
	// letter spacing.
	line := func(label string, spacing float64) {
		pdf.SetFont("dejavu", "", 9)
		pdf.CellFormat(0, 6, label, "", 1, "L", false, 0, "")
		pdf.SetFont("dejavu", "", 22)
		pdf.SetCharSpacing(spacing)
		pdf.CellFormat(0, 12, txt, "", 1, "L", false, 0, "")
		pdf.SetCharSpacing(0)
		pdf.Ln(4)
	}

	line("Default: every character takes its own width", 0)

	pdf.SetKerning(true)
	line("SetKerning(true): pairs such as AV, Ty, Wa and To are brought together", 0)
	line("SetCharSpacing(0.6): 0.6 mm more after each character", 0.6)
	line("SetCharSpacing(-0.25): a quarter of a millimeter less", -0.25)

	// Measuring takes kerning and spacing into account, so alignment and
	// justification keep working.
	pdf.SetFont("dejavu", "", 12)
	pdf.MultiCell(90, 6, "Justified text with kerning: "+txt+" "+txt, "1", "J", false)

	fileStr := example.Filename("Fpdf_SetKerning")
	err := pdf.OutputFileAndClose(fileStr)
	example.SummaryCompare(err, fileStr)
	// Output:
	// Successfully generated pdf/Fpdf_SetKerning.pdf
}

// ExampleFpdf_SVGBasicDraw demonstrates how to fill the paths of an SVG
// image, a logo for instance. The image of this example has a ring, made of
// two arcs each way so that the inner circle is a hole, and a shape drawn
// with smooth curves.
func ExampleFpdf_SVGBasicDraw() {
	const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 220 100">
<path d="M50 5a45 45 0 010 90a45 45 0 010-90zm0 20a25 25 0 000 50a25 25 0 000-50z"/>
<path d="M120 80c0-40 15-60 30-60s20 20 30 20 15-10 30-10v50z"/>
</svg>`
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	sig, err := gofpdf.SVGBasicParse([]byte(svg))
	if err == nil {
		scale := 110 / sig.Wd // 110 mm wide
		// Filled
		pdf.SetFillColor(36, 49, 101)
		pdf.SetXY(20, 20)
		pdf.SVGBasicDraw(&sig, scale, "F")
		// Outlined, half the size
		pdf.SetDrawColor(36, 49, 101)
		pdf.SetLineWidth(0.3)
		pdf.SetXY(20, 80)
		pdf.SVGBasicDraw(&sig, scale/2, "D")
		// Filled and outlined
		pdf.SetFillColor(220, 226, 240)
		pdf.SetXY(85, 80)
		pdf.SVGBasicDraw(&sig, scale/2, "DF")
	} else {
		pdf.SetError(err)
	}
	fileStr := example.Filename("Fpdf_SVGBasicDraw")
	err = pdf.OutputFileAndClose(fileStr)
	example.SummaryCompare(err, fileStr)
	// Output:
	// Successfully generated pdf/Fpdf_SVGBasicDraw.pdf
}
