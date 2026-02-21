package sign

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"encoding/hex"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	verifySigTypeRe    = regexp.MustCompile(`/Type\s+/Sig\b`)
	verifyByteRangeRe  = regexp.MustCompile(`/ByteRange\s*\[([^\]]+)\]`)
	verifyContentsRe   = regexp.MustCompile(`/Contents\s*<([0-9a-fA-F]+)>`)
)

// Verify checks the digital signatures in a PDF document.
// It extracts signature dictionaries, recomputes digests from byte ranges,
// and returns information about each signature found.
//
// Note: Without certificates embedded in the PDF, cryptographic verification
// requires the signer's certificate. Use VerifyWithCertificate for full validation.
func Verify(input io.ReadSeeker) ([]SignatureInfo, error) {
	data, err := io.ReadAll(input)
	if err != nil {
		return nil, fmt.Errorf("sign: reading input: %w", err)
	}

	sigs := findSignatureDicts(data)
	if len(sigs) == 0 {
		return nil, nil
	}

	var results []SignatureInfo
	for _, sig := range sigs {
		info := SignatureInfo{
			Reason:   sig.reason,
			Location: sig.location,
			SignedAt:  sig.signedAt,
		}

		// Verify byte range integrity
		if sig.byteRange[1] > 0 && sig.byteRange[3] > 0 {
			digest, err := computeByteRangeDigest(data, sig.byteRange)
			if err != nil {
				info.Errors = append(info.Errors, fmt.Errorf("computing digest: %w", err))
			} else {
				info.digest = digest
				info.rawSignature = sig.contents
			}
		}

		results = append(results, info)
	}

	return results, nil
}

// VerifyWithCertificate verifies signatures using the provided certificate.
// This performs full cryptographic verification of each signature found.
func VerifyWithCertificate(input io.ReadSeeker, cert crypto.PublicKey) ([]SignatureInfo, error) {
	data, err := io.ReadAll(input)
	if err != nil {
		return nil, fmt.Errorf("sign: reading input: %w", err)
	}

	sigs := findSignatureDicts(data)
	if len(sigs) == 0 {
		return nil, nil
	}

	var results []SignatureInfo
	for _, sig := range sigs {
		info := SignatureInfo{
			Reason:   sig.reason,
			Location: sig.location,
			SignedAt:  sig.signedAt,
		}

		if sig.byteRange[1] == 0 || sig.byteRange[3] == 0 {
			info.Errors = append(info.Errors, fmt.Errorf("invalid byte range"))
			results = append(results, info)
			continue
		}

		digest, err := computeByteRangeDigest(data, sig.byteRange)
		if err != nil {
			info.Errors = append(info.Errors, fmt.Errorf("computing digest: %w", err))
			results = append(results, info)
			continue
		}

		// Verify the signature
		valid := verifyRawSignature(cert, digest, sig.contents)
		info.Valid = valid
		if !valid {
			info.Errors = append(info.Errors, fmt.Errorf("signature verification failed"))
		}

		results = append(results, info)
	}

	return results, nil
}

// rawSigInfo holds parsed signature dictionary data.
type rawSigInfo struct {
	byteRange [4]int
	contents  []byte // decoded hex contents
	reason    string
	location  string
	signedAt  time.Time
}

// findSignatureDicts searches the raw PDF bytes for /Type /Sig dictionaries.
func findSignatureDicts(data []byte) []rawSigInfo {
	var results []rawSigInfo

	// Find all /Type /Sig occurrences
	matches := verifySigTypeRe.FindAllIndex(data, -1)

	for _, m := range matches {
		sig := rawSigInfo{}

		// Find the surrounding dict
		dictStart := findSigDictStart(data, m[0])
		dictEnd := findSigDictEnd(data, m[0])
		if dictStart < 0 || dictEnd < 0 {
			continue
		}
		dict := data[dictStart : dictEnd+1]

		// Extract /ByteRange [a b c d]
		sig.byteRange = extractByteRange(dict)

		// Extract /Contents <hex>
		sig.contents = extractContents(data, dictStart, dictEnd)

		// Extract /Reason (text)
		sig.reason = extractPDFString(dict, "/Reason")

		// Extract /Location (text)
		sig.location = extractPDFString(dict, "/Location")

		// Extract /M (date)
		sig.signedAt = extractDate(dict)

		results = append(results, sig)
	}

	return results
}

// computeByteRangeDigest computes SHA-256 digest over the specified byte ranges.
func computeByteRangeDigest(data []byte, br [4]int) ([]byte, error) {
	if br[0]+br[1] > len(data) || br[2]+br[3] > len(data) {
		return nil, fmt.Errorf("byte range exceeds data length")
	}
	if br[0] < 0 || br[1] < 0 || br[2] < 0 || br[3] < 0 {
		return nil, fmt.Errorf("negative byte range value")
	}

	h := crypto.SHA256.New()
	h.Write(data[br[0] : br[0]+br[1]])
	h.Write(data[br[2] : br[2]+br[3]])
	return h.Sum(nil), nil
}

// verifyRawSignature verifies a raw signature against a digest using the given public key.
func verifyRawSignature(pub crypto.PublicKey, digest, signature []byte) bool {
	switch key := pub.(type) {
	case *ecdsa.PublicKey:
		return ecdsa.VerifyASN1(key, digest, signature)
	case *rsa.PublicKey:
		err := rsa.VerifyPKCS1v15(key, crypto.SHA256, digest, signature)
		return err == nil
	default:
		return false
	}
}

// findSigDictStart finds the start of the dictionary containing the position.
func findSigDictStart(data []byte, pos int) int {
	// Search backward from pos for "<<"
	// We need to handle nested dicts: the /Type /Sig could be inside
	// a nested dict, but we want the outermost sig dict
	for i := pos - 1; i > 0; i-- {
		if data[i] == '<' && i > 0 && data[i-1] == '<' {
			return i - 1
		}
		// If we hit endobj or another >>, we've gone too far
		if i >= 6 && string(data[i-6:i+1]) == "endobj\n" {
			break
		}
	}
	return -1
}

// findSigDictEnd finds the end of the outermost dictionary.
func findSigDictEnd(data []byte, pos int) int {
	start := findSigDictStart(data, pos)
	if start < 0 {
		return -1
	}
	depth := 0
	for i := start; i < len(data)-1; i++ {
		if data[i] == '<' && data[i+1] == '<' {
			depth++
			i++
			continue
		}
		if data[i] == '>' && data[i+1] == '>' {
			depth--
			if depth == 0 {
				return i + 1
			}
			i++
		}
	}
	return -1
}

// extractByteRange extracts the /ByteRange array from a signature dict.
func extractByteRange(dict []byte) [4]int {
	var br [4]int
	m := verifyByteRangeRe.FindSubmatch(dict)
	if m == nil {
		return br
	}

	parts := strings.Fields(string(m[1]))
	if len(parts) != 4 {
		return br
	}

	for i, p := range parts {
		// Remove any %OFFSET placeholders
		if strings.HasPrefix(p, "%") {
			continue
		}
		v, err := strconv.Atoi(p)
		if err != nil {
			continue
		}
		br[i] = v
	}
	return br
}

// extractContents extracts and hex-decodes the /Contents value from the signature.
func extractContents(data []byte, dictStart, dictEnd int) []byte {
	// Look for /Contents <hex...> in the broader context
	// The hex string may be very large, so we search from dictStart
	searchArea := data[dictStart:]
	m := verifyContentsRe.FindSubmatch(searchArea)
	if m == nil {
		return nil
	}

	// Remove trailing zero padding
	hexStr := strings.TrimRight(string(m[1]), "0")
	if len(hexStr)%2 != 0 {
		hexStr += "0"
	}

	decoded, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil
	}
	return decoded
}

// extractPDFString extracts a PDF string value for a given key.
func extractPDFString(dict []byte, key string) string {
	pattern := regexp.MustCompile(regexp.QuoteMeta(key) + `\s*\(([^)]*)\)`)
	m := pattern.FindSubmatch(dict)
	if m == nil {
		return ""
	}
	return string(m[1])
}

// extractDate extracts the /M date from a signature dict.
func extractDate(dict []byte) time.Time {
	mStr := extractPDFString(dict, "/M")
	if mStr == "" {
		return time.Time{}
	}

	// PDF date format: D:YYYYMMDDHHmmSS+HH'MM'
	mStr = strings.TrimPrefix(mStr, "D:")
	if len(mStr) < 14 {
		return time.Time{}
	}

	// Try parsing with timezone
	layouts := []string{
		"20060102150405-07'00'",
		"20060102150405+07'00'",
		"20060102150405Z",
		"20060102150405",
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, mStr); err == nil {
			return t
		}
	}
	return time.Time{}
}
