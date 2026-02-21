package mcp

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lvillar/gofpdf/reader"
)

// RegisterDefaultResources adds all built-in PDF resources to the server.
// Resources use URI templates with the pdf:// scheme.
func RegisterDefaultResources(s *Server) {
	s.AddResource(Resource{
		URI:         "pdf://text",
		Name:        "PDF Text Content",
		Description: "Extract all text content from a PDF file. Pass the file path as a query parameter: pdf://text?path=/path/to/file.pdf",
		MIMEType:    "text/plain",
		Handler:     handleTextResource,
	})

	s.AddResource(Resource{
		URI:         "pdf://metadata",
		Name:        "PDF Metadata",
		Description: "Get metadata from a PDF file (title, author, subject, etc.). Pass the file path as a query parameter: pdf://metadata?path=/path/to/file.pdf",
		MIMEType:    "application/json",
		Handler:     handleMetadataResource,
	})

	s.AddResource(Resource{
		URI:         "pdf://pages",
		Name:        "PDF Page Info",
		Description: "Get page information from a PDF (count, dimensions). Pass the file path as a query parameter: pdf://pages?path=/path/to/file.pdf",
		MIMEType:    "application/json",
		Handler:     handlePagesResource,
	})

	s.AddResource(Resource{
		URI:         "pdf://form-fields",
		Name:        "PDF Form Fields",
		Description: "List all form fields in a PDF. Pass the file path as a query parameter: pdf://form-fields?path=/path/to/file.pdf",
		MIMEType:    "application/json",
		Handler:     handleFormFieldsResource,
	})
}

func extractPathFromURI(uri string) string {
	// Parse path from URI like pdf://text?path=/foo/bar.pdf
	if idx := strings.Index(uri, "path="); idx >= 0 {
		return uri[idx+5:]
	}
	return ""
}

func handleTextResource(uri string) ([]ResourceContent, error) {
	path := extractPathFromURI(uri)
	if path == "" {
		return nil, fmt.Errorf("missing 'path' parameter in URI")
	}

	doc, err := reader.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening PDF: %w", err)
	}

	var result strings.Builder
	for pageNum, page := range doc.Pages() {
		text, err := page.ExtractText()
		if err != nil {
			fmt.Fprintf(&result, "--- Page %d (error: %v) ---\n", pageNum, err)
			continue
		}
		fmt.Fprintf(&result, "--- Page %d ---\n%s\n\n", pageNum, text)
	}

	return []ResourceContent{{
		URI:      uri,
		MIMEType: "text/plain",
		Text:     result.String(),
	}}, nil
}

func handleMetadataResource(uri string) ([]ResourceContent, error) {
	path := extractPathFromURI(uri)
	if path == "" {
		return nil, fmt.Errorf("missing 'path' parameter in URI")
	}

	doc, err := reader.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening PDF: %w", err)
	}

	info := map[string]interface{}{
		"version":  doc.Version,
		"numPages": doc.NumPages(),
		"metadata": doc.Metadata(),
	}

	jsonBytes, _ := json.MarshalIndent(info, "", "  ")
	return []ResourceContent{{
		URI:      uri,
		MIMEType: "application/json",
		Text:     string(jsonBytes),
	}}, nil
}

func handlePagesResource(uri string) ([]ResourceContent, error) {
	path := extractPathFromURI(uri)
	if path == "" {
		return nil, fmt.Errorf("missing 'path' parameter in URI")
	}

	doc, err := reader.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening PDF: %w", err)
	}

	pages := make([]map[string]interface{}, 0)
	for pageNum, page := range doc.Pages() {
		mb := page.MediaBox
		pages = append(pages, map[string]interface{}{
			"page":   pageNum,
			"width":  mb.Width(),
			"height": mb.Height(),
			"rotate": page.Rotate,
		})
	}

	info := map[string]interface{}{
		"numPages": doc.NumPages(),
		"pages":    pages,
	}

	jsonBytes, _ := json.MarshalIndent(info, "", "  ")
	return []ResourceContent{{
		URI:      uri,
		MIMEType: "application/json",
		Text:     string(jsonBytes),
	}}, nil
}

func handleFormFieldsResource(uri string) ([]ResourceContent, error) {
	path := extractPathFromURI(uri)
	if path == "" {
		return nil, fmt.Errorf("missing 'path' parameter in URI")
	}

	doc, err := reader.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening PDF: %w", err)
	}

	fields, err := doc.FormFields()
	if err != nil {
		return nil, fmt.Errorf("reading form fields: %w", err)
	}

	fieldInfos := make([]map[string]interface{}, 0)
	for _, f := range flattenFormFields(fields) {
		fi := map[string]interface{}{
			"name":     f.FullName,
			"type":     f.Type,
			"value":    f.Value,
			"readOnly": f.IsReadOnly(),
			"required": f.IsRequired(),
		}
		if len(f.Options) > 0 {
			fi["options"] = f.Options
		}
		fieldInfos = append(fieldInfos, fi)
	}

	info := map[string]interface{}{
		"fieldCount": len(fieldInfos),
		"fields":     fieldInfos,
	}

	jsonBytes, _ := json.MarshalIndent(info, "", "  ")
	return []ResourceContent{{
		URI:      uri,
		MIMEType: "application/json",
		Text:     string(jsonBytes),
	}}, nil
}
