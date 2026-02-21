package form

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"

	"github.com/lvillar/gofpdf/reader"
)

// Fill reads a PDF from input, fills form fields with the provided values,
// and writes the result to output. Field names are matched case-sensitively.
//
// After modifying field values, the xref table is rebuilt to ensure validity.
func Fill(input io.ReadSeeker, output io.Writer, values map[string]string) error {
	if len(values) == 0 {
		if _, err := input.Seek(0, io.SeekStart); err != nil {
			return fmt.Errorf("form: seeking input: %w", err)
		}
		_, err := io.Copy(output, input)
		return err
	}

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
		return fmt.Errorf("form: no form fields found in PDF")
	}

	allFields := flattenFields(fields)

	fieldMap := make(map[string]*reader.FormField)
	for _, f := range allFields {
		fieldMap[f.FullName] = f
	}
	for name := range values {
		if _, ok := fieldMap[name]; !ok {
			return fmt.Errorf("form: field %q not found in PDF", name)
		}
	}

	// Work on a copy
	modified := make([]byte, len(data))
	copy(modified, data)

	for name, value := range values {
		field := fieldMap[name]
		modified = setFieldValue(modified, field, value)
	}

	// Rebuild xref table to account for any byte offset changes
	modified = rebuildXref(modified)

	_, err = io.Copy(output, bytes.NewReader(modified))
	return err
}

// FillFile reads a PDF from inputPath, fills form fields, and writes to outputPath.
func FillFile(inputPath, outputPath string, values map[string]string) error {
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

	return Fill(input, out, values)
}

// flattenFields returns a flat list of all form fields, recursing into kids.
func flattenFields(fields []*reader.FormField) []*reader.FormField {
	var result []*reader.FormField
	for _, f := range fields {
		result = append(result, f)
		if len(f.Kids) > 0 {
			result = append(result, flattenFields(f.Kids)...)
		}
	}
	return result
}

// setFieldValue modifies the raw PDF bytes to set a field's /V entry.
// Updates all occurrences (field appears in /Annots and /AcroForm /Fields).
// May change total data length; caller must rebuild xref after.
func setFieldValue(data []byte, field *reader.FormField, value string) []byte {
	escapedName := escapePDFString(field.Name)
	pattern := []byte(fmt.Sprintf("/T (%s)", escapedName))
	altPattern := []byte(fmt.Sprintf("/T(%s)", escapedName))

	// Process up to 10 occurrences (field dict duplicated in Annots + Fields)
	for pass := 0; pass < 10; pass++ {
		idx := bytes.Index(data, pattern)
		if idx < 0 {
			idx = bytes.Index(data, altPattern)
		}
		if idx < 0 {
			break
		}

		dictStart := findDictStart(data, idx)
		dictEnd := findDictEnd(data, idx)
		if dictStart < 0 || dictEnd < 0 {
			break
		}

		fieldDict := make([]byte, dictEnd+2-dictStart)
		copy(fieldDict, data[dictStart:dictEnd+2])

		var newValueStr string
		switch field.Type {
		case "Btn":
			if value == "true" || value == "Yes" || value == "on" {
				newValueStr = "/V /Yes /AS /Yes"
			} else {
				newValueStr = "/V /Off /AS /Off"
			}
		default:
			newValueStr = fmt.Sprintf("/V (%s)", escapePDFString(value))
		}

		var newDict []byte
		replaced := false

		if loc := regexp.MustCompile(`/V\s*\([^)]*\)`).FindIndex(fieldDict); loc != nil {
			newDict = make([]byte, 0, len(fieldDict))
			newDict = append(newDict, fieldDict[:loc[0]]...)
			newDict = append(newDict, []byte(newValueStr)...)
			newDict = append(newDict, fieldDict[loc[1]:]...)
			replaced = true
		}
		if !replaced {
			if loc := regexp.MustCompile(`/V\s+/[A-Za-z]+(\s+/AS\s+/[A-Za-z]+)?`).FindIndex(fieldDict); loc != nil {
				newDict = make([]byte, 0, len(fieldDict))
				newDict = append(newDict, fieldDict[:loc[0]]...)
				newDict = append(newDict, []byte(newValueStr)...)
				newDict = append(newDict, fieldDict[loc[1]:]...)
				replaced = true
			}
		}
		if !replaced {
			newDict = make([]byte, 0, len(fieldDict)+len(newValueStr)+1)
			newDict = append(newDict, fieldDict[:len(fieldDict)-2]...)
			newDict = append(newDict, ' ')
			newDict = append(newDict, []byte(newValueStr)...)
			newDict = append(newDict, '>', '>')
		}

		if bytes.Equal(fieldDict, newDict) {
			break
		}

		result := make([]byte, 0, len(data)-len(fieldDict)+len(newDict))
		result = append(result, data[:dictStart]...)
		result = append(result, newDict...)
		result = append(result, data[dictEnd+2:]...)
		data = result
	}

	return data
}

// rebuildXref scans the PDF body for object definitions and rebuilds the
// xref table with correct offsets. This handles byte-level modifications
// that shift object positions.
func rebuildXref(data []byte) []byte {
	// Find all "N G obj" markers
	objPattern := regexp.MustCompile(`(?m)^(\d+)\s+(\d+)\s+obj\b`)
	matches := objPattern.FindAllSubmatchIndex(data, -1)
	if len(matches) == 0 {
		return data
	}

	type objInfo struct {
		num, gen, offset int
	}
	var objects []objInfo
	maxObj := 0

	for _, m := range matches {
		num, _ := strconv.Atoi(string(data[m[2]:m[3]]))
		gen, _ := strconv.Atoi(string(data[m[4]:m[5]]))
		objects = append(objects, objInfo{num: num, gen: gen, offset: m[0]})
		if num > maxObj {
			maxObj = num
		}
	}

	// Find old xref table position
	xrefIdx := bytes.LastIndex(data, []byte("\nxref\n"))
	if xrefIdx < 0 {
		xrefIdx = bytes.Index(data, []byte("xref\n"))
		if xrefIdx > 0 {
			xrefIdx-- // include preceding newline for body slice
		}
	}
	if xrefIdx < 0 {
		return data
	}

	// Extract trailer dict
	trailerIdx := bytes.Index(data[xrefIdx:], []byte("trailer"))
	if trailerIdx < 0 {
		return data
	}
	trailerAbsIdx := xrefIdx + trailerIdx

	startxrefIdx := bytes.Index(data[trailerAbsIdx:], []byte("startxref"))
	if startxrefIdx < 0 {
		return data
	}
	trailerDict := bytes.TrimSpace(data[trailerAbsIdx+7 : trailerAbsIdx+startxrefIdx])

	// Body = everything up to and including the newline before "xref"
	body := data[:xrefIdx+1]

	// Build new xref
	var xref bytes.Buffer
	xref.WriteString("xref\n")
	xref.WriteString(fmt.Sprintf("0 %d\n", maxObj+1))
	xref.WriteString("0000000000 65535 f \n")

	offsets := make(map[int]objInfo)
	for _, obj := range objects {
		offsets[obj.num] = obj
	}

	for i := 1; i <= maxObj; i++ {
		if obj, ok := offsets[i]; ok {
			xref.WriteString(fmt.Sprintf("%010d %05d n \n", obj.offset, obj.gen))
		} else {
			xref.WriteString("0000000000 00000 f \n")
		}
	}

	// Calculate new xref offset
	newXrefOffset := len(body)

	// Assemble result
	var result bytes.Buffer
	result.Write(body)
	result.Write(xref.Bytes())
	result.WriteString("trailer\n")
	result.Write(trailerDict)
	result.WriteString(fmt.Sprintf("\nstartxref\n%d\n%%%%EOF\n", newXrefOffset))

	return result.Bytes()
}

// findDictStart searches backward from pos for the nearest "<<".
func findDictStart(data []byte, pos int) int {
	depth := 0
	for i := pos - 1; i > 0; i-- {
		if i+1 < len(data) && data[i] == '>' && data[i+1] == '>' {
			depth++
		}
		if data[i] == '<' && i > 0 && data[i-1] == '<' {
			if depth == 0 {
				return i - 1
			}
			depth--
		}
	}
	return -1
}

// findDictEnd searches forward from pos for the matching ">>".
func findDictEnd(data []byte, pos int) int {
	depth := 0
	for i := pos; i < len(data)-1; i++ {
		if data[i] == '<' && data[i+1] == '<' {
			depth++
			i++
			continue
		}
		if data[i] == '>' && data[i+1] == '>' {
			if depth <= 1 {
				return i
			}
			depth--
			i++
		}
	}
	return -1
}
