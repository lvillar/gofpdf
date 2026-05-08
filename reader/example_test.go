package reader_test

import (
	"bytes"
	"fmt"

	gofpdf "github.com/lvillar/gofpdf"
	"github.com/lvillar/gofpdf/reader"
)

// ExampleOpen demonstrates reading a PDF, inspecting its metadata, and
// iterating over its pages.
func ExampleReadFrom() {
	// Build a small in-memory PDF so the example is self-contained.
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle("Quarterly Report", true)
	pdf.SetAuthor("Acme Analytics", true)
	pdf.SetFont("Helvetica", "B", 16)

	pdf.AddPage()
	pdf.Cell(0, 10, "Page 1: Summary")
	pdf.AddPage()
	pdf.Cell(0, 10, "Page 2: Details")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		fmt.Println(err)
		return
	}

	// Now read it back.
	doc, err := reader.ReadFrom(&buf)
	if err != nil {
		fmt.Println(err)
		return
	}

	meta := doc.Metadata()
	fmt.Printf("Pages: %d\n", doc.NumPages())
	fmt.Printf("Title: %s\n", meta["Title"])
	fmt.Printf("Author: %s\n", meta["Author"])

	// Output:
	// Pages: 2
	// Title: Quarterly Report
	// Author: Acme Analytics
}
