package form

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"

	"github.com/lvillar/gofpdf/reader"
)

// Flatten reads a PDF with form fields and converts all field widgets into
// static page content, removing the interactive AcroForm structure.
// The resulting PDF will look the same but fields will no longer be editable.
//
// Uses space-replacement to preserve byte offsets and xref table validity.
func Flatten(input io.ReadSeeker, output io.Writer) error {
	data, err := io.ReadAll(input)
	if err != nil {
		return fmt.Errorf("form: reading input: %w", err)
	}

	doc, err := reader.ReadFrom(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("form: parsing PDF: %w", err)
	}

	fields, err := doc.FormFields()
	if err != nil {
		return fmt.Errorf("form: reading form fields: %w", err)
	}

	if len(fields) == 0 {
		_, err = io.Copy(output, bytes.NewReader(data))
		return err
	}

	modified := make([]byte, len(data))
	copy(modified, data)

	// Blank out /AcroForm from the catalog (replace with spaces)
	blankAcroForm(modified)

	// Blank out interactive markers from each field
	allFields := flattenFields(fields)
	for _, field := range allFields {
		blankFieldMarkers(modified, field)
	}

	_, err = io.Copy(output, bytes.NewReader(modified))
	return err
}

// FlattenFile reads a PDF from inputPath, flattens form fields, and writes to outputPath.
func FlattenFile(inputPath, outputPath string) error {
	input, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("form: opening %s: %w", inputPath, err)
	}
	defer input.Close()

	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("form: creating %s: %w", outputPath, err)
	}
	defer out.Close()

	return Flatten(input, out)
}

// blankAcroForm replaces the /AcroForm entry in the catalog with spaces
// (same byte length) to preserve xref offsets.
func blankAcroForm(data []byte) {
	acroStart := bytes.Index(data, []byte("/AcroForm"))
	if acroStart < 0 {
		return
	}

	// Find what follows /AcroForm
	pos := acroStart + len("/AcroForm")
	for pos < len(data) && (data[pos] == ' ' || data[pos] == '\n' || data[pos] == '\r') {
		pos++
	}

	var acroEnd int
	if pos < len(data)-1 && data[pos] == '<' && data[pos+1] == '<' {
		// Inline dict: find matching >>
		depth := 1
		i := pos + 2
		for i < len(data)-1 && depth > 0 {
			if data[i] == '<' && data[i+1] == '<' {
				depth++
				i += 2
				continue
			}
			if data[i] == '>' && data[i+1] == '>' {
				depth--
				if depth == 0 {
					acroEnd = i + 2
					break
				}
				i += 2
				continue
			}
			i++
		}
	} else {
		// Reference: /AcroForm N N R
		re := regexp.MustCompile(`\d+\s+\d+\s+R`)
		remaining := data[pos:]
		if loc := re.FindIndex(remaining); loc != nil && loc[0] == 0 {
			acroEnd = pos + loc[1]
		}
	}

	if acroEnd <= acroStart {
		return
	}

	// Replace with spaces (preserves byte offsets)
	for i := acroStart; i < acroEnd; i++ {
		data[i] = ' '
	}
}

// blankFieldMarkers replaces interactive field markers (/FT, /Subtype /Widget)
// with spaces to de-interactivize the field while preserving byte offsets.
func blankFieldMarkers(data []byte, field *reader.FormField) {
	escapedName := escapePDFString(field.Name)
	patterns := []string{
		fmt.Sprintf("/T (%s)", escapedName),
		fmt.Sprintf("/T(%s)", escapedName),
	}

	for _, pattern := range patterns {
		idx := bytes.Index(data, []byte(pattern))
		if idx < 0 {
			continue
		}

		dictStart := findDictStart(data, idx)
		dictEnd := findDictEnd(data, idx)
		if dictStart < 0 || dictEnd < 0 {
			continue
		}

		fieldDict := data[dictStart : dictEnd+2]

		// Blank out /FT /Type entries
		blankPattern(fieldDict, `/FT\s+/[A-Za-z]+`)

		// Blank out /Subtype /Widget
		blankPattern(fieldDict, `/Subtype\s+/Widget`)

		// Blank out /DA (default appearance)
		blankPattern(fieldDict, `/DA\s*\([^)]*\)`)

		// Blank out /NeedAppearances
		blankPattern(fieldDict, `/NeedAppearances\s+(true|false)`)

		break
	}
}

// blankPattern replaces all matches of a regex pattern in data with spaces.
func blankPattern(data []byte, pattern string) {
	re := regexp.MustCompile(pattern)
	for _, loc := range re.FindAllIndex(data, -1) {
		for i := loc[0]; i < loc[1]; i++ {
			data[i] = ' '
		}
	}
}
