package reader_test

import (
	"bytes"
	"testing"

	gofpdf "github.com/lvillar/gofpdf"
	"github.com/lvillar/gofpdf/form"
	"github.com/lvillar/gofpdf/reader"
)

func generateFormPDF(t *testing.T) []byte {
	t.Helper()
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetFont("Helvetica", "", 12)
	pdf.AddPage()
	pdf.Text(10, 10, "Form test")

	fb := form.NewFormBuilder(pdf)
	fb.AddTextField("name", 1, 40, 5, 80, 10)
	fb.AddTextField("email", 1, 40, 20, 80, 10).SetRequired(true)
	fb.AddCheckbox("agree", 1, 40, 35, 5)
	fb.AddDropdown("country", 1, 40, 50, 80, 8, []string{"USA", "Canada", "Mexico"})

	if err := fb.Build(); err != nil {
		t.Fatalf("build form: %v", err)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		t.Fatalf("output: %v", err)
	}
	return buf.Bytes()
}

func TestFormFieldsParsing(t *testing.T) {
	data := generateFormPDF(t)

	doc, err := reader.ReadFrom(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("reading PDF: %v", err)
	}

	fields, err := doc.FormFields()
	if err != nil {
		t.Fatalf("FormFields: %v", err)
	}

	if len(fields) < 4 {
		t.Fatalf("expected at least 4 fields, got %d", len(fields))
	}

	// Check that we can find each field by name
	names := make(map[string]bool)
	for _, f := range fields {
		names[f.FullName] = true
		t.Logf("Field: name=%q type=%q flags=%d", f.FullName, f.Type, f.Flags)
	}

	for _, expected := range []string{"name", "email", "agree", "country"} {
		if !names[expected] {
			t.Errorf("expected field %q not found", expected)
		}
	}
}

func TestFormFieldTypes(t *testing.T) {
	data := generateFormPDF(t)

	doc, err := reader.ReadFrom(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("reading PDF: %v", err)
	}

	fields, err := doc.FormFields()
	if err != nil {
		t.Fatalf("FormFields: %v", err)
	}

	typeMap := make(map[string]string)
	for _, f := range fields {
		typeMap[f.FullName] = f.Type
	}

	if typeMap["name"] != "Tx" {
		t.Errorf("field 'name' type = %q, want 'Tx'", typeMap["name"])
	}
	if typeMap["agree"] != "Btn" {
		t.Errorf("field 'agree' type = %q, want 'Btn'", typeMap["agree"])
	}
	if typeMap["country"] != "Ch" {
		t.Errorf("field 'country' type = %q, want 'Ch'", typeMap["country"])
	}
}

func TestFormFieldFlags(t *testing.T) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetFont("Helvetica", "", 12)
	pdf.AddPage()

	fb := form.NewFormBuilder(pdf)
	fb.AddTextField("readonly_field", 1, 10, 10, 80, 8).SetReadOnly(true)
	fb.AddTextField("required_field", 1, 10, 25, 80, 8).SetRequired(true)

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

	fields, err := doc.FormFields()
	if err != nil {
		t.Fatalf("FormFields: %v", err)
	}

	for _, f := range fields {
		switch f.FullName {
		case "readonly_field":
			if !f.IsReadOnly() {
				t.Error("readonly_field should be read-only")
			}
		case "required_field":
			if !f.IsRequired() {
				t.Error("required_field should be required")
			}
		}
	}
}

func TestFormFieldsEmpty(t *testing.T) {
	data := generateTestPDF(t, "No form here")

	doc, err := reader.ReadFrom(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("reading PDF: %v", err)
	}

	fields, err := doc.FormFields()
	if err != nil {
		t.Fatalf("FormFields: %v", err)
	}

	if len(fields) != 0 {
		t.Errorf("expected 0 fields for non-form PDF, got %d", len(fields))
	}
}

func TestFormFieldByName(t *testing.T) {
	data := generateFormPDF(t)

	doc, err := reader.ReadFrom(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("reading PDF: %v", err)
	}

	field, err := doc.FormField("email")
	if err != nil {
		t.Fatalf("FormField: %v", err)
	}
	if field == nil {
		t.Fatal("expected to find 'email' field")
	}
	if field.Type != "Tx" {
		t.Errorf("email field type = %q, want 'Tx'", field.Type)
	}

	// Non-existent field
	missing, err := doc.FormField("nonexistent")
	if err != nil {
		t.Fatalf("FormField: %v", err)
	}
	if missing != nil {
		t.Error("expected nil for non-existent field")
	}
}

func TestCatalog(t *testing.T) {
	data := generateFormPDF(t)

	doc, err := reader.ReadFrom(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("reading PDF: %v", err)
	}

	catalog, err := doc.Catalog()
	if err != nil {
		t.Fatalf("Catalog: %v", err)
	}

	if catalog == nil {
		t.Fatal("expected non-nil catalog")
	}

	// Should have /Type /Catalog
	if typ := catalog.GetName("Type"); typ != "Catalog" {
		t.Errorf("catalog type = %q, want 'Catalog'", typ)
	}
}
