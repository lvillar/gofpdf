// Package doctpl provides a JSON-based document template DSL for generating PDFs.
//
// It allows defining PDF documents using a declarative JSON schema that is easy
// for both humans and LLMs to generate. The schema supports text, headings,
// paragraphs, tables, images, lines, rectangles, and spacers.
//
// Example JSON:
//
//	{
//	  "title": "My Document",
//	  "pageSize": "A4",
//	  "pages": [{
//	    "elements": [
//	      {"type": "heading", "text": "Hello World", "level": 1},
//	      {"type": "paragraph", "text": "Some body text here."}
//	    ]
//	  }]
//	}
package doctpl

// Document is the top-level template that describes an entire PDF.
type Document struct {
	Title    string   `json:"title,omitempty"`
	Author   string   `json:"author,omitempty"`
	Subject  string   `json:"subject,omitempty"`
	PageSize string   `json:"pageSize,omitempty"` // A4, Letter, Legal (default: A4)
	Unit     string   `json:"unit,omitempty"`     // mm, cm, in, pt (default: mm)
	Margin   *Margin  `json:"margin,omitempty"`
	Font     *Font    `json:"font,omitempty"` // default font for the document
	Pages    []Page   `json:"pages"`
	Header   *Header  `json:"header,omitempty"` // repeated on every page
	Footer   *Footer  `json:"footer,omitempty"` // repeated on every page
}

// Margin defines page margins.
type Margin struct {
	Top    float64 `json:"top"`
	Right  float64 `json:"right"`
	Bottom float64 `json:"bottom"`
	Left   float64 `json:"left"`
}

// Font specifies a font face.
type Font struct {
	Family string  `json:"family"` // Helvetica, Courier, Times
	Style  string  `json:"style"`  // "" (regular), "B" (bold), "I" (italic), "BI"
	Size   float64 `json:"size"`
}

// Color is an RGB color.
type Color struct {
	R int `json:"r"`
	G int `json:"g"`
	B int `json:"b"`
}

// Page represents a single page of the document.
type Page struct {
	Size     string    `json:"size,omitempty"` // override document page size
	Elements []Element `json:"elements"`
}

// Element is a single visual element within a page.
// The Type field determines which other fields are relevant.
type Element struct {
	Type string `json:"type"` // heading, paragraph, table, image, line, rect, spacer, list, hr

	// Text content (heading, paragraph)
	Text  string `json:"text,omitempty"`
	Level int    `json:"level,omitempty"` // heading level 1-6
	Align string `json:"align,omitempty"` // L, C, R (default: L)

	// Font override for this element
	Font  *Font  `json:"font,omitempty"`
	Color *Color `json:"color,omitempty"`

	// Table
	Columns     []TableColumn `json:"columns,omitempty"`
	Rows        [][]string    `json:"rows,omitempty"`
	HeaderStyle *CellStyle    `json:"headerStyle,omitempty"`
	CellStyle   *CellStyle    `json:"cellStyle,omitempty"`

	// Image
	Src    string  `json:"src,omitempty"`
	X      float64 `json:"x,omitempty"`
	Y      float64 `json:"y,omitempty"`
	Width  float64 `json:"width,omitempty"`
	Height float64 `json:"height,omitempty"`

	// Line
	X1 float64 `json:"x1,omitempty"`
	Y1 float64 `json:"y1,omitempty"`
	X2 float64 `json:"x2,omitempty"`
	Y2 float64 `json:"y2,omitempty"`

	// Spacer / HR
	SpacerHeight float64 `json:"spacerHeight,omitempty"`
	LineWidth    float64 `json:"lineWidth,omitempty"`

	// List
	Items     []string `json:"items,omitempty"`
	Ordered   bool     `json:"ordered,omitempty"`
	BulletStr string   `json:"bullet,omitempty"` // custom bullet character

	// Background (rect)
	FillColor *Color  `json:"fillColor,omitempty"`
	Border    bool    `json:"border,omitempty"`
}

// TableColumn defines a column in a table element.
type TableColumn struct {
	Header string  `json:"header"`
	Width  float64 `json:"width,omitempty"` // 0 = auto
	Align  string  `json:"align,omitempty"` // L, C, R
}

// CellStyle defines styling for table cells.
type CellStyle struct {
	FillColor *Color `json:"fillColor,omitempty"`
	TextColor *Color `json:"textColor,omitempty"`
	Font      *Font  `json:"font,omitempty"`
}

// Header defines content repeated at the top of every page.
type Header struct {
	Text  string `json:"text,omitempty"`
	Align string `json:"align,omitempty"`
	Font  *Font  `json:"font,omitempty"`
	Color *Color `json:"color,omitempty"`
}

// Footer defines content repeated at the bottom of every page.
type Footer struct {
	Text   string `json:"text,omitempty"` // supports {page} and {pages} placeholders
	Align  string `json:"align,omitempty"`
	Font   *Font  `json:"font,omitempty"`
	Color  *Color `json:"color,omitempty"`
}
