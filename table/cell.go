package table

import (
	"fmt"
)

// CellContent represents the content of a table cell.
type CellContent interface {
	cellContent()
}

// TextContent is a simple text cell content.
type TextContent struct {
	Text string
}

func (TextContent) cellContent() {}

// ImageContent is an image cell content.
type ImageContent struct {
	Path string
	Type string // "jpg", "png", etc. Empty for auto-detect.
}

func (ImageContent) cellContent() {}

// Cell represents a single cell in a table row.
type Cell struct {
	content CellContent
	colspan int
	rowspan int
	style   *CellStyle
}

// SetColspan sets the number of columns this cell spans.
func (c *Cell) SetColspan(n int) *Cell {
	if n > 0 {
		c.colspan = n
	}
	return c
}

// SetRowspan sets the number of rows this cell spans.
func (c *Cell) SetRowspan(n int) *Cell {
	if n > 0 {
		c.rowspan = n
	}
	return c
}

// SetStyle sets the style for this cell, overriding table/row defaults.
func (c *Cell) SetStyle(s CellStyle) *Cell {
	c.style = &s
	return c
}

// SetAlign sets the horizontal alignment for this cell.
func (c *Cell) SetAlign(align string) *Cell {
	if c.style == nil {
		c.style = &CellStyle{}
	}
	c.style.Align = align
	return c
}

// SetFillColor sets the background color for this cell.
func (c *Cell) SetFillColor(r, g, b int) *Cell {
	if c.style == nil {
		c.style = &CellStyle{}
	}
	c.style.FillColor = &RGBColor{r, g, b}
	return c
}

// Row represents a single row in a table.
type Row struct {
	cells    []*Cell
	style    *CellStyle
	isHeader bool
	minH     float64 // minimum row height
}

// AddCell adds a text cell to the row and returns the cell for chaining.
func (r *Row) AddCell(text string) *Cell {
	c := &Cell{
		content: TextContent{Text: text},
		colspan: 1,
		rowspan: 1,
	}
	r.cells = append(r.cells, c)
	return c
}

// AddCellf adds a formatted text cell to the row.
func (r *Row) AddCellf(format string, args ...any) *Cell {
	return r.AddCell(fmt.Sprintf(format, args...))
}

// AddImageCell adds an image cell to the row.
func (r *Row) AddImageCell(imagePath string) *Cell {
	c := &Cell{
		content: ImageContent{Path: imagePath},
		colspan: 1,
		rowspan: 1,
	}
	r.cells = append(r.cells, c)
	return c
}

// SetStyle sets the style for all cells in this row.
func (r *Row) SetStyle(s CellStyle) *Row {
	r.style = &s
	return r
}

// SetMinHeight sets the minimum height for this row.
func (r *Row) SetMinHeight(h float64) *Row {
	r.minH = h
	return r
}
