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
	pdf, err := buildMergedPDF(inputPaths)
	if err != nil {
		return err
	}
	return writePDFToFile(pdf, outputPath)
}

// Merge combines multiple PDF files and writes the result to w.
func Merge(w io.Writer, inputPaths ...string) error {
	pdf, err := buildMergedPDF(inputPaths)
	if err != nil {
		return err
	}
	return writePDF(pdf, w)
}

func buildMergedPDF(inputPaths []string) (*gofpdf.Fpdf, error) {
	if len(inputPaths) == 0 {
		return nil, fmt.Errorf("pageops: no input files provided")
	}

	pdf, _ := newBasePDF()

	for _, inputPath := range inputPaths {
		pageCount, err := getPageCount(inputPath)
		if err != nil {
			return nil, fmt.Errorf("pageops: merging %s: %w", inputPath, err)
		}

		imp := gofpdi.NewImporter()
		for i := 1; i <= pageCount; i++ {
			addImportedPage(pdf, imp, inputPath, i)
		}
	}

	if pdf.Err() {
		return nil, fmt.Errorf("pageops: merge: %w", pdf.Error())
	}
	return pdf, nil
}
