# GoFPDF - The Most Productive PDF Library for Go

[![MIT
licensed](https://img.shields.io/badge/license-MIT-blue.svg)](https://raw.githubusercontent.com/lvillar/gofpdf/master/LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/lvillar/gofpdf.svg)](https://pkg.go.dev/github.com/lvillar/gofpdf)
[![Go 1.23+](https://img.shields.io/badge/Go-1.23+-00ADD8.svg)](https://go.dev/)

A comprehensive PDF library for Go — generate, read, merge, fill forms, sign, and more.

## Features

### PDF Generation (core)
- UTF-8 TrueType fonts and right-to-left language support
- Choice of measurement unit, page format and margins
- Page header/footer management, automatic page breaks
- Text, drawing, images (JPEG, PNG, GIF, TIFF, SVG paths)
- Colors, gradients, alpha channel transparency
- Outline bookmarks, internal/external links
- Lines, Bezier curves, arcs, ellipses, rotation, scaling, clipping
- Document protection (password encryption)
- Layers, templates, barcodes, charting
- Import existing PDFs as templates

### PDF Reader (`reader/`)
- Parse and inspect existing PDF documents
- Extract text content from pages
- Access document metadata (title, author, etc.)
- Navigate page tree, resolve cross-references
- Decompress FlateDecode streams
- **Decrypt password-protected PDFs** (RC4 40-bit, RC4 128-bit)

### High-Level Tables (`table/`)
- Declarative table creation with functional options
- Automatic column width calculation and text wrapping
- Styled headers, alternating row colors, cell alignment
- Multi-page tables with repeated headers

### Page Operations (`pageops/`)
- **Merge** multiple PDFs into one
- **Split** PDFs by page ranges
- **Rotate** pages (90, 180, 270 degrees)
- **Add watermarks** (text overlays on every page)

### Interactive Forms (`form/`)
- **Create** forms with text fields, checkboxes, dropdowns, radio buttons
- **Fill** existing PDF forms programmatically
- **Flatten** forms (convert interactive fields to static content)

### Digital Signatures (`sign/`)
- **Sign** PDFs with PKCS#7 detached signatures
- **Verify** signatures and detect tampering
- Support for ECDSA and RSA keys
- Signature metadata: reason, location, timestamp

## Installation

```shell
go get github.com/lvillar/gofpdf
```

Requires **Go 1.23** or later.

## Quick Start

### Generate a PDF

```go
package main

import gofpdf "github.com/lvillar/gofpdf"

func main() {
    pdf := gofpdf.New("P", "mm", "A4", "")
    pdf.AddPage()
    pdf.SetFont("Helvetica", "B", 16)
    pdf.Cell(40, 10, "Hello, world")
    pdf.OutputFileAndClose("hello.pdf")
}
```

### Read a PDF

```go
doc, err := reader.Open("document.pdf")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Pages: %d\n", doc.NumPages())

for i, page := range doc.Pages() {
    text, _ := page.ExtractText()
    fmt.Printf("Page %d: %s\n", i, text)
}
```

### Create a Table

```go
pdf := gofpdf.New("P", "mm", "A4", "")
pdf.AddPage()
pdf.SetFont("Helvetica", "", 10)

tbl := table.New(pdf,
    table.WithColumns("Name", "Role", "Status"),
    table.WithHeaderStyle(table.CellStyle{FillColor: table.RGB(41, 128, 185), TextColor: table.RGB(255, 255, 255)}),
)
tbl.Row("Alice", "Engineer", "Active")
tbl.Row("Bob", "Designer", "On Leave")
tbl.Done()
```

### Merge PDFs

```go
err := pageops.MergeFiles([]string{"a.pdf", "b.pdf", "c.pdf"}, "merged.pdf")
```

### Fill a Form

```go
err := form.FillFile("template.pdf", "filled.pdf", map[string]string{
    "name":    "John Doe",
    "email":   "john@example.com",
    "country": "USA",
})
```

### Sign a Document

```go
err := sign.Sign(input, output, sign.Options{
    Certificate: cert,
    PrivateKey:  key,
    Reason:      "Approved",
    Location:    "New York",
})
```

### Open an Encrypted PDF

```go
doc, err := reader.OpenWithPassword("secret.pdf", "mypassword")
if err != nil {
    log.Fatal(err)
}
meta := doc.Metadata()
fmt.Println("Title:", meta["Title"])
```

## Package Overview

| Package | Description |
|---------|-------------|
| `gofpdf` (root) | PDF document generator — the core library |
| `reader/` | PDF parser — open, inspect, extract text |
| `table/` | High-level table builder with styling |
| `pageops/` | Merge, split, rotate, watermark operations |
| `form/` | Create, fill, and flatten interactive forms |
| `sign/` | Digital signatures — sign and verify |
| `contrib/` | Community contributions (barcodes, etc.) |
| `makefont/` | Font definition file generator |

## Migration from jung-kurt/gofpdf

This is a maintained fork of `jung-kurt/gofpdf` with full backward compatibility.
To migrate, update your import paths:

```go
// Before
import "github.com/jung-kurt/gofpdf"

// After
import "github.com/lvillar/gofpdf"
```

All existing code continues to work. The new packages (`reader`, `table`, `pageops`, `form`, `sign`) are additive — they don't change the core API.

## Error Handling

The core `Fpdf` type uses an error accumulation pattern — if an error occurs,
subsequent method calls are no-ops until you check `Output()` or `Error()`:

```go
pdf := gofpdf.New("P", "mm", "A4", "")
pdf.AddPage()
pdf.SetFont("Helvetica", "", 12)
pdf.Cell(40, 10, "Hello")
err := pdf.OutputFileAndClose("out.pdf") // check error here
```

The newer packages (`reader`, `table`, `pageops`, `form`, `sign`) return errors
directly following standard Go conventions.

## Contributing

Contributions are welcome. Please ensure your changes:

- Are compatible with the MIT License
- Are formatted with `go fmt`
- Pass `go vet` without warnings
- Include tests for new functionality
- Don't diminish test coverage

## License

MIT License. See [LICENSE](LICENSE) for details.

## Acknowledgments

This library is based on the excellent work of Kurt Jung and all
contributors to the original [jung-kurt/gofpdf](https://github.com/jung-kurt/gofpdf),
which itself is derived from the [FPDF](http://www.fpdf.org/) library by Olivier Plathey.
