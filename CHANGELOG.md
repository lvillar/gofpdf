# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.2.0] - 2026-11-17

Typography that matches the browser: pair kerning and letter spacing, so that
a PDF can reproduce an HTML design to the tenth of a millimeter, and SVG
paths that can be filled, for logos. Kerning and letter spacing are off unless
requested: existing documents keep their layout byte for byte.

### Added

- **`Fpdf.SetKerning(enable)`** — pair kerning for UTF-8 fonts, read from
  the `kern` feature of the font's GPOS table (pairs listed one by one, pairs
  by class and extension lookups, for the default language system of each
  script). With kerning enabled, `Text`, `Cell`, `CellFormat`, `MultiCell` and
  `Write` tighten or loosen pairs such as "AV" or "To" the way browsers and
  word processors do, and `GetStringWidth`, alignment, justification and
  underlines measure the kerned text. Widths match the ones Chrome (HarfBuzz)
  computes for the same font. It is not applied to right-to-left text. The
  amount of data read from a font is bounded, so a damaged or hostile file
  cannot take time or memory out of proportion to its size.
- **`Fpdf.SetCharSpacing(space)`** — letter spacing (tracking): space added
  after each character, in the unit of the document, negative to tighten.
  It is retained from page to page and `GetStringWidth`, alignment and
  underlines take it into account. It works with every kind of font. Headers,
  footers and templates are drawn without the spacing of the text around
  them.
- **Line breaking aware of kerning and letter spacing** — `MultiCell`,
  `Write`, `SplitText` and `SplitLines` measure the text as it will be
  written, so a line with letter spacing does not run past the edge of its
  cell and kerned text breaks where it really ends.
- **`Fpdf.SVGBasicDraw(sb, scale, styleStr)`** — paints a basic SVG image
  building each of its paths as a single PDF path, so that it can be filled
  (`"F"`, `"F*"`, `"DF"`, …). `SVGBasicWrite` strokes every segment on its
  own and cannot fill, which ruled out logos and other filled shapes.
- **All the SVG path commands in `SVGBasicParse`** — smooth curves (`S`,
  `s`, `T`, `t`) and elliptical arcs (`A`, `a`) are now understood. They are
  returned as the `C` and `Q` segments they stand for, so existing code that
  walks `SVGBasicType.Segments` keeps working.
- **`viewBox` in `SVGBasicParse`** — a document without `width` and
  `height`, or with them given as percentages, takes its size from the
  `viewBox`. `width` and `height` in pixels (`"24px"`) are accepted.
- Fuzz targets for the SVG path parser and the GPOS reader.

### Changed

- `SVGBasicParse` now applies the `viewBox` of a document that also has
  `width` and `height`: the path coordinates are converted from the units of
  the `viewBox` to those of the size, which is what the SVG format means by
  it. An icon drawn on a 24 x 24 grid with a size of 96 x 96 used to come out
  at a quarter of its size. Documents whose `viewBox` is `0 0 width height`,
  or that have none, are read exactly as before.

### Fixed

- `SVGBasicParse` rejected path data written without optional separators
  (`M1.5.5-2`, `a10 10 0 01-10 10`), which is how optimized SVG files are
  saved.
- `SVGBasicParse` measured a relative command that follows a closepath
  (`zm1 1`) from the last point drawn instead of from the start of the
  closed subpath.

## [1.1.0] - 2026-05-08

LLM-friendly improvements to the MCP server and the `doctpl` template DSL.
Fully backward compatible: existing templates and existing MCP clients keep
working unchanged.

### Added

- **`doctpl.Validate(jsonBytes) ([]ValidationError, error)`** — lint a
  template without rendering. Returns a structured list of problems
  (`Path`, `Field`, `Message`) so an LLM (or human) can self-correct
  before generating a PDF. Companion `ValidateDocument(*Document)` works
  on already-parsed documents.
- **`validate_template` MCP tool** — exposes `doctpl.Validate` over the
  Model Context Protocol so LLM clients (Claude Desktop, etc.) can lint
  templates in a single round-trip with no PDF written to disk.
- **Rich `inputSchema` for the `create_pdf` MCP tool** — the JSON Schema
  now describes every supported field of a `doctpl.Document` (page sizes,
  units, margins, fonts, colors, page elements, table columns, list items)
  with descriptions and enums. LLMs can now author templates correctly
  on the first try instead of guessing field names.
- **`reader/example_test.go`** — runnable godoc example showing how to
  read a PDF and inspect its metadata.
- **`sign/example_test.go`** — runnable godoc example demonstrating the
  full sign + verify flow with a self-signed ECDSA certificate.

## [1.0.0] - 2026-05-08

First tagged release of the `lvillar/gofpdf` library. Production-ready.

The core `gofpdf` API (root package) remains backward compatible with the
upstream `jung-kurt/gofpdf` API; the only breaking change relative to upstream
is the module path, which is now `github.com/lvillar/gofpdf`.

### Added

- **PDF reader** (`reader/`) — parse existing PDFs, extract text, navigate the
  page tree, decompress FlateDecode streams, decrypt RC4 40-bit and 128-bit
  password-protected documents.
- **High-level tables** (`table/`) — declarative table builder with functional
  options, automatic column-width calculation, multi-page tables with repeated
  headers, alternating row colors and styled cells.
- **Page operations** (`pageops/`) — merge multiple PDFs, split by page ranges,
  rotate pages (90/180/270°), add text watermarks and page numbers.
- **Interactive forms** (`form/`) — create forms with text fields, checkboxes,
  dropdowns and radio buttons; fill existing PDF forms programmatically;
  flatten interactive fields into static content.
- **Digital signatures** (`sign/`) — sign PDFs with PKCS#7 detached signatures
  using ECDSA or RSA keys; verify signatures and detect tampering.
  *(Foundation implementation; full PAdES-B and LTV support planned for
  a future release.)*
- **JSON template DSL** (`doctpl/`) — declarative PDF generation from JSON
  documents with headings, paragraphs, tables, lists, images, lines,
  rectangles, horizontal rules and spacers. Designed to be friendly for
  LLM-generated content.
- **MCP server** (`mcp/`, `cmd/gofpdf-mcp/`) — Model Context Protocol server
  exposing 10 PDF tools (`create_pdf`, `read_pdf`, `read_pdf_text`,
  `merge_pdfs`, `add_watermark`, `add_page_numbers`, `fill_form`,
  `flatten_form`, `rotate_pages`, `pdf_info`) and 4 resources
  (`pdf://text`, `pdf://metadata`, `pdf://pages`, `pdf://form-fields`)
  via JSON-RPC 2.0 over stdio. Zero external dependencies.
- Modernization to Go 1.23: replaced `io/ioutil` with `os.ReadFile`,
  `io.ReadAll`, `io.Discard`; introduced typed constants
  (`Orientation`, `Unit`, `PageSize`); added `NewDocument` with functional
  options; replaced `strings.Replace(..., -1)` with `strings.ReplaceAll`.

### Changed

- Module path migrated to `github.com/lvillar/gofpdf` (was
  `github.com/jung-kurt/gofpdf` in the upstream). Existing code only needs
  to update its import paths; the API surface is preserved.

### Fixed

- `CompareBytes` length-mismatch detection for PDF comparison helpers.
- Pre-compilation of regexes in the TTF parser and other hot paths.
- Edge cases in core utilities; removal of dead code; consolidation of
  duplicated logic.

[Unreleased]: https://github.com/lvillar/gofpdf/compare/v1.2.0...HEAD
[1.2.0]: https://github.com/lvillar/gofpdf/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/lvillar/gofpdf/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/lvillar/gofpdf/releases/tag/v1.0.0
