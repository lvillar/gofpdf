package doctpl

import (
	"strings"
	"testing"
)

func TestValidate_MinimalValidDocument(t *testing.T) {
	tpl := []byte(`{
		"title": "Test",
		"pages": [{"elements": []}]
	}`)
	errs, err := Validate(tpl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("expected no validation errors, got %d: %+v", len(errs), errs)
	}
}

func TestValidate_MalformedJSON(t *testing.T) {
	tpl := []byte(`{not valid json`)
	_, err := Validate(tpl)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestValidate_UnknownElementType(t *testing.T) {
	tpl := []byte(`{
		"pages": [{"elements": [
			{"type": "frobnicate", "text": "x"}
		]}]
	}`)
	errs, _ := Validate(tpl)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %+v", len(errs), errs)
	}
	if errs[0].Path != "pages[0].elements[0]" {
		t.Errorf("expected path pages[0].elements[0], got %q", errs[0].Path)
	}
	if errs[0].Field != "type" {
		t.Errorf("expected field 'type', got %q", errs[0].Field)
	}
	if !strings.Contains(errs[0].Message, "frobnicate") {
		t.Errorf("expected message to mention 'frobnicate', got %q", errs[0].Message)
	}
}

func TestValidate_HeadingMissingText(t *testing.T) {
	tpl := []byte(`{
		"pages": [{"elements": [
			{"type": "heading", "level": 1}
		]}]
	}`)
	errs, _ := Validate(tpl)
	if len(errs) == 0 {
		t.Fatal("expected validation error for heading without text")
	}
	found := false
	for _, e := range errs {
		if e.Field == "text" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error on field 'text', got %+v", errs)
	}
}

func TestValidate_HeadingInvalidLevel(t *testing.T) {
	tpl := []byte(`{
		"pages": [{"elements": [
			{"type": "heading", "text": "x", "level": 7}
		]}]
	}`)
	errs, _ := Validate(tpl)
	if len(errs) == 0 {
		t.Fatal("expected error for heading level > 6")
	}
}

func TestValidate_TableInconsistentColumns(t *testing.T) {
	tpl := []byte(`{
		"pages": [{"elements": [
			{
				"type": "table",
				"columns": [{"header": "A"}, {"header": "B"}],
				"rows": [["a", "b"], ["c"]]
			}
		]}]
	}`)
	errs, _ := Validate(tpl)
	if len(errs) == 0 {
		t.Fatal("expected error for table row with wrong cell count")
	}
	found := false
	for _, e := range errs {
		if strings.Contains(e.Path, "rows[1]") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected error referencing rows[1], got %+v", errs)
	}
}

func TestValidate_ImageMissingSrc(t *testing.T) {
	tpl := []byte(`{
		"pages": [{"elements": [
			{"type": "image", "width": 10, "height": 10}
		]}]
	}`)
	errs, _ := Validate(tpl)
	if len(errs) == 0 {
		t.Fatal("expected error for image without src")
	}
}

func TestValidate_InvalidPageSize(t *testing.T) {
	tpl := []byte(`{
		"pageSize": "Tabloid",
		"pages": [{"elements": []}]
	}`)
	errs, _ := Validate(tpl)
	if len(errs) == 0 {
		t.Fatal("expected error for invalid pageSize")
	}
	if errs[0].Field != "pageSize" {
		t.Errorf("expected field pageSize, got %q", errs[0].Field)
	}
}

func TestValidate_NoPages(t *testing.T) {
	tpl := []byte(`{"title": "x"}`)
	errs, _ := Validate(tpl)
	if len(errs) == 0 {
		t.Fatal("expected error when no pages provided")
	}
	if errs[0].Field != "pages" {
		t.Errorf("expected field pages, got %q", errs[0].Field)
	}
}

func TestValidate_AlignEnum(t *testing.T) {
	tpl := []byte(`{
		"pages": [{"elements": [
			{"type": "paragraph", "text": "x", "align": "Z"}
		]}]
	}`)
	errs, _ := Validate(tpl)
	if len(errs) == 0 {
		t.Fatal("expected error for invalid align value")
	}
}

func TestValidate_MultipleErrors(t *testing.T) {
	tpl := []byte(`{
		"pageSize": "Tabloid",
		"pages": [{"elements": [
			{"type": "heading", "level": 99},
			{"type": "image"}
		]}]
	}`)
	errs, _ := Validate(tpl)
	if len(errs) < 3 {
		t.Fatalf("expected at least 3 errors (pageSize + heading text + heading level + image src), got %d: %+v", len(errs), errs)
	}
}
