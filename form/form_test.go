package form_test

import (
	"bytes"
	"testing"

	gofpdf "github.com/lvillar/gofpdf"
	"github.com/lvillar/gofpdf/form"
	"github.com/lvillar/gofpdf/reader"
)

func TestTextFieldCreation(t *testing.T) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetFont("Helvetica", "", 12)
	pdf.AddPage()
	pdf.Text(10, 10, "Name:")

	fb := form.NewFormBuilder(pdf)
	fb.AddTextField("name", 1, 40, 5, 80, 10)
	fb.AddTextField("email", 1, 40, 20, 80, 10).SetRequired(true)

	if err := fb.Build(); err != nil {
		t.Fatalf("build: %v", err)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		t.Fatalf("output: %v", err)
	}

	// Verify the PDF is valid
	doc, err := reader.ReadFrom(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("reading PDF: %v", err)
	}
	if doc.NumPages() != 1 {
		t.Errorf("expected 1 page, got %d", doc.NumPages())
	}

	// Verify AcroForm is in the output
	pdfBytes := buf.Bytes()
	if !bytes.Contains(pdfBytes, []byte("/AcroForm")) {
		t.Error("expected /AcroForm in PDF output")
	}
	if !bytes.Contains(pdfBytes, []byte("/FT /Tx")) {
		t.Error("expected text field /FT /Tx in PDF output")
	}
	t.Logf("Form PDF with text fields: %d bytes", buf.Len())
}

func TestCheckboxCreation(t *testing.T) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetFont("Helvetica", "", 12)
	pdf.AddPage()
	pdf.Text(10, 10, "Accept terms:")

	fb := form.NewFormBuilder(pdf)
	fb.AddCheckbox("accept", 1, 60, 5, 5)

	if err := fb.Build(); err != nil {
		t.Fatalf("build: %v", err)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		t.Fatalf("output: %v", err)
	}

	pdfBytes := buf.Bytes()
	if !bytes.Contains(pdfBytes, []byte("/FT /Btn")) {
		t.Error("expected button field /FT /Btn in PDF output")
	}
	t.Logf("Form PDF with checkbox: %d bytes", buf.Len())
}

func TestDropdownCreation(t *testing.T) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetFont("Helvetica", "", 12)
	pdf.AddPage()
	pdf.Text(10, 10, "Country:")

	fb := form.NewFormBuilder(pdf)
	fb.AddDropdown("country", 1, 40, 5, 60, 8, []string{"USA", "Canada", "Mexico", "Brazil"}).
		SetValue("USA")

	if err := fb.Build(); err != nil {
		t.Fatalf("build: %v", err)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		t.Fatalf("output: %v", err)
	}

	pdfBytes := buf.Bytes()
	if !bytes.Contains(pdfBytes, []byte("/FT /Ch")) {
		t.Error("expected choice field /FT /Ch in PDF output")
	}
	if !bytes.Contains(pdfBytes, []byte("(USA)")) {
		t.Error("expected options in PDF output")
	}
	t.Logf("Form PDF with dropdown: %d bytes", buf.Len())
}

func TestMultipleFieldTypes(t *testing.T) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetFont("Helvetica", "", 12)
	pdf.AddPage()

	pdf.Text(10, 15, "Full Name:")
	pdf.Text(10, 30, "Email:")
	pdf.Text(10, 45, "Country:")
	pdf.Text(10, 60, "Accept Terms:")
	pdf.Text(10, 75, "Comments:")

	fb := form.NewFormBuilder(pdf)
	fb.AddTextField("fullname", 1, 50, 10, 80, 8).SetRequired(true)
	fb.AddTextField("email", 1, 50, 25, 80, 8).SetRequired(true)
	fb.AddDropdown("country", 1, 50, 40, 80, 8, []string{"USA", "Canada", "Mexico"})
	fb.AddCheckbox("terms", 1, 50, 55, 5)
	fb.AddTextField("comments", 1, 50, 70, 80, 20).SetMultiLine(true)

	if err := fb.Build(); err != nil {
		t.Fatalf("build: %v", err)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		t.Fatalf("output: %v", err)
	}

	doc, err := reader.ReadFrom(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("reading PDF: %v", err)
	}
	if doc.NumPages() != 1 {
		t.Errorf("expected 1 page, got %d", doc.NumPages())
	}
	t.Logf("Multi-field form PDF: %d bytes, %d pages", buf.Len(), doc.NumPages())
}

func TestEmptyFormBuild(t *testing.T) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	fb := form.NewFormBuilder(pdf)
	if err := fb.Build(); err != nil {
		t.Fatalf("empty form build should not error: %v", err)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		t.Fatalf("output: %v", err)
	}

	// No AcroForm should be present
	if bytes.Contains(buf.Bytes(), []byte("/AcroForm")) {
		t.Error("empty form should not contain /AcroForm")
	}
}

func TestReadOnlyField(t *testing.T) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetFont("Helvetica", "", 12)
	pdf.AddPage()

	fb := form.NewFormBuilder(pdf)
	fb.AddTextField("readonly_field", 1, 10, 10, 80, 8).
		SetValue("Cannot edit").
		SetReadOnly(true)

	if err := fb.Build(); err != nil {
		t.Fatalf("build: %v", err)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		t.Fatalf("output: %v", err)
	}

	if !bytes.Contains(buf.Bytes(), []byte("/Ff")) {
		t.Error("expected field flags /Ff in output")
	}
	t.Logf("Read-only field PDF: %d bytes", buf.Len())
}
