# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/lvillar/gofpdf/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/lvillar/gofpdf/releases/tag/v1.0.0
