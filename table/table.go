package table

import (
	"strings"

	gofpdf "github.com/lvillar/gofpdf"
)

// ColumnDef defines the properties of a table column.
type ColumnDef struct {
	Width    float64 // Fixed width. 0 means auto/fill.
	MinWidth float64 // Minimum width for auto columns.
	MaxWidth float64 // Maximum width for auto columns. 0 means unlimited.
	Align    string  // Default alignment for this column ("L", "C", "R").
}

// Table is a high-level table builder for generating PDF tables.
type Table struct {
	pdf        *gofpdf.Fpdf
	columns    []ColumnDef
	rows       []*Row
	headerRows int
	style      TableStyle
	x, y       float64 // starting position (0,0 means current)
	tableWidth float64 // total table width (0 means page width minus margins)
}

// New creates a new Table associated with the given PDF document.
func New(pdf *gofpdf.Fpdf) *Table {
	return &Table{
		pdf: pdf,
		style: TableStyle{
			CellPadding: UniformPadding(1),
		},
	}
}

// SetColumns sets column definitions for the table.
func (t *Table) SetColumns(cols ...ColumnDef) *Table {
	t.columns = cols
	return t
}

// SetColumnWidths is a convenience method to set column widths directly.
// A width of 0 means the column will auto-fill remaining space.
func (t *Table) SetColumnWidths(widths ...float64) *Table {
	t.columns = make([]ColumnDef, len(widths))
	for i, w := range widths {
		t.columns[i] = ColumnDef{Width: w}
	}
	return t
}

// SetHeaderRows marks the first n rows as header rows.
// Header rows are repeated at the top of each new page.
func (t *Table) SetHeaderRows(n int) *Table {
	t.headerRows = n
	return t
}

// SetStyle sets the table-wide style.
func (t *Table) SetStyle(s TableStyle) *Table {
	t.style = s
	return t
}

// SetPosition sets the starting position for the table.
// If not called, the table starts at the current PDF cursor position.
func (t *Table) SetPosition(x, y float64) *Table {
	t.x = x
	t.y = y
	return t
}

// SetWidth sets the total table width. If not called, uses page width minus margins.
func (t *Table) SetWidth(w float64) *Table {
	t.tableWidth = w
	return t
}

// AddRow adds a new data row to the table and returns it for chaining.
func (t *Table) AddRow() *Row {
	r := &Row{}
	t.rows = append(t.rows, r)
	return r
}

// AddHeaderRow adds a new header row and returns it for chaining.
func (t *Table) AddHeaderRow() *Row {
	r := &Row{isHeader: true}
	// Insert header row before data rows
	insertIdx := 0
	for i, existing := range t.rows {
		if !existing.isHeader {
			insertIdx = i
			break
		}
		insertIdx = i + 1
	}
	// Insert at the correct position
	t.rows = append(t.rows, nil)
	copy(t.rows[insertIdx+1:], t.rows[insertIdx:])
	t.rows[insertIdx] = r
	t.headerRows++
	return r
}

// Render draws the table to the PDF document.
func (t *Table) Render() error {
	if t.pdf.Err() {
		return t.pdf.Error()
	}

	widths := t.calculateWidths()

	// Save starting position
	startX := t.x
	if startX == 0 {
		startX = t.pdf.GetX()
	}
	if t.y != 0 {
		t.pdf.SetY(t.y)
	}

	// Separate header and body rows
	var headerRows, bodyRows []*Row
	for _, r := range t.rows {
		if r.isHeader {
			headerRows = append(headerRows, r)
		} else {
			bodyRows = append(bodyRows, r)
		}
	}

	// Render header rows first
	for _, r := range headerRows {
		t.renderRow(r, widths, startX, -1, true)
	}

	// Render body rows
	for i, r := range bodyRows {
		// Check if we need a page break
		rowH := t.calculateRowHeight(r, widths)
		_, pageH := t.pdf.GetPageSize()
		_, _, _, bMargin := t.pdf.GetMargins()

		if t.pdf.GetY()+rowH > pageH-bMargin {
			t.pdf.AddPage()
			// Re-render headers on new page
			for _, hr := range headerRows {
				t.renderRow(hr, widths, startX, -1, true)
			}
		}

		t.renderRow(r, widths, startX, i, false)
	}

	return t.pdf.Error()
}

// calculateWidths computes final column widths based on definitions and available space.
func (t *Table) calculateWidths() []float64 {
	totalWidth := t.tableWidth
	if totalWidth == 0 {
		pageW, _ := t.pdf.GetPageSize()
		lMargin, _, rMargin, _ := t.pdf.GetMargins()
		totalWidth = pageW - lMargin - rMargin
	}

	numCols := len(t.columns)
	if numCols == 0 {
		// Auto-detect from first row
		if len(t.rows) > 0 {
			numCols = len(t.rows[0].cells)
		}
		if numCols == 0 {
			return nil
		}
		t.columns = make([]ColumnDef, numCols)
	}

	widths := make([]float64, numCols)
	fixedTotal := 0.0
	autoCount := 0

	for i, col := range t.columns {
		if col.Width > 0 {
			widths[i] = col.Width
			fixedTotal += col.Width
		} else {
			autoCount++
		}
	}

	// Distribute remaining space to auto columns
	if autoCount > 0 {
		remaining := totalWidth - fixedTotal
		if remaining < 0 {
			remaining = 0
		}
		autoWidth := remaining / float64(autoCount)
		for i, col := range t.columns {
			if col.Width == 0 {
				w := autoWidth
				if col.MinWidth > 0 && w < col.MinWidth {
					w = col.MinWidth
				}
				if col.MaxWidth > 0 && w > col.MaxWidth {
					w = col.MaxWidth
				}
				widths[i] = w
			}
		}
	}

	return widths
}

// calculateRowHeight computes the height needed for a row based on cell content.
func (t *Table) calculateRowHeight(r *Row, widths []float64) float64 {
	maxH := 5.0 // minimum row height
	if r.minH > maxH {
		maxH = r.minH
	}

	padding := t.style.CellPadding

	for i, cell := range r.cells {
		if i >= len(widths) {
			break
		}

		// Calculate cell width (including colspan)
		cellW := widths[i]
		for j := 1; j < cell.colspan && i+j < len(widths); j++ {
			cellW += widths[i+j]
		}

		contentW := cellW - padding.Left - padding.Right
		if contentW < 1 {
			contentW = 1
		}

		switch c := cell.content.(type) {
		case TextContent:
			// Calculate number of lines needed
			lines := t.pdf.SplitLines([]byte(c.Text), contentW)
			_, fontSize := t.pdf.GetFontSize()
			lineH := fontSize * 1.5
			cellH := float64(len(lines))*lineH + padding.Top + padding.Bottom
			if cellH > maxH {
				maxH = cellH
			}
		case ImageContent:
			// Use a default image height
			cellH := 10.0 + padding.Top + padding.Bottom
			if cellH > maxH {
				maxH = cellH
			}
		}
	}

	return maxH
}

// renderRow renders a single row to the PDF.
func (t *Table) renderRow(r *Row, widths []float64, startX float64, bodyIdx int, isHeader bool) {
	rowH := t.calculateRowHeight(r, widths)
	padding := t.style.CellPadding

	t.pdf.SetX(startX)
	y := t.pdf.GetY()

	for i, cell := range r.cells {
		if i >= len(widths) {
			break
		}

		// Calculate cell width (including colspan)
		cellW := widths[i]
		for j := 1; j < cell.colspan && i+j < len(widths); j++ {
			cellW += widths[i+j]
		}

		// Determine cell style
		style := t.resolveCellStyle(cell, r, bodyIdx, isHeader)

		// Save state
		x := t.pdf.GetX()

		// Draw background
		if style.FillColor != nil {
			t.pdf.SetFillColor(style.FillColor.R, style.FillColor.G, style.FillColor.B)
			t.pdf.Rect(x, y, cellW, rowH, "F")
		}

		// Draw border
		if t.style.Border != nil {
			if t.style.Border.Color != (RGBColor{}) {
				bc := t.style.Border.Color
				t.pdf.SetDrawColor(bc.R, bc.G, bc.B)
			}
			if t.style.Border.Width > 0 {
				t.pdf.SetLineWidth(t.style.Border.Width)
			}
		}
		t.pdf.Rect(x, y, cellW, rowH, "D")

		// Set text properties
		if style.TextColor != nil {
			t.pdf.SetTextColor(style.TextColor.R, style.TextColor.G, style.TextColor.B)
		}
		if style.Font != nil {
			t.pdf.SetFont(style.Font.Family, style.Font.Style, style.Font.Size)
		}

		// Render content
		align := "L"
		if style.Align != "" {
			align = style.Align
		} else if i < len(t.columns) && t.columns[i].Align != "" {
			align = t.columns[i].Align
		}

		contentX := x + padding.Left
		contentY := y + padding.Top
		contentW := cellW - padding.Left - padding.Right

		switch c := cell.content.(type) {
		case TextContent:
			t.pdf.SetXY(contentX, contentY)
			// Use MultiCell for wrapped text, but we need to handle alignment
			if strings.Contains(c.Text, "\n") || t.pdf.GetStringWidth(c.Text) > contentW {
				t.pdf.MultiCell(contentW, rowH-padding.Top-padding.Bottom, c.Text, "", align, false)
			} else {
				t.pdf.CellFormat(contentW, rowH-padding.Top-padding.Bottom,
					c.Text, "", 0, align, false, 0, "")
			}
		case ImageContent:
			imgH := rowH - padding.Top - padding.Bottom
			t.pdf.Image(c.Path, contentX, contentY, 0, imgH, false, c.Type, 0, "")
		}

		// Move to next cell position
		t.pdf.SetXY(x+cellW, y)
	}

	// Restore colors to defaults
	t.pdf.SetDrawColor(0, 0, 0)
	t.pdf.SetFillColor(0, 0, 0)
	t.pdf.SetTextColor(0, 0, 0)

	// Move to next row
	t.pdf.SetXY(startX, y+rowH)
}

// resolveCellStyle determines the effective style for a cell by merging
// table, alternate row, header, row, and cell-level styles.
func (t *Table) resolveCellStyle(cell *Cell, row *Row, bodyIdx int, isHeader bool) CellStyle {
	var result CellStyle

	// Table-level font
	if t.style.CellFont != nil {
		result.Font = t.style.CellFont
	}

	// Header style
	if isHeader && t.style.HeaderStyle != nil {
		mergeStyle(&result, t.style.HeaderStyle)
	}

	// Alternate row colors (only for body rows)
	if !isHeader && t.style.AlternateRows != nil && bodyIdx >= 0 {
		if bodyIdx%2 == 0 {
			mergeStyle(&result, &t.style.AlternateRows.Even)
		} else {
			mergeStyle(&result, &t.style.AlternateRows.Odd)
		}
	}

	// Row-level style
	if row.style != nil {
		mergeStyle(&result, row.style)
	}

	// Cell-level style (highest priority)
	if cell.style != nil {
		mergeStyle(&result, cell.style)
	}

	return result
}

// mergeStyle copies non-nil fields from src to dst.
func mergeStyle(dst, src *CellStyle) {
	if src.FillColor != nil {
		dst.FillColor = src.FillColor
	}
	if src.TextColor != nil {
		dst.TextColor = src.TextColor
	}
	if src.BorderColor != nil {
		dst.BorderColor = src.BorderColor
	}
	if src.Font != nil {
		dst.Font = src.Font
	}
	if src.Align != "" {
		dst.Align = src.Align
	}
	if src.Padding != nil {
		dst.Padding = src.Padding
	}
}
