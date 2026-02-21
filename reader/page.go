package reader

import (
	"fmt"
)

// Rectangle represents a PDF rectangle (typically [llx lly urx ury]).
type Rectangle struct {
	LLX, LLY, URX, URY float64
}

// Width returns the width of the rectangle.
func (r Rectangle) Width() float64 { return r.URX - r.LLX }

// Height returns the height of the rectangle.
func (r Rectangle) Height() float64 { return r.URY - r.LLY }

// Page represents a single page in a PDF document.
type Page struct {
	Number    int
	MediaBox  Rectangle
	CropBox   *Rectangle
	Resources Dict
	Contents  []Stream
	Rotate    int
	dict      Dict     // original page dictionary
	doc       *Document // back-reference for resolving objects
}

// ContentStream returns the decompressed content stream data for this page.
// If the page has multiple content streams, they are concatenated.
func (p *Page) ContentStream() ([]byte, error) {
	var result []byte
	for _, s := range p.Contents {
		decoded, err := decodeStream(s)
		if err != nil {
			return nil, fmt.Errorf("reader: decoding page %d content: %w", p.Number, err)
		}
		result = append(result, decoded...)
		result = append(result, '\n')
	}
	return result, nil
}

// parseRectangle parses a PDF rectangle array [llx lly urx ury].
func parseRectangle(obj Object) (Rectangle, error) {
	arr, ok := obj.(Array)
	if !ok || len(arr) != 4 {
		return Rectangle{}, fmt.Errorf("reader: rectangle must be a 4-element array")
	}

	vals := make([]float64, 4)
	for i, v := range arr {
		switch n := v.(type) {
		case Integer:
			vals[i] = float64(n)
		case Real:
			vals[i] = float64(n)
		default:
			return Rectangle{}, fmt.Errorf("reader: rectangle element %d is not numeric", i)
		}
	}
	return Rectangle{LLX: vals[0], LLY: vals[1], URX: vals[2], URY: vals[3]}, nil
}

// buildPageList traverses the page tree and returns a flat list of pages.
func (d *Document) buildPageList() error {
	catalog := d.trailer.GetDict("Root")
	if catalog == nil {
		// Root might be a reference
		rootRef, ok := d.trailer["Root"].(Reference)
		if !ok {
			return fmt.Errorf("reader: missing /Root in trailer")
		}
		rootObj, err := d.resolve(rootRef)
		if err != nil {
			return fmt.Errorf("reader: resolving root: %w", err)
		}
		var isCatalog bool
		catalog, isCatalog = rootObj.(Dict)
		if !isCatalog {
			return fmt.Errorf("reader: /Root is not a dictionary")
		}
	}

	pagesRef, ok := catalog["Pages"].(Reference)
	if !ok {
		return fmt.Errorf("reader: /Pages is not a reference")
	}

	pagesObj, err := d.resolve(pagesRef)
	if err != nil {
		return fmt.Errorf("reader: resolving /Pages: %w", err)
	}
	pagesDict, ok := pagesObj.(Dict)
	if !ok {
		return fmt.Errorf("reader: /Pages is not a dictionary")
	}

	d.pages = nil
	return d.traversePageTree(pagesDict, nil, 0)
}

// traversePageTree recursively traverses the page tree collecting leaf pages.
func (d *Document) traversePageTree(node Dict, inherited Dict, rotate int) error {
	nodeType := node.GetName("Type")

	// Inherit properties from parent
	merged := make(Dict)
	for k, v := range inherited {
		merged[k] = v
	}
	// Override with node's own properties
	for _, key := range []Name{"MediaBox", "CropBox", "Resources", "Rotate"} {
		if v, ok := node[key]; ok {
			merged[key] = v
		}
	}

	if nodeType == "Page" {
		page := &Page{
			Number: len(d.pages) + 1,
			dict:   node,
			doc:    d,
		}

		// MediaBox
		if mb, ok := merged["MediaBox"]; ok {
			resolved, err := d.resolveIfRef(mb)
			if err == nil {
				if rect, err := parseRectangle(resolved); err == nil {
					page.MediaBox = rect
				}
			}
		}

		// CropBox
		if cb, ok := merged["CropBox"]; ok {
			resolved, err := d.resolveIfRef(cb)
			if err == nil {
				if rect, err := parseRectangle(resolved); err == nil {
					page.CropBox = &rect
				}
			}
		}

		// Resources
		if res, ok := merged["Resources"]; ok {
			resolved, err := d.resolveIfRef(res)
			if err == nil {
				if resDict, ok := resolved.(Dict); ok {
					page.Resources = resDict
				}
			}
		}

		// Rotate
		if rotVal, ok := merged["Rotate"]; ok {
			resolved, err := d.resolveIfRef(rotVal)
			if err == nil {
				if intVal, ok := resolved.(Integer); ok {
					page.Rotate = int(intVal)
				}
			}
		}

		// Contents
		if contents, ok := node["Contents"]; ok {
			resolved, err := d.resolveIfRef(contents)
			if err != nil {
				return fmt.Errorf("reader: page %d contents: %w", page.Number, err)
			}

			switch c := resolved.(type) {
			case Stream:
				page.Contents = []Stream{c}
			case Array:
				for _, item := range c {
					streamObj, err := d.resolveIfRef(item)
					if err != nil {
						continue
					}
					if s, ok := streamObj.(Stream); ok {
						page.Contents = append(page.Contents, s)
					}
				}
			}
		}

		d.pages = append(d.pages, page)
		return nil
	}

	// Pages node - traverse children
	kids := node.GetArray("Kids")
	if kids == nil {
		if kidsRef, ok := node["Kids"].(Reference); ok {
			kidsObj, err := d.resolve(kidsRef)
			if err != nil {
				return fmt.Errorf("reader: resolving /Kids: %w", err)
			}
			kids, _ = kidsObj.(Array)
		}
	}

	for _, kid := range kids {
		kidObj, err := d.resolveIfRef(kid)
		if err != nil {
			return fmt.Errorf("reader: resolving page tree kid: %w", err)
		}
		kidDict, ok := kidObj.(Dict)
		if !ok {
			continue
		}
		if err := d.traversePageTree(kidDict, merged, rotate); err != nil {
			return err
		}
	}

	return nil
}
