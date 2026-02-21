package pageops

import (
	"fmt"
	"io"
	"math"

	gofpdf "github.com/lvillar/gofpdf"
	"github.com/lvillar/gofpdf/contrib/gofpdi"
)

// TextWatermark defines a text-based watermark.
type TextWatermark struct {
	Text     string   // watermark text
	FontSize float64  // font size in points (default: 60)
	Color    RGBColor // text color (default: light gray)
	Opacity  float64  // 0.0 to 1.0 (default: 0.3)
	Angle    float64  // rotation angle in degrees (default: 45)
}

// RGBColor represents an RGB color value.
type RGBColor struct {
	R, G, B int
}

// AddTextWatermark adds a text watermark to all pages of a PDF.
func AddTextWatermark(w io.Writer, inputPath string, wm TextWatermark) error {
	return addTextWatermarkToPages(w, inputPath, wm, nil)
}

// AddTextWatermarkToFile adds a text watermark and saves to a file.
func AddTextWatermarkToFile(inputPath, outputPath string, wm TextWatermark) error {
	return addTextWatermarkToFilePages(inputPath, outputPath, wm, nil)
}

// AddTextWatermarkToPages adds a text watermark to specific pages (1-based).
// If pages is nil, the watermark is applied to all pages.
func AddTextWatermarkToPages(w io.Writer, inputPath string, wm TextWatermark, pages []int) error {
	return addTextWatermarkToPages(w, inputPath, wm, pages)
}

func addTextWatermarkToPages(w io.Writer, inputPath string, wm TextWatermark, pages []int) error {
	pdf, err := buildWatermarkedPDF(inputPath, wm, pages)
	if err != nil {
		return err
	}
	return writePDF(pdf, w)
}

func addTextWatermarkToFilePages(inputPath, outputPath string, wm TextWatermark, pages []int) error {
	pdf, err := buildWatermarkedPDF(inputPath, wm, pages)
	if err != nil {
		return err
	}
	return writePDFToFile(pdf, outputPath)
}

func buildWatermarkedPDF(inputPath string, wm TextWatermark, pages []int) (*gofpdf.Fpdf, error) {
	// Set defaults
	if wm.FontSize == 0 {
		wm.FontSize = 60
	}
	if wm.Opacity == 0 {
		wm.Opacity = 0.3
	}
	if wm.Angle == 0 {
		wm.Angle = 45
	}
	if wm.Color == (RGBColor{}) {
		wm.Color = RGBColor{200, 200, 200}
	}

	pageCount, err := getPageCount(inputPath)
	if err != nil {
		return nil, err
	}

	// Build set of pages to watermark
	watermarkPages := make(map[int]bool)
	if pages == nil {
		for i := 1; i <= pageCount; i++ {
			watermarkPages[i] = true
		}
	} else {
		for _, p := range pages {
			watermarkPages[p] = true
		}
	}

	pdf := gofpdf.New("P", "pt", "A4", "")
	pdf.SetAutoPageBreak(false, 0)
	imp := gofpdi.NewImporter()

	for i := 1; i <= pageCount; i++ {
		tplID, pw, ph := importPage(pdf, imp, inputPath, i)
		if pw == 0 || ph == 0 {
			pw = 595.28
			ph = 841.89
		}

		pdf.AddPageFormat("P", gofpdf.SizeType{Wd: pw, Ht: ph})
		imp.UseImportedTemplate(pdf, tplID, 0, 0, pw, ph)

		// Add watermark overlay if this page is in the set
		if watermarkPages[i] {
			drawTextWatermark(pdf, wm, pw, ph)
		}
	}

	if pdf.Err() {
		return nil, fmt.Errorf("pageops: watermark: %w", pdf.Error())
	}
	return pdf, nil
}

// drawTextWatermark renders the watermark text centered on the current page.
func drawTextWatermark(pdf *gofpdf.Fpdf, wm TextWatermark, pageW, pageH float64) {
	pdf.SetFont("Helvetica", "B", wm.FontSize)
	pdf.SetTextColor(wm.Color.R, wm.Color.G, wm.Color.B)
	pdf.SetAlpha(wm.Opacity, "Normal")

	// Calculate center position
	textW := pdf.GetStringWidth(wm.Text)
	cx := pageW / 2
	cy := pageH / 2

	// Apply rotation around center
	pdf.TransformBegin()
	pdf.TransformRotate(wm.Angle, cx, cy)

	// Position text centered at rotation point
	x := cx - textW/2
	y := cy + wm.FontSize/3 // approximate vertical centering

	pdf.Text(x, y, wm.Text)
	pdf.TransformEnd()

	// Reset alpha
	pdf.SetAlpha(1.0, "Normal")
}

// AddPageNumbers adds page numbers to all pages of a PDF.
func AddPageNumbers(w io.Writer, inputPath string, style PageNumberStyle) error {
	pdf, err := buildPageNumberedPDF(inputPath, style)
	if err != nil {
		return err
	}
	return writePDF(pdf, w)
}

// AddPageNumbersToFile adds page numbers and saves to a file.
func AddPageNumbersToFile(inputPath, outputPath string, style PageNumberStyle) error {
	pdf, err := buildPageNumberedPDF(inputPath, style)
	if err != nil {
		return err
	}
	return writePDFToFile(pdf, outputPath)
}

// PageNumberStyle defines the appearance and position of page numbers.
type PageNumberStyle struct {
	Format   string   // fmt format string, e.g. "Page %d of %d" (receives pageNum, totalPages)
	Position Position // where to place the number (default: BottomCenter)
	FontSize float64  // font size in points (default: 10)
	Color    RGBColor // text color (default: black)
	Margin   float64  // margin from page edge in points (default: 30)
}

func buildPageNumberedPDF(inputPath string, style PageNumberStyle) (*gofpdf.Fpdf, error) {
	// Defaults
	if style.Format == "" {
		style.Format = "Page %d of %d"
	}
	if style.FontSize == 0 {
		style.FontSize = 10
	}
	if style.Margin == 0 {
		style.Margin = 30
	}

	pageCount, err := getPageCount(inputPath)
	if err != nil {
		return nil, err
	}

	pdf := gofpdf.New("P", "pt", "A4", "")
	pdf.SetAutoPageBreak(false, 0)
	imp := gofpdi.NewImporter()

	for i := 1; i <= pageCount; i++ {
		tplID, pw, ph := importPage(pdf, imp, inputPath, i)
		if pw == 0 || ph == 0 {
			pw = 595.28
			ph = 841.89
		}

		pdf.AddPageFormat("P", gofpdf.SizeType{Wd: pw, Ht: ph})
		imp.UseImportedTemplate(pdf, tplID, 0, 0, pw, ph)

		// Draw page number
		text := fmt.Sprintf(style.Format, i, pageCount)
		pdf.SetFont("Helvetica", "", style.FontSize)
		pdf.SetTextColor(style.Color.R, style.Color.G, style.Color.B)

		textW := pdf.GetStringWidth(text)
		x, y := calculatePosition(style.Position, pw, ph, textW, style.FontSize, style.Margin)
		pdf.Text(x, y, text)
	}

	if pdf.Err() {
		return nil, fmt.Errorf("pageops: page numbers: %w", pdf.Error())
	}
	return pdf, nil
}

// calculatePosition returns x, y coordinates for text placement.
func calculatePosition(pos Position, pageW, pageH, textW, textH, margin float64) (x, y float64) {
	_ = math.Abs // ensure math import is used
	switch pos {
	case TopLeft:
		return margin, margin + textH
	case TopCenter:
		return (pageW - textW) / 2, margin + textH
	case TopRight:
		return pageW - textW - margin, margin + textH
	case BottomLeft:
		return margin, pageH - margin
	case BottomRight:
		return pageW - textW - margin, pageH - margin
	case Center:
		return (pageW - textW) / 2, pageH / 2
	default: // BottomCenter
		return (pageW - textW) / 2, pageH - margin
	}
}
