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

### JSON Template DSL (`doctpl/`)
- Create PDFs from declarative **JSON templates** — ideal for LLM-generated content
- Headings (h1–h6), paragraphs, tables, lists, images, horizontal rules, spacers
- Custom fonts, colors, margins, headers, and footers
- JSON round-trip: templates can be serialized, stored, and re-rendered

### MCP Server (`mcp/`, `cmd/gofpdf-mcp/`)
- **Model Context Protocol** server for AI assistants (Claude Desktop, etc.)
- 10 tools: `create_pdf`, `read_pdf`, `read_pdf_text`, `merge_pdfs`, `add_watermark`, `add_page_numbers`, `fill_form`, `flatten_form`, `rotate_pages`, `pdf_info`
- 4 resources: `pdf://text`, `pdf://metadata`, `pdf://pages`, `pdf://form-fields`
- JSON-RPC 2.0 over stdio — zero external dependencies

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

### Create a PDF from a JSON Template

```go
template := `{
    "title": "Invoice #1234",
    "pageSize": "A4",
    "pages": [{
        "elements": [
            {"type": "heading", "text": "Invoice #1234", "level": 1},
            {"type": "paragraph", "text": "Date: 2024-01-15\nBill To: John Doe"},
            {"type": "table",
             "columns": [
                {"header": "Item", "width": 80},
                {"header": "Qty", "width": 20, "align": "C"},
                {"header": "Price", "width": 30, "align": "R"}
             ],
             "rows": [
                ["Widget A", "10", "$5.00"],
                ["Widget B", "5", "$12.00"]
             ]},
            {"type": "paragraph", "text": "Total: $110.00", "align": "R",
             "font": {"style": "B", "size": 14}}
        ]
    }]
}`

var buf bytes.Buffer
doctpl.Render(&buf, []byte(template))
os.WriteFile("invoice.pdf", buf.Bytes(), 0644)
```

### Use with AI Assistants (MCP)

Install the MCP server:

```shell
go install github.com/lvillar/gofpdf/cmd/gofpdf-mcp@latest
```

Add to your Claude Desktop configuration (`~/.config/claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "gofpdf": {
      "command": "gofpdf-mcp"
    }
  }
}
```

Once configured, you can ask Claude to:
- *"Create a PDF invoice for Acme Corp with these line items..."*
- *"Extract the text from report.pdf"*
- *"Merge these 3 PDFs and add a CONFIDENTIAL watermark"*
- *"Fill the form fields in application.pdf with this data..."*
- *"What are the form fields in this PDF?"*

## Package Overview

| Package | Description |
|---------|-------------|
| `gofpdf` (root) | PDF document generator — the core library |
| `reader/` | PDF parser — open, inspect, extract text |
| `table/` | High-level table builder with styling |
| `pageops/` | Merge, split, rotate, watermark operations |
| `form/` | Create, fill, and flatten interactive forms |
| `sign/` | Digital signatures — sign and verify |
| `doctpl/` | JSON template DSL — declarative PDF generation |
| `mcp/` | MCP server — expose PDF tools to AI assistants |
| `cmd/gofpdf-mcp/` | MCP server binary for Claude Desktop |
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

All existing code continues to work. The new packages (`reader`, `table`, `pageops`, `form`, `sign`, `doctpl`, `mcp`) are additive — they don't change the core API.

## JSON Template Schema Reference

The `doctpl` package accepts JSON documents with the following structure:

```json
{
  "title": "string",
  "author": "string",
  "subject": "string",
  "pageSize": "A4 | Letter | Legal",
  "unit": "mm | cm | in | pt",
  "margin": {"top": 0, "right": 0, "bottom": 0, "left": 0},
  "font": {"family": "Helvetica", "style": "", "size": 11},
  "header": {"text": "...", "align": "L|C|R"},
  "footer": {"text": "Page {page}", "align": "C"},
  "pages": [{"elements": [...]}]
}
```

### Supported Element Types

| Type | Key Fields | Description |
|------|-----------|-------------|
| `heading` | `text`, `level` (1–6), `align`, `font`, `color` | Section heading with automatic sizing |
| `paragraph` | `text`, `align`, `font`, `color` | Body text with word wrapping |
| `table` | `columns` [{header, width, align}], `rows` [[...]], `headerStyle`, `cellStyle` | Data table with styled headers and alternating rows |
| `list` | `items` [...], `ordered`, `bullet` | Bulleted or numbered list |
| `image` | `src`, `x`, `y`, `width`, `height` | Embedded image (JPEG, PNG, GIF) |
| `line` | `x1`, `y1`, `x2`, `y2`, `lineWidth`, `color` | Arbitrary line |
| `rect` | `x`, `y`, `width`, `height`, `fillColor`, `border` | Rectangle shape |
| `spacer` | `spacerHeight` | Vertical whitespace |
| `hr` | `lineWidth`, `color` | Horizontal rule across the page |

## MCP Server Reference

The `gofpdf-mcp` binary exposes the following tools and resources via the Model Context Protocol:

### Tools

| Tool | Description |
|------|-------------|
| `create_pdf` | Create a PDF from a JSON template. Accepts `template` (object) and optional `outputPath` (string). |
| `read_pdf` | Read PDF metadata (version, page count, title, author). Accepts `path`. |
| `read_pdf_text` | Extract text content from specific or all pages. Accepts `path` and optional `pages` array. |
| `merge_pdfs` | Merge multiple PDFs into one. Accepts `inputPaths` array and `outputPath`. |
| `add_watermark` | Add a text watermark. Accepts `inputPath`, `outputPath`, `text`, and optional `fontSize`, `opacity`, `angle`. |
| `add_page_numbers` | Add page numbers. Accepts `inputPath`, `outputPath`, and optional `format`, `position`. |
| `fill_form` | Fill form fields. Accepts `inputPath`, `outputPath`, and `values` object (field name to value). |
| `flatten_form` | Flatten form fields to static content. Accepts `inputPath` and `outputPath`. |
| `rotate_pages` | Rotate pages by 90/180/270 degrees. Accepts `inputPath`, `outputPath`, `angle`, and optional `pages` array. |
| `pdf_info` | Get detailed PDF info (metadata, pages, form fields, dimensions). Accepts `path`. |

### Resources

| URI | Description |
|-----|-------------|
| `pdf://text?path=...` | Full text content of a PDF |
| `pdf://metadata?path=...` | Document metadata (title, author, version, page count) |
| `pdf://pages?path=...` | Page dimensions and rotation info |
| `pdf://form-fields?path=...` | Form field names, types, values, and options |

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
