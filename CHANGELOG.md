# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/lvillar/gofpdf/compare/v1.1.0...HEAD
[1.1.0]: https://github.com/lvillar/gofpdf/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/lvillar/gofpdf/releases/tag/v1.0.0
