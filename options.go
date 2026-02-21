package gofpdf

// Option is a functional option for configuring a new PDF document via NewDocument.
type Option func(*documentConfig)

type documentConfig struct {
	orientation string
	unit        string
	size        string
	fontDir     string
	pageSize    SizeType
}

// WithOrientation sets the default page orientation.
// Use OrientationPortrait ("portrait") or OrientationLandscape ("landscape").
func WithOrientation(orientation string) Option {
	return func(c *documentConfig) {
		c.orientation = orientation
	}
}

// WithUnit sets the measurement unit for page dimensions and drawing.
// Use UnitPoint ("pt"), UnitMillimeter ("mm"), UnitCentimeter ("cm"), or UnitInch ("inch").
func WithUnit(unit string) Option {
	return func(c *documentConfig) {
		c.unit = unit
	}
}

// WithPageSize sets the default page size by name.
// Use PageSizeA3, PageSizeA4, PageSizeA5, PageSizeLetter, PageSizeLegal, or "Tabloid".
func WithPageSize(size string) Option {
	return func(c *documentConfig) {
		c.size = size
	}
}

// WithPageSizeCustom sets a custom default page size in the configured unit.
func WithPageSizeCustom(width, height float64) Option {
	return func(c *documentConfig) {
		c.pageSize = SizeType{Wd: width, Ht: height}
	}
}

// WithFontDir sets the directory where font files are located.
func WithFontDir(dir string) Option {
	return func(c *documentConfig) {
		c.fontDir = dir
	}
}

// NewDocument creates a new PDF document using functional options.
// If no options are specified, defaults to portrait A4 with millimeter units.
//
// Example:
//
//	pdf := gofpdf.NewDocument(
//	    gofpdf.WithPageSize(gofpdf.PageSizeA4),
//	    gofpdf.WithOrientation(gofpdf.OrientationPortrait),
//	    gofpdf.WithUnit(gofpdf.UnitMillimeter),
//	)
func NewDocument(opts ...Option) *Fpdf {
	cfg := &documentConfig{
		orientation: "portrait",
		unit:        "mm",
		size:        "A4",
		fontDir:     "",
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return fpdfNew(cfg.orientation, cfg.unit, cfg.size, cfg.fontDir, cfg.pageSize)
}
