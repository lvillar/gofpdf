package pageops

import (
	"fmt"
	"io"

	gofpdf "github.com/lvillar/gofpdf"
	"github.com/lvillar/gofpdf/contrib/gofpdi"
)

// MergeFiles combines multiple PDF files into a single output file.
// Pages are added in order: all pages from the first file, then all from the second, etc.
func MergeFiles(outputPath string, inputPaths ...string) error {
	if len(inputPaths) == 0 {
		return fmt.Errorf("pageops: no input files provided")
	}

	pdf := gofpdf.New("P", "pt", "A4", "")
	pdf.SetAutoPageBreak(false, 0)

	for _, inputPath := range inputPaths {
		if err := appendFile(pdf, inputPath); err != nil {
			return fmt.Errorf("pageops: merging %s: %w", inputPath, err)
		}
	}

	return writePDFToFile(pdf, outputPath)
}

// Merge combines multiple PDF files and writes the result to w.
func Merge(w io.Writer, inputPaths ...string) error {
	if len(inputPaths) == 0 {
		return fmt.Errorf("pageops: no input files provided")
	}

	pdf := gofpdf.New("P", "pt", "A4", "")
	pdf.SetAutoPageBreak(false, 0)

	for _, inputPath := range inputPaths {
		if err := appendFile(pdf, inputPath); err != nil {
			return fmt.Errorf("pageops: merging %s: %w", inputPath, err)
		}
	}

	return writePDF(pdf, w)
}

// appendFile imports all pages from a PDF file into the target PDF.
func appendFile(pdf *gofpdf.Fpdf, inputPath string) error {
	pageCount, err := getPageCount(inputPath)
	if err != nil {
		return err
	}

	imp := gofpdi.NewImporter()

	for i := 1; i <= pageCount; i++ {
		tplID, w, h := importPage(pdf, imp, inputPath, i)
		if w == 0 || h == 0 {
			w = 595.28 // A4 default
			h = 841.89
		}

		pdf.AddPageFormat("P", gofpdf.SizeType{Wd: w, Ht: h})
		imp.UseImportedTemplate(pdf, tplID, 0, 0, w, h)
	}

	return pdf.Error()
}
