package form_test

import (
	"bytes"
	"testing"

	gofpdf "github.com/lvillar/gofpdf"
	"github.com/lvillar/gofpdf/form"
	"github.com/lvillar/gofpdf/reader"
)

func generateFilledFormPDF(t *testing.T) []byte {
	t.Helper()
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetFont("Helvetica", "", 12)
	pdf.AddPage()
	pdf.Text(10, 10, "Fill test form")

	fb := form.NewFormBuilder(pdf)
	fb.AddTextField("name", 1, 40, 5, 80, 10)
	fb.AddTextField("email", 1, 40, 20, 80, 10)
	fb.AddDropdown("country", 1, 40, 35, 80, 8, []string{"USA", "Canada", "Mexico"})

	if err := fb.Build(); err != nil {
		t.Fatalf("build form: %v", err)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		t.Fatalf("output: %v", err)
	}
	return buf.Bytes()
}

func TestFillTextField(t *testing.T) {
	pdfData := generateFilledFormPDF(t)

	var output bytes.Buffer
	err := form.Fill(bytes.NewReader(pdfData), &output, map[string]string{
		"name":  "John Doe",
		"email": "john@example.com",
	})
	if err != nil {
		t.Fatalf("Fill: %v", err)
	}

	// Verify the filled PDF contains the new values
	result := output.Bytes()
	if !bytes.Contains(result, []byte("John Doe")) {
		t.Error("expected filled PDF to contain 'John Doe'")
	}
	if !bytes.Contains(result, []byte("john@example.com")) {
		t.Error("expected filled PDF to contain 'john@example.com'")
	}

	// Verify it's still a valid PDF
	doc, err := reader.ReadFrom(bytes.NewReader(result))
	if err != nil {
		t.Fatalf("reading filled PDF: %v", err)
	}
	if doc.NumPages() != 1 {
		t.Errorf("expected 1 page, got %d", doc.NumPages())
	}

	t.Logf("Filled PDF: %d bytes (original: %d bytes)", output.Len(), len(pdfData))
}

func TestFillNonExistentField(t *testing.T) {
	pdfData := generateFilledFormPDF(t)

	var output bytes.Buffer
	err := form.Fill(bytes.NewReader(pdfData), &output, map[string]string{
		"nonexistent": "value",
	})
	if err == nil {
		t.Error("expected error when filling non-existent field")
	}
}

func TestFillEmptyValues(t *testing.T) {
	pdfData := generateFilledFormPDF(t)

	var output bytes.Buffer
	err := form.Fill(bytes.NewReader(pdfData), &output, map[string]string{})
	if err != nil {
		t.Fatalf("Fill with empty values: %v", err)
	}

	// Should be same size (just a copy)
	if output.Len() != len(pdfData) {
		t.Errorf("expected same size output, got %d vs %d", output.Len(), len(pdfData))
	}
}

func TestFillNoFormPDF(t *testing.T) {
	// Create a PDF without forms
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetFont("Helvetica", "", 12)
	pdf.AddPage()
	pdf.Text(10, 10, "No forms here")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		t.Fatalf("output: %v", err)
	}

	var output bytes.Buffer
	err := form.Fill(bytes.NewReader(buf.Bytes()), &output, map[string]string{
		"field": "value",
	})
	if err == nil {
		t.Error("expected error when filling non-form PDF")
	}
}

func TestFlattenForm(t *testing.T) {
	pdfData := generateFilledFormPDF(t)

	var output bytes.Buffer
	err := form.Flatten(bytes.NewReader(pdfData), &output)
	if err != nil {
		t.Fatalf("Flatten: %v", err)
	}

	result := output.Bytes()

	// After flattening, /AcroForm should be removed
	if bytes.Contains(result, []byte("/AcroForm")) {
		t.Error("flattened PDF should not contain /AcroForm")
	}

	// The /FT entries should be removed
	if bytes.Contains(result, []byte("/FT /Tx")) {
		t.Error("flattened PDF should not contain /FT /Tx")
	}

	// Should still be a valid PDF
	doc, err := reader.ReadFrom(bytes.NewReader(result))
	if err != nil {
		t.Fatalf("reading flattened PDF: %v", err)
	}
	if doc.NumPages() != 1 {
		t.Errorf("expected 1 page, got %d", doc.NumPages())
	}

	t.Logf("Flattened PDF: %d bytes (original: %d bytes)", output.Len(), len(pdfData))
}

func TestFlattenNoForm(t *testing.T) {
	// Create a PDF without forms
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetFont("Helvetica", "", 12)
	pdf.AddPage()
	pdf.Text(10, 10, "No forms")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		t.Fatalf("output: %v", err)
	}

	var output bytes.Buffer
	err := form.Flatten(bytes.NewReader(buf.Bytes()), &output)
	if err != nil {
		t.Fatalf("Flatten no-form: %v", err)
	}

	// Should be same content
	if output.Len() != buf.Len() {
		t.Errorf("expected same size for no-form flatten, got %d vs %d", output.Len(), buf.Len())
	}
}

func TestFillThenFlatten(t *testing.T) {
	pdfData := generateFilledFormPDF(t)

	// Step 1: Fill the form
	var filled bytes.Buffer
	err := form.Fill(bytes.NewReader(pdfData), &filled, map[string]string{
		"name":    "Jane Smith",
		"email":   "jane@example.com",
		"country": "Canada",
	})
	if err != nil {
		t.Fatalf("Fill: %v", err)
	}

	// Step 2: Flatten the filled form
	var flattened bytes.Buffer
	err = form.Flatten(bytes.NewReader(filled.Bytes()), &flattened)
	if err != nil {
		t.Fatalf("Flatten: %v", err)
	}

	result := flattened.Bytes()

	// Should contain the filled values as static content
	if !bytes.Contains(result, []byte("Jane Smith")) {
		t.Error("expected flattened PDF to contain 'Jane Smith'")
	}

	// Should not have AcroForm
	if bytes.Contains(result, []byte("/AcroForm")) {
		t.Error("flattened PDF should not contain /AcroForm")
	}

	// Should be valid
	doc, err := reader.ReadFrom(bytes.NewReader(result))
	if err != nil {
		t.Fatalf("reading result: %v", err)
	}
	if doc.NumPages() != 1 {
		t.Errorf("expected 1 page, got %d", doc.NumPages())
	}

	t.Logf("Fill+Flatten: original=%d, filled=%d, flattened=%d bytes",
		len(pdfData), filled.Len(), flattened.Len())
}
