// Package table provides a high-level API for creating tables in PDF documents.
//
// It integrates with the Fpdf type from the parent package and provides
// features like auto-width columns, repeating headers, alternating row colors,
// colspan/rowspan, and automatic page breaks.
package table

// RGBColor represents an RGB color value.
type RGBColor struct {
	R, G, B int
}

// FontSpec defines font properties for text rendering.
type FontSpec struct {
	Family string
	Style  string  // "", "B", "I", "BI"
	Size   float64 // in points
}

// Padding defines spacing inside a cell.
type Padding struct {
	Top, Right, Bottom, Left float64
}

// UniformPadding creates a Padding with the same value on all sides.
func UniformPadding(v float64) Padding {
	return Padding{Top: v, Right: v, Bottom: v, Left: v}
}

// BorderStyle defines the appearance of cell borders.
type BorderStyle struct {
	Width float64
	Color RGBColor
}

// CellStyle defines the visual appearance of a cell.
type CellStyle struct {
	FillColor   *RGBColor
	TextColor   *RGBColor
	BorderColor *RGBColor
	Font        *FontSpec
	Align       string // "L", "C", "R" (horizontal), "T", "M", "B" (vertical)
	Padding     *Padding
}

// AlternateStyle defines alternating row colors.
type AlternateStyle struct {
	Even CellStyle
	Odd  CellStyle
}

// TableStyle defines the overall appearance of a table.
type TableStyle struct {
	Border        *BorderStyle
	AlternateRows *AlternateStyle
	HeaderStyle   *CellStyle
	CellPadding   Padding
	CellFont      *FontSpec
}
