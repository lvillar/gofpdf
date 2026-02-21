package pageops

import (
	"fmt"
	"io"
	"path/filepath"

	gofpdf "github.com/lvillar/gofpdf"
	"github.com/lvillar/gofpdf/contrib/gofpdi"
)

// SplitToFiles splits a PDF into individual pages, saving each to outputDir.
// Files are named page_001.pdf, page_002.pdf, etc.
func SplitToFiles(inputPath, outputDir string) error {
	pageCount, err := getPageCount(inputPath)
	if err != nil {
		return err
	}

	for i := 1; i <= pageCount; i++ {
		outputPath := filepath.Join(outputDir, fmt.Sprintf("page_%03d.pdf", i))
		if err := ExtractPagesToFile(inputPath, outputPath, i); err != nil {
			return fmt.Errorf("pageops: splitting page %d: %w", i, err)
		}
	}

	return nil
}

// ExtractPages extracts specific pages from a PDF and writes them to w.
// Page numbers are 1-based.
func ExtractPages(w io.Writer, inputPath string, pages ...int) error {
	if len(pages) == 0 {
		return fmt.Errorf("pageops: no pages specified")
	}

	pdf := gofpdf.New("P", "pt", "A4", "")
	pdf.SetAutoPageBreak(false, 0)
	imp := gofpdi.NewImporter()

	for _, pageNum := range pages {
		tplID, pw, ph := importPage(pdf, imp, inputPath, pageNum)
		if pw == 0 || ph == 0 {
			pw = 595.28
			ph = 841.89
		}
		pdf.AddPageFormat("P", gofpdf.SizeType{Wd: pw, Ht: ph})
		imp.UseImportedTemplate(pdf, tplID, 0, 0, pw, ph)
	}

	return writePDF(pdf, w)
}

// ExtractPagesToFile extracts specific pages and saves to a file.
func ExtractPagesToFile(inputPath, outputPath string, pages ...int) error {
	if len(pages) == 0 {
		return fmt.Errorf("pageops: no pages specified")
	}

	pdf := gofpdf.New("P", "pt", "A4", "")
	pdf.SetAutoPageBreak(false, 0)
	imp := gofpdi.NewImporter()

	for _, pageNum := range pages {
		tplID, pw, ph := importPage(pdf, imp, inputPath, pageNum)
		if pw == 0 || ph == 0 {
			pw = 595.28
			ph = 841.89
		}
		pdf.AddPageFormat("P", gofpdf.SizeType{Wd: pw, Ht: ph})
		imp.UseImportedTemplate(pdf, tplID, 0, 0, pw, ph)
	}

	return writePDFToFile(pdf, outputPath)
}

// ExtractPageRange extracts a range of pages (inclusive, 1-based).
func ExtractPageRange(w io.Writer, inputPath string, start, end int) error {
	if start < 1 || end < start {
		return fmt.Errorf("pageops: invalid page range [%d, %d]", start, end)
	}

	pages := make([]int, 0, end-start+1)
	for i := start; i <= end; i++ {
		pages = append(pages, i)
	}

	return ExtractPages(w, inputPath, pages...)
}
