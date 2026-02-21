// Package pageops provides operations for manipulating existing PDF documents,
// including merging, splitting, watermarking, and rotating pages.
//
// It uses the reader package to parse input PDFs and the gofpdi contrib package
// to import pages as templates into new PDF documents.
package pageops

import (
	"fmt"
	"io"
	"os"

	gofpdf "github.com/lvillar/gofpdf"
	"github.com/lvillar/gofpdf/contrib/gofpdi"
	"github.com/lvillar/gofpdf/reader"
)

// Position specifies where to place an element on a page.
type Position int

const (
	Center      Position = iota
	TopLeft
	TopCenter
	TopRight
	BottomLeft
	BottomCenter
	BottomRight
)

const (
	// Default A4 page dimensions in points (72 dpi).
	defaultPageWidth  = 595.28
	defaultPageHeight = 841.89
)

// newBasePDF creates a new PDF configured for page operations (no auto page break).
func newBasePDF() (*gofpdf.Fpdf, *gofpdi.Importer) {
	pdf := gofpdf.New("P", "pt", "A4", "")
	pdf.SetAutoPageBreak(false, 0)
	return pdf, gofpdi.NewImporter()
}

// addImportedPage imports a page from source and adds it to the PDF with default dimensions.
func addImportedPage(pdf *gofpdf.Fpdf, imp *gofpdi.Importer, sourceFile string, pageNum int) (pw, ph float64) {
	tplID, pw, ph := importPage(pdf, imp, sourceFile, pageNum)
	if pw == 0 || ph == 0 {
		pw = defaultPageWidth
		ph = defaultPageHeight
	}
	pdf.AddPageFormat("P", gofpdf.SizeType{Wd: pw, Ht: ph})
	imp.UseImportedTemplate(pdf, tplID, 0, 0, pw, ph)
	return pw, ph
}

// buildPageSet creates a map of selected page numbers.
// If pages is nil, all pages 1..pageCount are selected.
func buildPageSet(pages []int, pageCount int) map[int]bool {
	m := make(map[int]bool)
	if pages == nil {
		for i := 1; i <= pageCount; i++ {
			m[i] = true
		}
	} else {
		for _, p := range pages {
			m[p] = true
		}
	}
	return m
}

// importPage imports a single page from a source file into the target PDF.
// Returns the template ID and page dimensions.
func importPage(pdf *gofpdf.Fpdf, imp *gofpdi.Importer, sourceFile string, pageNum int) (tplID int, w, h float64) {
	tplID = imp.ImportPage(pdf, sourceFile, pageNum, "/MediaBox")
	sizes := imp.GetPageSizes()
	if dims, ok := sizes[pageNum]; ok {
		if mb, ok := dims["/MediaBox"]; ok {
			w = mb["w"]
			h = mb["h"]
		}
	}
	return
}

// getPageCount returns the number of pages in a PDF file.
func getPageCount(filename string) (int, error) {
	doc, err := reader.Open(filename)
	if err != nil {
		return 0, fmt.Errorf("pageops: reading %s: %w", filename, err)
	}
	return doc.NumPages(), nil
}

// getPageCountFromReader returns the number of pages from a reader.
func getPageCountFromReader(r io.ReadSeeker) (int, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return 0, fmt.Errorf("pageops: reading input: %w", err)
	}
	doc, err := reader.ReadFrom(io.NopCloser(io.NewSectionReader(newBytesReaderAt(data), 0, int64(len(data)))))
	if err != nil {
		return 0, err
	}
	return doc.NumPages(), nil
}

// writePDF writes the PDF to a writer.
func writePDF(pdf *gofpdf.Fpdf, w io.Writer) error {
	return pdf.Output(w)
}

// writePDFToFile writes the PDF to a file.
func writePDFToFile(pdf *gofpdf.Fpdf, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("pageops: creating %s: %w", filename, err)
	}
	defer f.Close()
	return pdf.Output(f)
}

// bytesReaderAt wraps a byte slice as an io.ReaderAt.
type bytesReaderAt struct {
	data []byte
}

func newBytesReaderAt(data []byte) *bytesReaderAt {
	return &bytesReaderAt{data: data}
}

func (b *bytesReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	if off >= int64(len(b.data)) {
		return 0, io.EOF
	}
	n = copy(p, b.data[off:])
	if n < len(p) {
		err = io.EOF
	}
	return
}
