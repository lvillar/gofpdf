package doctpl

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRenderMinimalDocument(t *testing.T) {
	doc := Document{
		Pages: []Page{{
			Elements: []Element{
				{Type: "paragraph", Text: "Hello, World!"},
			},
		}},
	}

	var buf bytes.Buffer
	if err := RenderDocument(&buf, &doc); err != nil {
		t.Fatalf("RenderDocument failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("expected non-empty PDF output")
	}

	// Check it starts with %PDF
	if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF")) {
		t.Fatal("output does not start with %PDF header")
	}
}

func TestRenderFromJSON(t *testing.T) {
	jsonTemplate := `{
		"title": "Test Document",
		"author": "Test Author",
		"pageSize": "A4",
		"pages": [{
			"elements": [
				{"type": "heading", "text": "Chapter 1", "level": 1},
				{"type": "paragraph", "text": "This is the first paragraph."},
				{"type": "hr"},
				{"type": "heading", "text": "Section 1.1", "level": 2},
				{"type": "paragraph", "text": "Another paragraph with more text.", "align": "C"}
			]
		}]
	}`

	var buf bytes.Buffer
	if err := Render(&buf, []byte(jsonTemplate)); err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("expected non-empty PDF output")
	}
}

func TestRenderWithTable(t *testing.T) {
	doc := Document{
		Pages: []Page{{
			Elements: []Element{
				{Type: "heading", Text: "Invoice", Level: 1},
				{
					Type: "table",
					Columns: []TableColumn{
						{Header: "Item", Width: 80},
						{Header: "Qty", Width: 30, Align: "C"},
						{Header: "Price", Width: 40, Align: "R"},
					},
					Rows: [][]string{
						{"Widget A", "10", "$5.00"},
						{"Widget B", "5", "$12.00"},
						{"Widget C", "3", "$8.50"},
					},
				},
			},
		}},
	}

	var buf bytes.Buffer
	if err := RenderDocument(&buf, &doc); err != nil {
		t.Fatalf("RenderDocument failed: %v", err)
	}

	if buf.Len() < 100 {
		t.Fatal("PDF output seems too small")
	}
}

func TestRenderWithList(t *testing.T) {
	doc := Document{
		Pages: []Page{{
			Elements: []Element{
				{Type: "heading", Text: "Shopping List", Level: 2},
				{
					Type:  "list",
					Items: []string{"Apples", "Bananas", "Oranges"},
				},
				{
					Type:    "list",
					Items:   []string{"First step", "Second step", "Third step"},
					Ordered: true,
				},
			},
		}},
	}

	var buf bytes.Buffer
	if err := RenderDocument(&buf, &doc); err != nil {
		t.Fatalf("RenderDocument failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("expected non-empty PDF output")
	}
}

func TestRenderWithSpacer(t *testing.T) {
	doc := Document{
		Pages: []Page{{
			Elements: []Element{
				{Type: "paragraph", Text: "Before spacer"},
				{Type: "spacer", SpacerHeight: 20},
				{Type: "paragraph", Text: "After spacer"},
			},
		}},
	}

	var buf bytes.Buffer
	if err := RenderDocument(&buf, &doc); err != nil {
		t.Fatalf("RenderDocument failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("expected non-empty PDF output")
	}
}

func TestRenderMultiplePages(t *testing.T) {
	doc := Document{
		Title: "Multi-page Document",
		Pages: []Page{
			{Elements: []Element{{Type: "heading", Text: "Page 1", Level: 1}}},
			{Elements: []Element{{Type: "heading", Text: "Page 2", Level: 1}}},
			{Elements: []Element{{Type: "heading", Text: "Page 3", Level: 1}}},
		},
	}

	var buf bytes.Buffer
	if err := RenderDocument(&buf, &doc); err != nil {
		t.Fatalf("RenderDocument failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("expected non-empty PDF output")
	}
}

func TestRenderWithHeaderFooter(t *testing.T) {
	doc := Document{
		Title: "Report",
		Header: &Header{
			Text:  "Company Report 2024",
			Align: "C",
		},
		Footer: &Footer{
			Text:  "Page {page}",
			Align: "C",
		},
		Pages: []Page{{
			Elements: []Element{
				{Type: "paragraph", Text: "Document body content."},
			},
		}},
	}

	var buf bytes.Buffer
	if err := RenderDocument(&buf, &doc); err != nil {
		t.Fatalf("RenderDocument failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("expected non-empty PDF output")
	}
}

func TestRenderWithCustomFont(t *testing.T) {
	doc := Document{
		Font: &Font{Family: "Courier", Size: 12},
		Pages: []Page{{
			Elements: []Element{
				{Type: "paragraph", Text: "In Courier font"},
				{
					Type: "paragraph",
					Text: "In Times Bold",
					Font: &Font{Family: "Times", Style: "B", Size: 14},
				},
			},
		}},
	}

	var buf bytes.Buffer
	if err := RenderDocument(&buf, &doc); err != nil {
		t.Fatalf("RenderDocument failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("expected non-empty PDF output")
	}
}

func TestRenderWithColors(t *testing.T) {
	doc := Document{
		Pages: []Page{{
			Elements: []Element{
				{
					Type:  "heading",
					Text:  "Red Heading",
					Level: 1,
					Color: &Color{R: 255, G: 0, B: 0},
				},
				{
					Type:  "paragraph",
					Text:  "Blue text paragraph",
					Color: &Color{R: 0, G: 0, B: 255},
				},
			},
		}},
	}

	var buf bytes.Buffer
	if err := RenderDocument(&buf, &doc); err != nil {
		t.Fatalf("RenderDocument failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("expected non-empty PDF output")
	}
}

func TestRenderWithMargins(t *testing.T) {
	doc := Document{
		Margin: &Margin{Top: 30, Right: 25, Bottom: 30, Left: 25},
		Pages: []Page{{
			Elements: []Element{
				{Type: "paragraph", Text: "Custom margins document"},
			},
		}},
	}

	var buf bytes.Buffer
	if err := RenderDocument(&buf, &doc); err != nil {
		t.Fatalf("RenderDocument failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("expected non-empty PDF output")
	}
}

func TestRenderInvalidJSON(t *testing.T) {
	var buf bytes.Buffer
	err := Render(&buf, []byte("not valid json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestRenderUnknownElementType(t *testing.T) {
	doc := Document{
		Pages: []Page{{
			Elements: []Element{
				{Type: "nonexistent"},
			},
		}},
	}

	var buf bytes.Buffer
	err := RenderDocument(&buf, &doc)
	if err == nil {
		t.Fatal("expected error for unknown element type")
	}
	if !strings.Contains(err.Error(), "unknown element type") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRenderEmptyPages(t *testing.T) {
	doc := Document{}

	var buf bytes.Buffer
	if err := RenderDocument(&buf, &doc); err != nil {
		t.Fatalf("RenderDocument failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("expected non-empty PDF output even with no pages")
	}
}

func TestRenderAllHeadingLevels(t *testing.T) {
	elements := make([]Element, 6)
	for i := range elements {
		elements[i] = Element{
			Type:  "heading",
			Text:  "Heading Level",
			Level: i + 1,
		}
	}

	doc := Document{
		Pages: []Page{{Elements: elements}},
	}

	var buf bytes.Buffer
	if err := RenderDocument(&buf, &doc); err != nil {
		t.Fatalf("RenderDocument failed: %v", err)
	}
}

func TestDocumentJSONRoundTrip(t *testing.T) {
	doc := Document{
		Title:    "Test",
		PageSize: "Letter",
		Font:     &Font{Family: "Helvetica", Size: 12},
		Pages: []Page{{
			Elements: []Element{
				{Type: "heading", Text: "Title", Level: 1},
				{Type: "paragraph", Text: "Body"},
			},
		}},
	}

	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var doc2 Document
	if err := json.Unmarshal(data, &doc2); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if doc2.Title != doc.Title {
		t.Fatalf("Title mismatch: %q vs %q", doc2.Title, doc.Title)
	}
	if doc2.PageSize != doc.PageSize {
		t.Fatalf("PageSize mismatch: %q vs %q", doc2.PageSize, doc.PageSize)
	}
	if len(doc2.Pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(doc2.Pages))
	}
	if len(doc2.Pages[0].Elements) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(doc2.Pages[0].Elements))
	}
}
