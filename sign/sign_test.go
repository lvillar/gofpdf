package sign_test

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"

	gofpdf "github.com/lvillar/gofpdf"
	"github.com/lvillar/gofpdf/sign"
)

// generateTestCert creates a self-signed certificate and key for testing.
func generateTestCert(t *testing.T) (*x509.Certificate, *ecdsa.PrivateKey) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generating key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "Test Signer",
			Organization: []string{"Test Org"},
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("creating certificate: %v", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatalf("parsing certificate: %v", err)
	}

	return cert, key
}

// generateTestPDF creates a simple PDF for signing tests.
func generateTestPDF(t *testing.T) []byte {
	t.Helper()
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetFont("Helvetica", "", 14)
	pdf.AddPage()
	pdf.Text(20, 30, "Document to be signed")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		t.Fatalf("generating test PDF: %v", err)
	}
	return buf.Bytes()
}

func TestSignBasic(t *testing.T) {
	cert, key := generateTestCert(t)
	pdfData := generateTestPDF(t)

	opts := sign.Options{
		Certificate: cert,
		PrivateKey:  key,
		Reason:      "Test signing",
		Location:    "Test Lab",
	}

	var output bytes.Buffer
	err := sign.Sign(bytes.NewReader(pdfData), &output, opts)
	if err != nil {
		t.Fatalf("signing: %v", err)
	}

	if output.Len() <= len(pdfData) {
		t.Errorf("signed PDF should be larger: input=%d, output=%d", len(pdfData), output.Len())
	}

	// Verify the output contains signature markers
	if !bytes.Contains(output.Bytes(), []byte("/Type /Sig")) {
		t.Error("expected /Type /Sig in signed PDF")
	}
	if !bytes.Contains(output.Bytes(), []byte("/Filter /Adobe.PPKLite")) {
		t.Error("expected /Filter /Adobe.PPKLite in signed PDF")
	}

	t.Logf("Signed PDF: input=%d bytes, output=%d bytes", len(pdfData), output.Len())
}

func TestSignRequiresCertificate(t *testing.T) {
	pdfData := generateTestPDF(t)

	var output bytes.Buffer
	err := sign.Sign(bytes.NewReader(pdfData), &output, sign.Options{})
	if err == nil {
		t.Error("expected error when certificate is missing")
	}
}

func TestSignRequiresPrivateKey(t *testing.T) {
	cert, _ := generateTestCert(t)
	pdfData := generateTestPDF(t)

	var output bytes.Buffer
	err := sign.Sign(bytes.NewReader(pdfData), &output, sign.Options{
		Certificate: cert,
	})
	if err == nil {
		t.Error("expected error when private key is missing")
	}
}

func TestVerifyReturnsNoError(t *testing.T) {
	pdfData := generateTestPDF(t)

	_, err := sign.Verify(bytes.NewReader(pdfData))
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
}
