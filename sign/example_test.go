package sign_test

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"time"

	gofpdf "github.com/lvillar/gofpdf"
	"github.com/lvillar/gofpdf/sign"
)

// ExampleSign demonstrates signing a PDF with a self-signed ECDSA certificate
// and then verifying the resulting signature with the same certificate's
// public key. In production you would load a certificate issued by a CA
// trusted by the relying parties.
func ExampleSign() {
	// 1. Build a small PDF to sign.
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetFont("Helvetica", "", 14)
	pdf.AddPage()
	pdf.Cell(0, 10, "Contract draft")

	var pdfBuf bytes.Buffer
	if err := pdf.Output(&pdfBuf); err != nil {
		fmt.Println(err)
		return
	}

	// 2. Generate a self-signed ECDSA certificate (test only).
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		fmt.Println(err)
		return
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Demo Signer"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		fmt.Println(err)
		return
	}
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		fmt.Println(err)
		return
	}

	// 3. Sign the PDF.
	var signed bytes.Buffer
	err = sign.Sign(bytes.NewReader(pdfBuf.Bytes()), &signed, sign.Options{
		Certificate: cert,
		PrivateKey:  key,
		Reason:      "Approved",
		Location:    "HQ",
	})
	if err != nil {
		fmt.Println(err)
		return
	}

	// 4. Verify against the certificate's public key.
	infos, err := sign.VerifyWithCertificate(bytes.NewReader(signed.Bytes()), cert.PublicKey)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("Signatures found: %d\n", len(infos))
	if len(infos) > 0 {
		fmt.Printf("Reason: %s\n", infos[0].Reason)
		fmt.Printf("Location: %s\n", infos[0].Location)
	}

	// Output:
	// Signatures found: 1
	// Reason: Approved
	// Location: HQ
}
