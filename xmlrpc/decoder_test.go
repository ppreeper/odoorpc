package xmlrpc

import (
	"strings"
	"testing"
	"time"
)

// wrapValue wraps an XML value fragment in the minimal envelope needed by unmarshal.
func wrapValue(inner string) []byte {
	return []byte(`<?xml version="1.0"?><methodResponse><params><param><value>` +
		inner + `</value></param></params></methodResponse>`)
}

func TestUnmarshalNonPointerError(t *testing.T) {
	t.Parallel()
	var v int
	err := unmarshal(wrapValue("<int>1</int>"), v)
	if err == nil {
		t.Fatal("expected error for non-pointer value, got nil")
	}
	if !strings.Contains(err.Error(), "non-pointer") {
		t.Errorf("expected 'non-pointer' in error, got %q", err)
	}
}

func TestUnmarshalInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		xml  string
		want int64
	}{
		{"int tag", "<int>42</int>", 42},
		{"i4 tag", "<i4>-7</i4>", -7},
		{"i8 tag", "<i8>1099511627776</i8>", 1099511627776},
		{"zero", "<int>0</int>", 0},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var v int64
			if err := unmarshal(wrapValue(tt.xml), &v); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if v != tt.want {
				t.Errorf("got %d, want %d", v, tt.want)
			}
		})
	}
}

func TestUnmarshalIntToInterface(t *testing.T) {
	t.Parallel()
	var v interface{}
	if err := unmarshal(wrapValue("<int>99</int>"), &v); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, ok := v.(int64)
	if !ok {
		t.Fatalf("expected int64, got %T", v)
	}
	if got != 99 {
		t.Errorf("got %d, want 99", got)
	}
}

func TestUnmarshalDouble(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		xml  string
		want float64
	}{
		{"positive", "<double>3.14</double>", 3.14},
		{"negative", "<double>-1.5</double>", -1.5},
		{"zero", "<double>0</double>", 0},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var v float64
			if err := unmarshal(wrapValue(tt.xml), &v); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if v != tt.want {
				t.Errorf("got %f, want %f", v, tt.want)
			}
		})
	}
}

func TestUnmarshalDoubleToInterface(t *testing.T) {
	t.Parallel()
	var v interface{}
	if err := unmarshal(wrapValue("<double>2.718</double>"), &v); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, ok := v.(float64)
	if !ok {
		t.Fatalf("expected float64, got %T", v)
	}
	if got != 2.718 {
		t.Errorf("got %f, want 2.718", got)
	}
}

func TestUnmarshalBoolean(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		xml  string
		want bool
	}{
		{"true 1", "<boolean>1</boolean>", true},
		{"false 0", "<boolean>0</boolean>", false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var v bool
			if err := unmarshal(wrapValue(tt.xml), &v); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if v != tt.want {
				t.Errorf("got %v, want %v", v, tt.want)
			}
		})
	}
}

func TestUnmarshalBoolToInterface(t *testing.T) {
	t.Parallel()
	var v interface{}
	if err := unmarshal(wrapValue("<boolean>1</boolean>"), &v); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, ok := v.(bool)
	if !ok {
		t.Fatalf("expected bool, got %T", v)
	}
	if !got {
		t.Error("expected true")
	}
}

func TestUnmarshalString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		xml  string
		want string
	}{
		{"string tag", "<string>hello</string>", "hello"},
		{"base64 tag", "<base64>aGVsbG8=</base64>", "aGVsbG8="},
		{"empty string", "<string></string>", ""},
		{"untyped chardata", "plain text", "plain text"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var v string
			if err := unmarshal(wrapValue(tt.xml), &v); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if v != tt.want {
				t.Errorf("got %q, want %q", v, tt.want)
			}
		})
	}
}

func TestUnmarshalStringToInterface(t *testing.T) {
	t.Parallel()
	var v interface{}
	if err := unmarshal(wrapValue("<string>world</string>"), &v); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, ok := v.(string)
	if !ok {
		t.Fatalf("expected string, got %T", v)
	}
	if got != "world" {
		t.Errorf("got %q, want %q", got, "world")
	}
}

func TestUnmarshalDateTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		want  time.Time
	}{
		{
			"iso8601 basic",
			"20240115T10:30:00",
			time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			"iso8601 hyphen",
			"2024-01-15T10:30:00",
			time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			"iso8601Z",
			"20240115T10:30:00Z",
			time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			"iso8601HyphenZ",
			"2024-01-15T10:30:00Z",
			time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			xml := "<dateTime.iso8601>" + tt.value + "</dateTime.iso8601>"
			var v time.Time
			if err := unmarshal(wrapValue(xml), &v); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !v.Equal(tt.want) {
				t.Errorf("got %v, want %v", v, tt.want)
			}
		})
	}
}

func TestUnmarshalDateTimeToInterface(t *testing.T) {
	t.Parallel()
	var v interface{}
	if err := unmarshal(wrapValue("<dateTime.iso8601>20240115T10:30:00</dateTime.iso8601>"), &v); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, ok := v.(time.Time)
	if !ok {
		t.Fatalf("expected time.Time, got %T", v)
	}
}

func TestUnmarshalArray(t *testing.T) {
	t.Parallel()

	xml := `<array><data>
		<value><string>a</string></value>
		<value><string>b</string></value>
		<value><string>c</string></value>
	</data></array>`

	var v []string
	if err := unmarshal(wrapValue(xml), &v); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(v) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(v))
	}
	want := []string{"a", "b", "c"}
	for i, w := range want {
		if v[i] != w {
			t.Errorf("[%d] got %q, want %q", i, v[i], w)
		}
	}
}

func TestUnmarshalArrayToInterface(t *testing.T) {
	t.Parallel()

	xml := `<array><data>
		<value><int>1</int></value>
		<value><int>2</int></value>
	</data></array>`

	var v interface{}
	if err := unmarshal(wrapValue(xml), &v); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	arr, ok := v.([]interface{})
	if !ok {
		t.Fatalf("expected []interface{}, got %T", v)
	}
	if len(arr) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(arr))
	}
}

func TestUnmarshalStruct(t *testing.T) {
	t.Parallel()

	type sample struct {
		Name  string `xmlrpc:"name"`
		Value int    `xmlrpc:"value"`
		Skip  string `xmlrpc:"-"`
	}

	xml := `<struct>
		<member><name>name</name><value><string>foo</string></value></member>
		<member><name>value</name><value><int>99</int></value></member>
	</struct>`

	var v sample
	if err := unmarshal(wrapValue(xml), &v); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Name != "foo" {
		t.Errorf("Name: got %q, want %q", v.Name, "foo")
	}
	if v.Value != 99 {
		t.Errorf("Value: got %d, want 99", v.Value)
	}
}

func TestUnmarshalStructToMap(t *testing.T) {
	t.Parallel()

	xml := `<struct>
		<member><name>key1</name><value><string>val1</string></value></member>
		<member><name>key2</name><value><int>42</int></value></member>
	</struct>`

	var v map[string]interface{}
	if err := unmarshal(wrapValue(xml), &v); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v["key1"] != "val1" {
		t.Errorf("key1: got %v, want %q", v["key1"], "val1")
	}
	if v["key2"] != int64(42) {
		t.Errorf("key2: got %v (%T), want int64(42)", v["key2"], v["key2"])
	}
}

func TestUnmarshalStructToInterface(t *testing.T) {
	t.Parallel()

	xml := `<struct>
		<member><name>x</name><value><int>1</int></value></member>
	</struct>`

	var v interface{}
	if err := unmarshal(wrapValue(xml), &v); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := v.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map[string]interface{}, got %T", v)
	}
	if m["x"] != int64(1) {
		t.Errorf("x: got %v, want int64(1)", m["x"])
	}
}

func TestUnmarshalTypeMismatchError(t *testing.T) {
	t.Parallel()

	// Try to decode a boolean into a string — should produce TypeMismatchError.
	var v string
	err := unmarshal(wrapValue("<boolean>1</boolean>"), &v)
	if err == nil {
		t.Fatal("expected TypeMismatchError, got nil")
	}
	var tme TypeMismatchError
	// TypeMismatchError is a string type implementing error; check via Error() content.
	if _, ok := err.(TypeMismatchError); !ok {
		// Also accept wrapped. Direct cast suffices since decodeValue returns it directly.
		t.Errorf("expected TypeMismatchError, got %T: %v", err, err)
	}
	_ = tme
}

func TestTypeMismatchErrorMessage(t *testing.T) {
	t.Parallel()
	e := TypeMismatchError("test mismatch")
	if e.Error() != "test mismatch" {
		t.Errorf("got %q, want %q", e.Error(), "test mismatch")
	}
}

func TestCheckType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		xmlData string
		dest    interface{}
		wantErr bool
	}{
		{"int to int", "<int>5</int>", new(int), false},
		{"double to float64", "<double>1.1</double>", new(float64), false},
		{"bool to bool", "<boolean>1</boolean>", new(bool), false},
		{"string to string", "<string>hi</string>", new(string), false},
		// Type mismatch cases
		{"int to string mismatch", "<int>5</int>", new(string), true},
		{"bool to int mismatch", "<boolean>1</boolean>", new(int), true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := unmarshal(wrapValue(tt.xmlData), tt.dest)
			if tt.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// ─── Pre-allocated slice decoding (#20) ───────────────────────────────────────

func TestUnmarshalIntoPreAllocatedStringSlice(t *testing.T) {
	t.Parallel()
	xml := `<array><data>` +
		`<value><string>hello</string></value>` +
		`<value><string>world</string></value>` +
		`</data></array>`

	// Pre-allocate with two slots so both elements hit the index < slice.Len() path.
	v := make([]string, 2)
	if err := unmarshal(wrapValue(xml), &v); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(v) != 2 {
		t.Fatalf("expected len 2, got %d", len(v))
	}
	if v[0] != "hello" {
		t.Errorf("v[0]: got %q, want %q", v[0], "hello")
	}
	if v[1] != "world" {
		t.Errorf("v[1]: got %q, want %q", v[1], "world")
	}
}

func TestUnmarshalIntoPreAllocatedInt64Slice(t *testing.T) {
	t.Parallel()
	xml := `<array><data>` +
		`<value><int>10</int></value>` +
		`<value><int>20</int></value>` +
		`</data></array>`

	v := make([]int64, 2)
	if err := unmarshal(wrapValue(xml), &v); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v[0] != 10 {
		t.Errorf("v[0]: got %d, want 10", v[0])
	}
	if v[1] != 20 {
		t.Errorf("v[1]: got %d, want 20", v[1])
	}
}

func TestUnmarshalPreAllocatedSliceGrowsBeyondCapacity(t *testing.T) {
	t.Parallel()
	// Pre-allocate 1 slot but provide 3 values; the remaining 2 must be appended.
	xml := `<array><data>` +
		`<value><string>a</string></value>` +
		`<value><string>b</string></value>` +
		`<value><string>c</string></value>` +
		`</data></array>`

	v := make([]string, 1)
	if err := unmarshal(wrapValue(xml), &v); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(v) != 3 {
		t.Fatalf("expected len 3, got %d: %v", len(v), v)
	}
	want := []string{"a", "b", "c"}
	for i, w := range want {
		if v[i] != w {
			t.Errorf("v[%d]: got %q, want %q", i, v[i], w)
		}
	}
}
