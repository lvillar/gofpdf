// Command gofpdf-mcp is an MCP (Model Context Protocol) server that exposes
// PDF generation and manipulation capabilities to AI assistants.
//
// # Installation
//
//	go install github.com/lvillar/gofpdf/cmd/gofpdf-mcp@latest
//
// # Configuration for Claude Desktop
//
// Add to ~/.config/claude/claude_desktop_config.json:
//
//	{
//	  "mcpServers": {
//	    "gofpdf": {
//	      "command": "gofpdf-mcp"
//	    }
//	  }
//	}
//
// # Available Tools
//
//   - create_pdf: Create PDFs from JSON templates
//   - read_pdf: Read PDF metadata
//   - read_pdf_text: Extract text from PDFs
//   - merge_pdfs: Merge multiple PDFs
//   - add_watermark: Add text watermarks
//   - add_page_numbers: Add page numbers
//   - fill_form: Fill PDF form fields
//   - flatten_form: Flatten PDF forms
//   - rotate_pages: Rotate PDF pages
//   - pdf_info: Get detailed PDF information
//
// # Available Resources
//
//   - pdf://text?path=... : Extract text content
//   - pdf://metadata?path=... : Get document metadata
//   - pdf://pages?path=... : Get page information
//   - pdf://form-fields?path=... : List form fields
package main

import (
	"fmt"
	"os"

	"github.com/lvillar/gofpdf/mcp"
)

func main() {
	server := mcp.NewServer()

	mcp.RegisterDefaultTools(server)
	mcp.RegisterDefaultResources(server)

	if err := server.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "gofpdf-mcp: %v\n", err)
		os.Exit(1)
	}
}
