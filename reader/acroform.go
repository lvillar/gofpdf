package reader

import (
	"fmt"
	"strings"
)

// FormField represents a form field parsed from a PDF's AcroForm dictionary.
type FormField struct {
	Name     string        // partial field name (/T)
	FullName string        // fully qualified dotted name
	Type     string        // field type: "Tx", "Btn", "Ch", "Sig"
	Value    string        // current value (/V)
	Default  string        // default value (/DV)
	Flags    int           // field flags (/Ff)
	Rect     Rectangle     // widget annotation rectangle
	Options  []string      // choice options (/Opt) for "Ch" fields
	Kids     []*FormField  // child fields in hierarchy
	ObjNum   int           // object number if from an indirect object
	dict     Dict          // original field dictionary
}

// IsReadOnly returns true if the field has the ReadOnly flag set (bit 1).
func (f *FormField) IsReadOnly() bool { return f.Flags&1 != 0 }

// IsRequired returns true if the field has the Required flag set (bit 2).
func (f *FormField) IsRequired() bool { return f.Flags&2 != 0 }

// Catalog returns the document's catalog dictionary (the /Root object).
func (d *Document) Catalog() (Dict, error) {
	rootObj, ok := d.trailer["Root"]
	if !ok {
		return nil, fmt.Errorf("reader: missing /Root in trailer")
	}
	resolved, err := d.resolveIfRef(rootObj)
	if err != nil {
		return nil, fmt.Errorf("reader: resolving /Root: %w", err)
	}
	catalog, ok := resolved.(Dict)
	if !ok {
		return nil, fmt.Errorf("reader: /Root is not a dictionary")
	}
	return catalog, nil
}

// FormFields returns all form fields found in the document's AcroForm.
// Returns an empty slice (not nil) if no AcroForm is present.
func (d *Document) FormFields() ([]*FormField, error) {
	catalog, err := d.Catalog()
	if err != nil {
		return []*FormField{}, nil
	}

	acroFormObj, ok := catalog["AcroForm"]
	if !ok {
		return []*FormField{}, nil
	}

	acroForm, err := d.resolveIfRef(acroFormObj)
	if err != nil {
		return nil, fmt.Errorf("reader: resolving AcroForm: %w", err)
	}
	acroDict, ok := acroForm.(Dict)
	if !ok {
		return []*FormField{}, nil
	}

	fieldsObj, ok := acroDict["Fields"]
	if !ok {
		return []*FormField{}, nil
	}
	fieldsResolved, err := d.resolveIfRef(fieldsObj)
	if err != nil {
		return nil, fmt.Errorf("reader: resolving AcroForm /Fields: %w", err)
	}
	fieldsArr, ok := fieldsResolved.(Array)
	if !ok {
		return []*FormField{}, nil
	}

	var fields []*FormField
	for _, fieldObj := range fieldsArr {
		field, err := d.parseFormField(fieldObj, "")
		if err != nil {
			continue // skip malformed fields
		}
		fields = append(fields, field)
	}

	if fields == nil {
		fields = []*FormField{}
	}
	return fields, nil
}

// FormField returns the form field with the given fully qualified name.
// Returns nil if the field is not found.
func (d *Document) FormField(name string) (*FormField, error) {
	fields, err := d.FormFields()
	if err != nil {
		return nil, err
	}
	return findField(fields, name), nil
}

// findField searches for a field by fully qualified name in a field tree.
func findField(fields []*FormField, name string) *FormField {
	for _, f := range fields {
		if f.FullName == name {
			return f
		}
		if found := findField(f.Kids, name); found != nil {
			return found
		}
	}
	return nil
}

// parseFormField parses a single form field dictionary.
func (d *Document) parseFormField(obj Object, parentName string) (*FormField, error) {
	// Track object number for indirect objects
	objNum := 0
	if ref, ok := obj.(Reference); ok {
		objNum = ref.Number
	}

	resolved, err := d.resolveIfRef(obj)
	if err != nil {
		return nil, err
	}
	dict, ok := resolved.(Dict)
	if !ok {
		return nil, fmt.Errorf("reader: form field is not a dictionary")
	}

	field := &FormField{
		dict:   dict,
		ObjNum: objNum,
	}

	// Partial name (/T)
	if t, ok := dict["T"]; ok {
		if s, ok := t.(String); ok {
			field.Name = decodePDFString(s.Value)
		}
	}

	// Build full name
	if parentName != "" && field.Name != "" {
		field.FullName = parentName + "." + field.Name
	} else if field.Name != "" {
		field.FullName = field.Name
	} else {
		field.FullName = parentName
	}

	// Field type (/FT) — may be inherited from parent
	if ft := dict.GetName("FT"); ft != "" {
		field.Type = string(ft)
	}

	// Value (/V)
	if v, ok := dict["V"]; ok {
		field.Value = objectToString(v)
	}

	// Default value (/DV)
	if dv, ok := dict["DV"]; ok {
		field.Default = objectToString(dv)
	}

	// Field flags (/Ff)
	if ff, ok := dict.GetInt("Ff"); ok {
		field.Flags = int(ff)
	}

	// Rectangle (/Rect)
	if rectObj, ok := dict["Rect"]; ok {
		rectResolved, err := d.resolveIfRef(rectObj)
		if err == nil {
			if rect, err := parseRectangle(rectResolved); err == nil {
				field.Rect = rect
			}
		}
	}

	// Options (/Opt) for choice fields
	if optObj, ok := dict["Opt"]; ok {
		optResolved, err := d.resolveIfRef(optObj)
		if err == nil {
			if optArr, ok := optResolved.(Array); ok {
				for _, item := range optArr {
					field.Options = append(field.Options, objectToString(item))
				}
			}
		}
	}

	// Kids (/Kids) — traverse field hierarchy
	if kidsObj, ok := dict["Kids"]; ok {
		kidsResolved, err := d.resolveIfRef(kidsObj)
		if err == nil {
			if kidsArr, ok := kidsResolved.(Array); ok {
				for _, kidObj := range kidsArr {
					kid, err := d.parseFormField(kidObj, field.FullName)
					if err != nil {
						continue
					}
					// Inherit field type from parent
					if kid.Type == "" {
						kid.Type = field.Type
					}
					field.Kids = append(field.Kids, kid)
				}
			}
		}
	}

	return field, nil
}

// objectToString converts a PDF object to its string representation for field values.
func objectToString(obj Object) string {
	switch v := obj.(type) {
	case String:
		return decodePDFString(v.Value)
	case Name:
		return string(v)
	case Integer:
		return fmt.Sprintf("%d", int64(v))
	case Real:
		return fmt.Sprintf("%g", float64(v))
	case Boolean:
		if v {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

// GetString returns the string value for a dictionary key, resolving references.
func (d Dict) GetString(key Name) string {
	v, ok := d[key]
	if !ok {
		return ""
	}
	if s, ok := v.(String); ok {
		return strings.TrimSpace(string(s.Value))
	}
	return ""
}
