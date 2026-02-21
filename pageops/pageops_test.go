package pageops_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	gofpdf "github.com/lvillar/gofpdf"
	"github.com/lvillar/gofpdf/pageops"
	"github.com/lvillar/gofpdf/reader"
)

// createTestPDF generates a simple test PDF file with the given number of pages.
func createTestPDF(t *testing.T, filename string, numPages int) {
	t.Helper()
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetFont("Helvetica", "", 14)
	for i := 1; i <= numPages; i++ {
		pdf.AddPage()
		pdf.Text(20, 30, fmt.Sprintf("Page %d of %d", i, numPages))
	}
	if err := pdf.OutputFileAndClose(filename); err != nil {
		t.Fatalf("creating test PDF: %v", err)
	}
}

func TestMergeFiles(t *testing.T) {
	dir := t.TempDir()

	// Create two test PDFs
	file1 := filepath.Join(dir, "doc1.pdf")
	file2 := filepath.Join(dir, "doc2.pdf")
	output := filepath.Join(dir, "merged.pdf")

	createTestPDF(t, file1, 2)
	createTestPDF(t, file2, 3)

	// Merge them
	if err := pageops.MergeFiles(output, file1, file2); err != nil {
		t.Fatalf("merge: %v", err)
	}

	// Verify result
	doc, err := reader.Open(output)
	if err != nil {
		t.Fatalf("reading merged PDF: %v", err)
	}
	if doc.NumPages() != 5 {
		t.Errorf("expected 5 pages, got %d", doc.NumPages())
	}
	t.Logf("Merged PDF: %d pages", doc.NumPages())
}

func TestMergeToWriter(t *testing.T) {
	dir := t.TempDir()
	file1 := filepath.Join(dir, "doc1.pdf")
	file2 := filepath.Join(dir, "doc2.pdf")

	createTestPDF(t, file1, 1)
	createTestPDF(t, file2, 1)

	var buf bytes.Buffer
	if err := pageops.Merge(&buf, file1, file2); err != nil {
		t.Fatalf("merge: %v", err)
	}

	doc, err := reader.ReadFrom(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("reading merged PDF: %v", err)
	}
	if doc.NumPages() != 2 {
		t.Errorf("expected 2 pages, got %d", doc.NumPages())
	}
}

func TestSplitToFiles(t *testing.T) {
	dir := t.TempDir()
	inputFile := filepath.Join(dir, "input.pdf")
	outputDir := filepath.Join(dir, "output")
	os.MkdirAll(outputDir, 0755)

	createTestPDF(t, inputFile, 3)

	if err := pageops.SplitToFiles(inputFile, outputDir); err != nil {
		t.Fatalf("split: %v", err)
	}

	// Verify each page file exists and has 1 page
	for i := 1; i <= 3; i++ {
		pageFile := filepath.Join(outputDir, fmt.Sprintf("page_%03d.pdf", i))
		doc, err := reader.Open(pageFile)
		if err != nil {
			t.Errorf("page %d: %v", i, err)
			continue
		}
		if doc.NumPages() != 1 {
			t.Errorf("page %d: expected 1 page, got %d", i, doc.NumPages())
		}
	}
	t.Logf("Split into 3 individual page files")
}

func TestExtractPages(t *testing.T) {
	dir := t.TempDir()
	inputFile := filepath.Join(dir, "input.pdf")
	createTestPDF(t, inputFile, 5)

	var buf bytes.Buffer
	if err := pageops.ExtractPages(&buf, inputFile, 2, 4); err != nil {
		t.Fatalf("extract: %v", err)
	}

	doc, err := reader.ReadFrom(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("reading extracted PDF: %v", err)
	}
	if doc.NumPages() != 2 {
		t.Errorf("expected 2 pages, got %d", doc.NumPages())
	}
}

func TestExtractPageRange(t *testing.T) {
	dir := t.TempDir()
	inputFile := filepath.Join(dir, "input.pdf")
	createTestPDF(t, inputFile, 5)

	var buf bytes.Buffer
	if err := pageops.ExtractPageRange(&buf, inputFile, 2, 4); err != nil {
		t.Fatalf("extract range: %v", err)
	}

	doc, err := reader.ReadFrom(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("reading PDF: %v", err)
	}
	if doc.NumPages() != 3 {
		t.Errorf("expected 3 pages (2-4), got %d", doc.NumPages())
	}
}

func TestAddTextWatermark(t *testing.T) {
	dir := t.TempDir()
	inputFile := filepath.Join(dir, "input.pdf")
	outputFile := filepath.Join(dir, "watermarked.pdf")
	createTestPDF(t, inputFile, 2)

	wm := pageops.TextWatermark{
		Text:    "CONFIDENTIAL",
		Opacity: 0.3,
		Angle:   45,
	}

	if err := pageops.AddTextWatermarkToFile(inputFile, outputFile, wm); err != nil {
		t.Fatalf("watermark: %v", err)
	}

	doc, err := reader.Open(outputFile)
	if err != nil {
		t.Fatalf("reading watermarked PDF: %v", err)
	}
	if doc.NumPages() != 2 {
		t.Errorf("expected 2 pages, got %d", doc.NumPages())
	}

	// Verify the watermarked file is larger (has extra content)
	origInfo, _ := os.Stat(inputFile)
	wmInfo, _ := os.Stat(outputFile)
	if wmInfo.Size() <= origInfo.Size() {
		t.Errorf("watermarked file should be larger: orig=%d, wm=%d", origInfo.Size(), wmInfo.Size())
	}
	t.Logf("Watermarked: orig=%d bytes, watermarked=%d bytes", origInfo.Size(), wmInfo.Size())
}

func TestAddPageNumbers(t *testing.T) {
	dir := t.TempDir()
	inputFile := filepath.Join(dir, "input.pdf")
	outputFile := filepath.Join(dir, "numbered.pdf")
	createTestPDF(t, inputFile, 3)

	style := pageops.PageNumberStyle{
		Format:   "Page %d of %d",
		Position: pageops.BottomCenter,
	}

	if err := pageops.AddPageNumbersToFile(inputFile, outputFile, style); err != nil {
		t.Fatalf("page numbers: %v", err)
	}

	doc, err := reader.Open(outputFile)
	if err != nil {
		t.Fatalf("reading numbered PDF: %v", err)
	}
	if doc.NumPages() != 3 {
		t.Errorf("expected 3 pages, got %d", doc.NumPages())
	}
	t.Logf("Page numbers added to %d pages", doc.NumPages())
}

func TestMergeNoInputs(t *testing.T) {
	var buf bytes.Buffer
	if err := pageops.Merge(&buf); err == nil {
		t.Error("expected error for empty merge")
	}
}

func TestExtractPagesNoPages(t *testing.T) {
	var buf bytes.Buffer
	if err := pageops.ExtractPages(&buf, "nonexistent.pdf"); err == nil {
		t.Error("expected error for no pages")
	}
}

func TestInvalidPageRange(t *testing.T) {
	var buf bytes.Buffer
	if err := pageops.ExtractPageRange(&buf, "any.pdf", 5, 2); err == nil {
		t.Error("expected error for invalid range")
	}
}

func TestInvalidRotationAngle(t *testing.T) {
	var buf bytes.Buffer
	if err := pageops.RotatePages(&buf, "any.pdf", 45, nil); err == nil {
		t.Error("expected error for invalid rotation angle")
	}
}
