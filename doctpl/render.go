package doctpl

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	gofpdf "github.com/lvillar/gofpdf"
	"github.com/lvillar/gofpdf/table"
)

// Render parses a JSON template and writes the resulting PDF to w.
func Render(w io.Writer, jsonTemplate []byte) error {
	var doc Document
	if err := json.Unmarshal(jsonTemplate, &doc); err != nil {
		return fmt.Errorf("doctpl: parsing template: %w", err)
	}
	return RenderDocument(w, &doc)
}

// RenderDocument renders a Document struct to a PDF written to w.
func RenderDocument(w io.Writer, doc *Document) error {
	pageSize := doc.PageSize
	if pageSize == "" {
		pageSize = "A4"
	}
	unit := doc.Unit
	if unit == "" {
		unit = "mm"
	}

	pdf := gofpdf.New("P", unit, pageSize, "")

	// Apply margins
	if doc.Margin != nil {
		pdf.SetMargins(doc.Margin.Left, doc.Margin.Top, doc.Margin.Right)
		pdf.SetAutoPageBreak(true, doc.Margin.Bottom)
	} else {
		pdf.SetAutoPageBreak(true, 15)
	}

	// Set metadata
	if doc.Title != "" {
		pdf.SetTitle(doc.Title, true)
	}
	if doc.Author != "" {
		pdf.SetAuthor(doc.Author, true)
	}
	if doc.Subject != "" {
		pdf.SetSubject(doc.Subject, true)
	}

	// Default font
	defaultFont := Font{Family: "Helvetica", Style: "", Size: 11}
	if doc.Font != nil {
		if doc.Font.Family != "" {
			defaultFont.Family = doc.Font.Family
		}
		if doc.Font.Size > 0 {
			defaultFont.Size = doc.Font.Size
		}
		defaultFont.Style = doc.Font.Style
	}

	// Set up header/footer callbacks
	if doc.Header != nil {
		hdr := *doc.Header
		pdf.SetHeaderFunc(func() {
			renderHeader(pdf, hdr, defaultFont)
		})
	}
	if doc.Footer != nil {
		ftr := *doc.Footer
		pdf.SetFooterFunc(func() {
			renderFooter(pdf, ftr, defaultFont)
		})
	}

	// Render pages
	for pageIdx, page := range doc.Pages {
		if page.Size != "" && page.Size != pageSize {
			pdf.AddPageFormat("P", pdf.GetPageSizeStr(page.Size))
		} else {
			pdf.AddPage()
		}

		pdf.SetFont(defaultFont.Family, defaultFont.Style, defaultFont.Size)

		for _, elem := range page.Elements {
			if err := renderElement(pdf, elem, defaultFont); err != nil {
				return fmt.Errorf("doctpl: page %d: %w", pageIdx+1, err)
			}
		}
	}

	// If no pages were defined, add one empty page
	if len(doc.Pages) == 0 {
		pdf.AddPage()
	}

	if pdf.Err() {
		return fmt.Errorf("doctpl: %w", pdf.Error())
	}

	return pdf.Output(w)
}

func renderElement(pdf *gofpdf.Fpdf, elem Element, defaultFont Font) error {
	switch elem.Type {
	case "heading":
		return renderHeading(pdf, elem, defaultFont)
	case "paragraph", "text":
		return renderParagraph(pdf, elem, defaultFont)
	case "table":
		return renderTable(pdf, elem, defaultFont)
	case "image":
		return renderImage(pdf, elem)
	case "line":
		renderLine(pdf, elem)
	case "rect":
		renderRect(pdf, elem)
	case "spacer":
		renderSpacer(pdf, elem)
	case "hr":
		renderHR(pdf, elem)
	case "list":
		renderList(pdf, elem, defaultFont)
	default:
		return fmt.Errorf("unknown element type %q", elem.Type)
	}
	return nil
}

func renderHeading(pdf *gofpdf.Fpdf, elem Element, defaultFont Font) error {
	level := elem.Level
	if level < 1 {
		level = 1
	}
	if level > 6 {
		level = 6
	}

	// Heading sizes: h1=24, h2=20, h3=16, h4=14, h5=12, h6=11
	sizes := []float64{24, 20, 16, 14, 12, 11}
	size := sizes[level-1]

	family := defaultFont.Family
	style := "B"
	if elem.Font != nil {
		if elem.Font.Family != "" {
			family = elem.Font.Family
		}
		if elem.Font.Style != "" {
			style = elem.Font.Style
		}
		if elem.Font.Size > 0 {
			size = elem.Font.Size
		}
	}

	if elem.Color != nil {
		pdf.SetTextColor(elem.Color.R, elem.Color.G, elem.Color.B)
	}

	pdf.SetFont(family, style, size)

	// Add spacing before heading
	if level <= 2 {
		pdf.Ln(size * 0.4)
	} else {
		pdf.Ln(size * 0.3)
	}

	align := "L"
	if elem.Align != "" {
		align = strings.ToUpper(elem.Align)
	}

	pageW, _ := pdf.GetPageSize()
	lm, _, rm, _ := pdf.GetMargins()
	contentW := pageW - lm - rm

	pdf.MultiCell(contentW, size*0.5, elem.Text, "", align, false)
	pdf.Ln(size * 0.2)

	// Reset font and color
	pdf.SetFont(defaultFont.Family, defaultFont.Style, defaultFont.Size)
	if elem.Color != nil {
		pdf.SetTextColor(0, 0, 0)
	}

	return nil
}

func renderParagraph(pdf *gofpdf.Fpdf, elem Element, defaultFont Font) error {
	family := defaultFont.Family
	style := defaultFont.Style
	size := defaultFont.Size

	if elem.Font != nil {
		if elem.Font.Family != "" {
			family = elem.Font.Family
		}
		if elem.Font.Style != "" {
			style = elem.Font.Style
		}
		if elem.Font.Size > 0 {
			size = elem.Font.Size
		}
	}

	if elem.Color != nil {
		pdf.SetTextColor(elem.Color.R, elem.Color.G, elem.Color.B)
	}

	pdf.SetFont(family, style, size)

	align := "L"
	if elem.Align != "" {
		align = strings.ToUpper(elem.Align)
	}

	pageW, _ := pdf.GetPageSize()
	lm, _, rm, _ := pdf.GetMargins()
	contentW := pageW - lm - rm

	pdf.MultiCell(contentW, size*0.5, elem.Text, "", align, false)
	pdf.Ln(size * 0.3)

	// Reset
	pdf.SetFont(defaultFont.Family, defaultFont.Style, defaultFont.Size)
	if elem.Color != nil {
		pdf.SetTextColor(0, 0, 0)
	}

	return nil
}

func renderTable(pdf *gofpdf.Fpdf, elem Element, defaultFont Font) error {
	t := table.New(pdf)

	// Set up columns
	if len(elem.Columns) > 0 {
		cols := make([]table.ColumnDef, len(elem.Columns))
		for i, c := range elem.Columns {
			cols[i] = table.ColumnDef{
				Width: c.Width,
				Align: c.Align,
			}
		}
		t.SetColumns(cols...)

		// Add header row
		hr := t.AddHeaderRow()
		for _, c := range elem.Columns {
			hr.AddCell(c.Header)
		}

		// Style the header
		headerStyle := table.CellStyle{
			FillColor: &table.RGBColor{R: 63, G: 81, B: 181},
			TextColor: &table.RGBColor{R: 255, G: 255, B: 255},
			Font:      &table.FontSpec{Family: defaultFont.Family, Style: "B", Size: defaultFont.Size},
		}
		if elem.HeaderStyle != nil {
			if elem.HeaderStyle.FillColor != nil {
				headerStyle.FillColor = &table.RGBColor{
					R: elem.HeaderStyle.FillColor.R,
					G: elem.HeaderStyle.FillColor.G,
					B: elem.HeaderStyle.FillColor.B,
				}
			}
			if elem.HeaderStyle.TextColor != nil {
				headerStyle.TextColor = &table.RGBColor{
					R: elem.HeaderStyle.TextColor.R,
					G: elem.HeaderStyle.TextColor.G,
					B: elem.HeaderStyle.TextColor.B,
				}
			}
			if elem.HeaderStyle.Font != nil {
				if elem.HeaderStyle.Font.Family != "" {
					headerStyle.Font.Family = elem.HeaderStyle.Font.Family
				}
				if elem.HeaderStyle.Font.Style != "" {
					headerStyle.Font.Style = elem.HeaderStyle.Font.Style
				}
				if elem.HeaderStyle.Font.Size > 0 {
					headerStyle.Font.Size = elem.HeaderStyle.Font.Size
				}
			}
		}

		t.SetStyle(table.TableStyle{
			CellPadding: table.UniformPadding(2),
			HeaderStyle: &headerStyle,
			AlternateRows: &table.AlternateStyle{
				Even: table.CellStyle{
					FillColor: &table.RGBColor{R: 245, G: 245, B: 245},
				},
			},
		})
	}

	// Add data rows
	for _, row := range elem.Rows {
		r := t.AddRow()
		for _, cell := range row {
			r.AddCell(cell)
		}
	}

	pdf.Ln(2)
	return t.Render()
}

func renderImage(pdf *gofpdf.Fpdf, elem Element) error {
	if elem.Src == "" {
		return fmt.Errorf("image element requires 'src' field")
	}

	x := elem.X
	y := elem.Y
	w := elem.Width
	h := elem.Height

	// If no position specified, use flow position
	if x == 0 && y == 0 {
		x = pdf.GetX()
		y = pdf.GetY()
	}

	pdf.Image(elem.Src, x, y, w, h, false, "", 0, "")

	// Advance Y if using flow
	if elem.Y == 0 && h > 0 {
		pdf.SetY(y + h + 2)
	}

	return nil
}

func renderLine(pdf *gofpdf.Fpdf, elem Element) {
	if elem.LineWidth > 0 {
		pdf.SetLineWidth(elem.LineWidth)
	}
	if elem.Color != nil {
		pdf.SetDrawColor(elem.Color.R, elem.Color.G, elem.Color.B)
	}
	pdf.Line(elem.X1, elem.Y1, elem.X2, elem.Y2)
	if elem.Color != nil {
		pdf.SetDrawColor(0, 0, 0)
	}
	if elem.LineWidth > 0 {
		pdf.SetLineWidth(0.2)
	}
}

func renderRect(pdf *gofpdf.Fpdf, elem Element) {
	style := ""
	if elem.FillColor != nil {
		pdf.SetFillColor(elem.FillColor.R, elem.FillColor.G, elem.FillColor.B)
		style = "F"
	}
	if elem.Border {
		if style == "F" {
			style = "FD"
		} else {
			style = "D"
		}
	}
	if style == "" {
		style = "D"
	}
	pdf.Rect(elem.X, elem.Y, elem.Width, elem.Height, style)
	if elem.FillColor != nil {
		pdf.SetFillColor(0, 0, 0)
	}
}

func renderSpacer(pdf *gofpdf.Fpdf, elem Element) {
	h := elem.SpacerHeight
	if h == 0 {
		h = 10
	}
	pdf.Ln(h)
}

func renderHR(pdf *gofpdf.Fpdf, elem Element) {
	pageW, _ := pdf.GetPageSize()
	lm, _, rm, _ := pdf.GetMargins()

	pdf.Ln(3)
	y := pdf.GetY()

	lw := elem.LineWidth
	if lw == 0 {
		lw = 0.3
	}
	pdf.SetLineWidth(lw)

	if elem.Color != nil {
		pdf.SetDrawColor(elem.Color.R, elem.Color.G, elem.Color.B)
	} else {
		pdf.SetDrawColor(180, 180, 180)
	}

	pdf.Line(lm, y, pageW-rm, y)
	pdf.SetDrawColor(0, 0, 0)
	pdf.SetLineWidth(0.2)
	pdf.Ln(3)
}

func renderList(pdf *gofpdf.Fpdf, elem Element, defaultFont Font) {
	family := defaultFont.Family
	style := defaultFont.Style
	size := defaultFont.Size

	if elem.Font != nil {
		if elem.Font.Family != "" {
			family = elem.Font.Family
		}
		if elem.Font.Style != "" {
			style = elem.Font.Style
		}
		if elem.Font.Size > 0 {
			size = elem.Font.Size
		}
	}

	pdf.SetFont(family, style, size)

	pageW, _ := pdf.GetPageSize()
	lm, _, rm, _ := pdf.GetMargins()
	contentW := pageW - lm - rm - 10 // indent for bullet

	bullet := "\u2022 " // default bullet
	if elem.BulletStr != "" {
		bullet = elem.BulletStr + " "
	}

	for i, item := range elem.Items {
		prefix := bullet
		if elem.Ordered {
			prefix = fmt.Sprintf("%d. ", i+1)
		}

		pdf.SetX(lm + 5)
		pdf.MultiCell(contentW, size*0.5, prefix+item, "", "L", false)
		pdf.Ln(1)
	}

	pdf.Ln(2)
	pdf.SetFont(defaultFont.Family, defaultFont.Style, defaultFont.Size)
}

func renderHeader(pdf *gofpdf.Fpdf, hdr Header, defaultFont Font) {
	family := defaultFont.Family
	style := "B"
	size := 9.0

	if hdr.Font != nil {
		if hdr.Font.Family != "" {
			family = hdr.Font.Family
		}
		if hdr.Font.Style != "" {
			style = hdr.Font.Style
		}
		if hdr.Font.Size > 0 {
			size = hdr.Font.Size
		}
	}

	if hdr.Color != nil {
		pdf.SetTextColor(hdr.Color.R, hdr.Color.G, hdr.Color.B)
	}

	pdf.SetFont(family, style, size)

	pageW, _ := pdf.GetPageSize()
	lm, _, rm, _ := pdf.GetMargins()
	contentW := pageW - lm - rm

	align := "L"
	if hdr.Align != "" {
		align = strings.ToUpper(hdr.Align)
	}

	pdf.SetY(5)
	pdf.CellFormat(contentW, 10, hdr.Text, "", 0, align, false, 0, "")
	pdf.Ln(5)

	if hdr.Color != nil {
		pdf.SetTextColor(0, 0, 0)
	}
}

func renderFooter(pdf *gofpdf.Fpdf, ftr Footer, defaultFont Font) {
	family := defaultFont.Family
	style := ""
	size := 8.0

	if ftr.Font != nil {
		if ftr.Font.Family != "" {
			family = ftr.Font.Family
		}
		if ftr.Font.Style != "" {
			style = ftr.Font.Style
		}
		if ftr.Font.Size > 0 {
			size = ftr.Font.Size
		}
	}

	if ftr.Color != nil {
		pdf.SetTextColor(ftr.Color.R, ftr.Color.G, ftr.Color.B)
	} else {
		pdf.SetTextColor(128, 128, 128)
	}

	pdf.SetFont(family, style, size)

	pageW, _ := pdf.GetPageSize()
	lm, _, rm, _ := pdf.GetMargins()
	contentW := pageW - lm - rm

	align := "C"
	if ftr.Align != "" {
		align = strings.ToUpper(ftr.Align)
	}

	// Replace placeholders
	text := ftr.Text
	text = strings.ReplaceAll(text, "{page}", fmt.Sprintf("%d", pdf.PageNo()))
	// {pages} requires AliasNbPages which is set at generation time
	// We use a simple format here
	text = strings.ReplaceAll(text, "{pages}", "{nb}")

	pdf.SetY(-15)
	pdf.CellFormat(contentW, 10, text, "", 0, align, false, 0, "")

	pdf.SetTextColor(0, 0, 0)
}
