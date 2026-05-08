package doctpl

import (
	"encoding/json"
	"fmt"
)

// ValidationError describes a single semantic problem found in a template.
//
// Path locates the offending element using a dot/bracket notation that mirrors
// the JSON structure (e.g. "pages[0].elements[2].rows[1]"). Field is the JSON
// key whose value is invalid (e.g. "level"). Message is a human-readable
// explanation suitable for surfacing back to an LLM for self-correction.
type ValidationError struct {
	Path    string `json:"path"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Validate parses jsonBytes as a doctpl template and returns the list of
// semantic problems found. It returns a non-nil error only when jsonBytes is
// not valid JSON or otherwise cannot be unmarshaled into a Document; semantic
// problems are reported via the returned slice.
//
// An empty slice means the template is structurally valid and ready to be
// rendered with Render. Validate does not execute any rendering or I/O.
func Validate(jsonBytes []byte) ([]ValidationError, error) {
	var doc Document
	if err := json.Unmarshal(jsonBytes, &doc); err != nil {
		return nil, fmt.Errorf("doctpl: parsing template: %w", err)
	}
	return ValidateDocument(&doc), nil
}

// ValidateDocument runs the same checks as Validate against an already-parsed
// Document. It is useful when constructing templates programmatically.
func ValidateDocument(doc *Document) []ValidationError {
	var errs []ValidationError

	if doc.PageSize != "" && !isKnownPageSize(doc.PageSize) {
		errs = append(errs, ValidationError{
			Path:    "",
			Field:   "pageSize",
			Message: fmt.Sprintf("unsupported page size %q (expected one of: A4, Letter, Legal)", doc.PageSize),
		})
	}
	if doc.Unit != "" && !isKnownUnit(doc.Unit) {
		errs = append(errs, ValidationError{
			Path:    "",
			Field:   "unit",
			Message: fmt.Sprintf("unsupported unit %q (expected one of: mm, cm, in, pt)", doc.Unit),
		})
	}

	if len(doc.Pages) == 0 {
		errs = append(errs, ValidationError{
			Path:    "",
			Field:   "pages",
			Message: "at least one page is required",
		})
		return errs
	}

	for i, page := range doc.Pages {
		pagePath := fmt.Sprintf("pages[%d]", i)
		for j, elem := range page.Elements {
			elemPath := fmt.Sprintf("%s.elements[%d]", pagePath, j)
			errs = append(errs, validateElement(elem, elemPath)...)
		}
	}
	return errs
}

func validateElement(elem Element, path string) []ValidationError {
	var errs []ValidationError

	if !isKnownElementType(elem.Type) {
		return []ValidationError{{
			Path:    path,
			Field:   "type",
			Message: fmt.Sprintf("unknown element type %q (expected one of: heading, paragraph, text, table, image, line, rect, spacer, hr, list)", elem.Type),
		}}
	}

	if elem.Align != "" && !isKnownAlign(elem.Align) {
		errs = append(errs, ValidationError{
			Path:    path,
			Field:   "align",
			Message: fmt.Sprintf("invalid align %q (expected one of: L, C, R)", elem.Align),
		})
	}

	switch elem.Type {
	case "heading":
		if elem.Text == "" {
			errs = append(errs, ValidationError{
				Path: path, Field: "text",
				Message: "heading requires a non-empty 'text'",
			})
		}
		if elem.Level != 0 && (elem.Level < 1 || elem.Level > 6) {
			errs = append(errs, ValidationError{
				Path: path, Field: "level",
				Message: fmt.Sprintf("heading level must be 1-6, got %d", elem.Level),
			})
		}

	case "table":
		if len(elem.Columns) == 0 {
			errs = append(errs, ValidationError{
				Path: path, Field: "columns",
				Message: "table requires at least one column",
			})
		}
		for ri, row := range elem.Rows {
			if len(row) != len(elem.Columns) {
				errs = append(errs, ValidationError{
					Path:    fmt.Sprintf("%s.rows[%d]", path, ri),
					Field:   "row",
					Message: fmt.Sprintf("row has %d cells but table has %d columns", len(row), len(elem.Columns)),
				})
			}
		}

	case "image":
		if elem.Src == "" {
			errs = append(errs, ValidationError{
				Path: path, Field: "src",
				Message: "image requires 'src' (file path or URL)",
			})
		}

	case "list":
		if len(elem.Items) == 0 {
			errs = append(errs, ValidationError{
				Path: path, Field: "items",
				Message: "list requires at least one item",
			})
		}
	}

	return errs
}

func isKnownPageSize(s string) bool {
	switch s {
	case "A4", "Letter", "Legal":
		return true
	}
	return false
}

func isKnownUnit(s string) bool {
	switch s {
	case "mm", "cm", "in", "pt":
		return true
	}
	return false
}

func isKnownElementType(s string) bool {
	switch s {
	case "heading", "paragraph", "text", "table", "image", "line", "rect", "spacer", "hr", "list":
		return true
	}
	return false
}

func isKnownAlign(s string) bool {
	switch s {
	case "L", "C", "R", "l", "c", "r":
		return true
	}
	return false
}
