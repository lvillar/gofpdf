package reader

import (
	"testing"
)

func TestParseInteger(t *testing.T) {
	p := newParser([]byte("42"))
	obj, err := p.ParseObject()
	if err != nil {
		t.Fatalf("parsing integer: %v", err)
	}
	if v, ok := obj.(Integer); !ok || int64(v) != 42 {
		t.Errorf("expected Integer(42), got %T(%v)", obj, obj)
	}
}

func TestParseNegativeInteger(t *testing.T) {
	p := newParser([]byte("-7"))
	obj, err := p.ParseObject()
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}
	if v, ok := obj.(Integer); !ok || int64(v) != -7 {
		t.Errorf("expected Integer(-7), got %T(%v)", obj, obj)
	}
}

func TestParseReal(t *testing.T) {
	p := newParser([]byte("3.14"))
	obj, err := p.ParseObject()
	if err != nil {
		t.Fatalf("parsing real: %v", err)
	}
	if v, ok := obj.(Real); !ok || float64(v) < 3.13 || float64(v) > 3.15 {
		t.Errorf("expected Real(3.14), got %T(%v)", obj, obj)
	}
}

func TestParseName(t *testing.T) {
	p := newParser([]byte("/Type"))
	obj, err := p.ParseObject()
	if err != nil {
		t.Fatalf("parsing name: %v", err)
	}
	if v, ok := obj.(Name); !ok || string(v) != "Type" {
		t.Errorf("expected Name(Type), got %T(%v)", obj, obj)
	}
}

func TestParseNameWithHexEscape(t *testing.T) {
	p := newParser([]byte("/A#20B"))
	obj, err := p.ParseObject()
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}
	if v, ok := obj.(Name); !ok || string(v) != "A B" {
		t.Errorf("expected Name(A B), got %T(%v)", obj, obj)
	}
}

func TestParseLiteralString(t *testing.T) {
	p := newParser([]byte("(Hello World)"))
	obj, err := p.ParseObject()
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}
	s, ok := obj.(String)
	if !ok {
		t.Fatalf("expected String, got %T", obj)
	}
	if string(s.Value) != "Hello World" {
		t.Errorf("expected 'Hello World', got %q", s.Value)
	}
	if s.IsHex {
		t.Error("expected non-hex string")
	}
}

func TestParseLiteralStringNested(t *testing.T) {
	p := newParser([]byte("(Hello (nested) World)"))
	obj, err := p.ParseObject()
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}
	s := obj.(String)
	if string(s.Value) != "Hello (nested) World" {
		t.Errorf("expected 'Hello (nested) World', got %q", s.Value)
	}
}

func TestParseLiteralStringEscapes(t *testing.T) {
	p := newParser([]byte(`(Line1\nLine2\r\t\\)`))
	obj, err := p.ParseObject()
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}
	s := obj.(String)
	if string(s.Value) != "Line1\nLine2\r\t\\" {
		t.Errorf("unexpected value: %q", s.Value)
	}
}

func TestParseHexString(t *testing.T) {
	p := newParser([]byte("<48656C6C6F>"))
	obj, err := p.ParseObject()
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}
	s, ok := obj.(String)
	if !ok {
		t.Fatalf("expected String, got %T", obj)
	}
	if string(s.Value) != "Hello" {
		t.Errorf("expected 'Hello', got %q", s.Value)
	}
	if !s.IsHex {
		t.Error("expected hex string")
	}
}

func TestParseBoolean(t *testing.T) {
	p := newParser([]byte("true"))
	obj, err := p.ParseObject()
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}
	if v, ok := obj.(Boolean); !ok || !bool(v) {
		t.Errorf("expected Boolean(true), got %T(%v)", obj, obj)
	}
}

func TestParseNull(t *testing.T) {
	p := newParser([]byte("null"))
	obj, err := p.ParseObject()
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}
	if _, ok := obj.(Null); !ok {
		t.Errorf("expected Null, got %T", obj)
	}
}

func TestParseArray(t *testing.T) {
	p := newParser([]byte("[1 2.5 /Name (text)]"))
	obj, err := p.ParseObject()
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}
	arr, ok := obj.(Array)
	if !ok {
		t.Fatalf("expected Array, got %T", obj)
	}
	if len(arr) != 4 {
		t.Fatalf("expected 4 elements, got %d", len(arr))
	}
	if _, ok := arr[0].(Integer); !ok {
		t.Errorf("element 0: expected Integer, got %T", arr[0])
	}
	if _, ok := arr[1].(Real); !ok {
		t.Errorf("element 1: expected Real, got %T", arr[1])
	}
	if _, ok := arr[2].(Name); !ok {
		t.Errorf("element 2: expected Name, got %T", arr[2])
	}
	if _, ok := arr[3].(String); !ok {
		t.Errorf("element 3: expected String, got %T", arr[3])
	}
}

func TestParseDict(t *testing.T) {
	p := newParser([]byte("<< /Type /Page /Count 3 >>"))
	obj, err := p.ParseObject()
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}
	d, ok := obj.(Dict)
	if !ok {
		t.Fatalf("expected Dict, got %T", obj)
	}
	if d.GetName("Type") != "Page" {
		t.Errorf("Type = %v, want Page", d["Type"])
	}
	if v, ok := d.GetInt("Count"); !ok || v != 3 {
		t.Errorf("Count = %v, want 3", d["Count"])
	}
}

func TestParseReference(t *testing.T) {
	p := newParser([]byte("10 0 R"))
	obj, err := p.ParseObject()
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}
	ref, ok := obj.(Reference)
	if !ok {
		t.Fatalf("expected Reference, got %T", obj)
	}
	if ref.Number != 10 || ref.Generation != 0 {
		t.Errorf("expected 10 0 R, got %d %d R", ref.Number, ref.Generation)
	}
}

func TestParseIndirectObject(t *testing.T) {
	p := newParser([]byte("5 0 obj\n<< /Type /Page >>\nendobj"))
	obj, err := p.ParseIndirectObject()
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}
	if obj.Number != 5 || obj.Generation != 0 {
		t.Errorf("expected 5 0 obj, got %d %d obj", obj.Number, obj.Generation)
	}
	d, ok := obj.Value.(Dict)
	if !ok {
		t.Fatalf("expected Dict value, got %T", obj.Value)
	}
	if d.GetName("Type") != "Page" {
		t.Errorf("Type = %v, want Page", d.GetName("Type"))
	}
}

func TestParseComment(t *testing.T) {
	p := newParser([]byte("% this is a comment\n42"))
	obj, err := p.ParseObject()
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}
	if v, ok := obj.(Integer); !ok || int64(v) != 42 {
		t.Errorf("expected Integer(42), got %T(%v)", obj, obj)
	}
}

func TestDictHelpers(t *testing.T) {
	d := Dict{
		"Name":   Name("Test"),
		"Count":  Integer(5),
		"Sub":    Dict{"Key": Name("Value")},
		"Items":  Array{Integer(1), Integer(2)},
	}

	if d.GetName("Name") != "Test" {
		t.Errorf("GetName: %v", d.GetName("Name"))
	}
	if d.GetName("Missing") != "" {
		t.Errorf("GetName missing: %v", d.GetName("Missing"))
	}
	if v, ok := d.GetInt("Count"); !ok || v != 5 {
		t.Errorf("GetInt: %v %v", v, ok)
	}
	if sub := d.GetDict("Sub"); sub == nil || sub.GetName("Key") != "Value" {
		t.Errorf("GetDict: %v", d.GetDict("Sub"))
	}
	if arr := d.GetArray("Items"); len(arr) != 2 {
		t.Errorf("GetArray: %v", arr)
	}
}
