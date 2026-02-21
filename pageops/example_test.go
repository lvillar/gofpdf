package pageops_test

import (
	"fmt"
	"os"
	"path/filepath"

	gofpdf "github.com/lvillar/gofpdf"
	"github.com/lvillar/gofpdf/internal/example"
	"github.com/lvillar/gofpdf/pageops"
)

// createExamplePDF creates a simple PDF with labeled pages for use in examples.
func createExamplePDF(filename string, numPages int, label string) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetFont("Helvetica", "", 24)
	for i := 1; i <= numPages; i++ {
		pdf.AddPage()
		pdf.SetXY(20, 40)
		pdf.Cell(0, 15, fmt.Sprintf("%s - Page %d", label, i))
		pdf.Ln(20)
		pdf.SetFont("Helvetica", "", 12)
		pdf.MultiCell(170, 6, fmt.Sprintf(
			"This is page %d of the %s document. "+
				"It was generated to demonstrate the pageops merge functionality.",
			i, label), "", "", false)
		pdf.SetFont("Helvetica", "", 24)
	}
	return pdf.OutputFileAndClose(filename)
}

// ExampleMergeFiles demonstrates merging multiple PDF files into one.
func ExampleMergeFiles() {
	dir := example.PdfDir()

	// Create two source PDFs
	file1 := filepath.Join(dir, "_merge_input1.pdf")
	file2 := filepath.Join(dir, "_merge_input2.pdf")
	defer os.Remove(file1)
	defer os.Remove(file2)

	if err := createExamplePDF(file1, 2, "Document A"); err != nil {
		fmt.Println(err)
		return
	}
	if err := createExamplePDF(file2, 1, "Document B"); err != nil {
		fmt.Println(err)
		return
	}

	// Merge them
	outFile := example.Filename("PageOps_Merge")
	err := pageops.MergeFiles(outFile, file1, file2)
	example.Summary(err, outFile)
	// Output:
	// Successfully generated ../pdf/PageOps_Merge.pdf
}
