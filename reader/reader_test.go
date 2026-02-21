package reader_test

import (
	"bytes"
	"testing"

	gofpdf "github.com/lvillar/gofpdf"
	"github.com/lvillar/gofpdf/reader"
)

// generateTestPDF creates a simple PDF with the given text content using gofpdf.
func generateTestPDF(t *testing.T, texts ...string) []byte {
	t.Helper()
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetFont("Helvetica", "", 12)

	for _, text := range texts {
		pdf.AddPage()
		pdf.Text(10, 20, text)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		t.Fatalf("generating test PDF: %v", err)
	}
	return buf.Bytes()
}

func TestOpenRoundTrip(t *testing.T) {
	data := generateTestPDF(t, "Hello World", "Page Two")

	doc, err := reader.ReadFrom(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("reading PDF: %v", err)
	}

	if doc.NumPages() != 2 {
		t.Errorf("expected 2 pages, got %d", doc.NumPages())
	}

	if doc.Version == "" {
		t.Error("expected non-empty PDF version")
	}
}

func TestPageAccess(t *testing.T) {
	data := generateTestPDF(t, "First", "Second", "Third")

	doc, err := reader.ReadFrom(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("reading PDF: %v", err)
	}

	// Valid page access
	for i := 1; i <= 3; i++ {
		page, err := doc.Page(i)
		if err != nil {
			t.Errorf("page %d: %v", i, err)
			continue
		}
		if page.Number != i {
			t.Errorf("page %d: number = %d", i, page.Number)
		}
		// A4 MediaBox should be approximately 595 x 842
		if page.MediaBox.Width() < 500 || page.MediaBox.Height() < 700 {
			t.Errorf("page %d: unexpected MediaBox: %v", i, page.MediaBox)
		}
	}

	// Invalid page access
	if _, err := doc.Page(0); err == nil {
		t.Error("expected error for page 0")
	}
	if _, err := doc.Page(4); err == nil {
		t.Error("expected error for page 4")
	}
}

func TestPagesIterator(t *testing.T) {
	data := generateTestPDF(t, "A", "B")

	doc, err := reader.ReadFrom(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("reading PDF: %v", err)
	}

	count := 0
	for num, page := range doc.Pages() {
		count++
		if page.Number != num {
			t.Errorf("iterator: page.Number=%d, num=%d", page.Number, num)
		}
	}
	if count != 2 {
		t.Errorf("iterator: expected 2 iterations, got %d", count)
	}
}

func TestTextExtraction(t *testing.T) {
	data := generateTestPDF(t, "Hello PDF Reader")

	doc, err := reader.ReadFrom(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("reading PDF: %v", err)
	}

	page, err := doc.Page(1)
	if err != nil {
		t.Fatalf("getting page 1: %v", err)
	}

	text, err := page.ExtractText()
	if err != nil {
		t.Fatalf("extracting text: %v", err)
	}

	if text == "" {
		t.Error("expected non-empty text extraction")
	}

	// The text should contain our input (exact matching depends on encoding)
	t.Logf("Extracted text: %q", text)
}

func TestMetadata(t *testing.T) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle("Test Document", false)
	pdf.SetAuthor("Test Author", false)
	pdf.SetFont("Helvetica", "", 12)
	pdf.AddPage()
	pdf.Text(10, 20, "Metadata test")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		t.Fatalf("generating PDF: %v", err)
	}

	doc, err := reader.ReadFrom(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("reading PDF: %v", err)
	}

	meta := doc.Metadata()
	if meta["Title"] != "Test Document" {
		t.Errorf("Title = %q, want %q", meta["Title"], "Test Document")
	}
	if meta["Author"] != "Test Author" {
		t.Errorf("Author = %q, want %q", meta["Author"], "Test Author")
	}
}

func TestMultiPageContentStream(t *testing.T) {
	data := generateTestPDF(t, "Page 1 content")

	doc, err := reader.ReadFrom(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("reading PDF: %v", err)
	}

	page, err := doc.Page(1)
	if err != nil {
		t.Fatalf("getting page: %v", err)
	}

	content, err := page.ContentStream()
	if err != nil {
		t.Fatalf("getting content stream: %v", err)
	}

	if len(content) == 0 {
		t.Error("expected non-empty content stream")
	}
	t.Logf("Content stream length: %d bytes", len(content))
}
