package reader_test

import (
	"bytes"
	"testing"

	gofpdf "github.com/lvillar/gofpdf"
	"github.com/lvillar/gofpdf/reader"
)

func generateProtectedPDF(t *testing.T, userPass, ownerPass string) []byte {
	t.Helper()
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetFont("Helvetica", "", 12)
	pdf.AddPage()
	pdf.Text(10, 20, "Protected content")
	pdf.SetProtection(0, userPass, ownerPass)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		t.Fatalf("generating protected PDF: %v", err)
	}
	return buf.Bytes()
}

func TestReadProtectedWithUserPassword(t *testing.T) {
	data := generateProtectedPDF(t, "user123", "owner456")

	doc, err := reader.ReadFromWithPassword(bytes.NewReader(data), "user123")
	if err != nil {
		t.Fatalf("reading protected PDF: %v", err)
	}

	if doc.NumPages() != 1 {
		t.Errorf("expected 1 page, got %d", doc.NumPages())
	}

	t.Logf("Decrypted PDF: %d pages, version=%s", doc.NumPages(), doc.Version)
}

func TestReadProtectedWithEmptyPassword(t *testing.T) {
	// SetProtection with empty user password should be openable with ""
	data := generateProtectedPDF(t, "", "owner456")

	doc, err := reader.ReadFrom(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("reading protected PDF with empty password: %v", err)
	}

	if doc.NumPages() != 1 {
		t.Errorf("expected 1 page, got %d", doc.NumPages())
	}

	t.Logf("Opened owner-protected PDF: %d pages", doc.NumPages())
}

func TestReadProtectedWrongPassword(t *testing.T) {
	data := generateProtectedPDF(t, "correct", "owner")

	_, err := reader.ReadFromWithPassword(bytes.NewReader(data), "wrong")
	if err == nil {
		t.Error("expected error with wrong password")
	}

	t.Logf("Wrong password error: %v", err)
}

func TestUnprotectedPDFStillWorks(t *testing.T) {
	// Verify that unencrypted PDFs still work fine
	data := generateTestPDF(t, "No encryption here")

	doc, err := reader.ReadFrom(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("reading unprotected PDF: %v", err)
	}

	if doc.NumPages() != 1 {
		t.Errorf("expected 1 page, got %d", doc.NumPages())
	}

	text, err := doc.Page(1)
	if err != nil {
		t.Fatalf("getting page 1: %v", err)
	}
	_ = text
}

func TestReadProtectedMetadata(t *testing.T) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle("Secret Doc", false)
	pdf.SetAuthor("Agent", false)
	pdf.SetFont("Helvetica", "", 12)
	pdf.AddPage()
	pdf.Text(10, 20, "Classified")
	pdf.SetProtection(gofpdf.CnProtectCopy, "pass", "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		t.Fatalf("generating PDF: %v", err)
	}

	doc, err := reader.ReadFromWithPassword(bytes.NewReader(buf.Bytes()), "pass")
	if err != nil {
		t.Fatalf("reading: %v", err)
	}

	meta := doc.Metadata()
	if meta["Title"] != "Secret Doc" {
		t.Errorf("Title = %q, want 'Secret Doc'", meta["Title"])
	}
	if meta["Author"] != "Agent" {
		t.Errorf("Author = %q, want 'Agent'", meta["Author"])
	}

	t.Logf("Decrypted metadata: %v", meta)
}
