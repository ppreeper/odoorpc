package xmlrpc

import (
	"strings"
	"testing"
	"time"
)

func TestMarshalNil(t *testing.T) {
	b, err := marshal(nil)
	if err != nil {
		t.Fatalf("marshal(nil) unexpected error: %v", err)
	}
	if len(b) != 0 {
		t.Errorf("marshal(nil) = %q, want empty", b)
	}
}

func TestEncodeValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       any
		wantContain string
		wantErr     bool
	}{
		// integers
		{name: "int zero", input: int(0), wantContain: "<int>0</int>"},
		{name: "int positive", input: int(42), wantContain: "<int>42</int>"},
		{name: "int negative", input: int(-7), wantContain: "<int>-7</int>"},
		{name: "int8", input: int8(127), wantContain: "<int>127</int>"},
		{name: "int16", input: int16(1000), wantContain: "<int>1000</int>"},
		{name: "int32", input: int32(100000), wantContain: "<int>100000</int>"},
		{name: "int64", input: int64(1 << 40), wantContain: "<int>1099511627776</int>"},
		// unsigned integers encoded as <i4>
		{name: "uint", input: uint(5), wantContain: "<i4>5</i4>"},
		{name: "uint8", input: uint8(255), wantContain: "<i4>255</i4>"},
		// floats
		{name: "float64", input: float64(3.14), wantContain: "<double>3.14</double>"},
		{name: "float32", input: float32(1.5), wantContain: "<double>1.5</double>"},
		// booleans
		{name: "bool true", input: true, wantContain: "<boolean>1</boolean>"},
		{name: "bool false", input: false, wantContain: "<boolean>0</boolean>"},
		// strings
		{name: "string plain", input: "hello", wantContain: "<string>hello</string>"},
		{name: "string xml escape", input: "<>&\"", wantContain: "<string>&lt;&gt;&amp;&#34;</string>"},
		// Base64 type
		{name: "base64", input: Base64("data"), wantContain: "<base64>data</base64>"},
		// time
		{name: "time.Time", input: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC), wantContain: "<dateTime.iso8601>20240115T10:30:00</dateTime.iso8601>"},
		// pointer to int
		{name: "nil pointer", input: (*int)(nil), wantContain: "<value/>"},
		// slice
		{name: "int slice", input: []int{1, 2, 3}, wantContain: "<array><data>"},
		// map
		{name: "string map", input: map[string]any{"key": "val"}, wantContain: "<struct>"},
		// unsupported
		{name: "chan unsupported", input: make(chan int), wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			b, err := marshal(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("marshal(%v) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("marshal(%v) unexpected error: %v", tt.input, err)
			}
			got := string(b)
			if !strings.Contains(got, tt.wantContain) {
				t.Errorf("marshal(%v) = %q, want to contain %q", tt.input, got, tt.wantContain)
			}
		})
	}
}

func TestEncodeStruct(t *testing.T) {
	t.Parallel()

	type sample struct {
		Name  string `xmlrpc:"name"`
		Value int    `xmlrpc:"value"`
		Skip  string `xmlrpc:"-"`
	}

	b, err := marshal(sample{Name: "foo", Value: 99, Skip: "ignored"})
	if err != nil {
		t.Fatalf("marshal struct unexpected error: %v", err)
	}
	got := string(b)

	for _, want := range []string{"<struct>", "<name>name</name>", "<name>value</name>"} {
		if !strings.Contains(got, want) {
			t.Errorf("marshal struct missing %q in output: %s", want, got)
		}
	}
	if strings.Contains(got, "ignored") {
		t.Errorf("marshal struct should have omitted Skip field, but output contains 'ignored': %s", got)
	}
}

func TestEncodeStructOmitempty(t *testing.T) {
	t.Parallel()

	type sample struct {
		Present string `xmlrpc:"present"`
		Empty   string `xmlrpc:"empty,omitempty"`
	}

	b, err := marshal(sample{Present: "yes", Empty: ""})
	if err != nil {
		t.Fatalf("marshal struct omitempty unexpected error: %v", err)
	}
	got := string(b)
	if !strings.Contains(got, "present") {
		t.Errorf("expected 'present' in output, got: %s", got)
	}
	if strings.Contains(got, "empty") {
		t.Errorf("expected omitted 'empty' field, but found it in output: %s", got)
	}
}

func TestEncodeMapKeysSorted(t *testing.T) {
	t.Parallel()

	m := map[string]any{"z": 1, "a": 2, "m": 3}
	b, err := marshal(m)
	if err != nil {
		t.Fatalf("marshal map unexpected error: %v", err)
	}
	got := string(b)

	aPos := strings.Index(got, ">a<")
	mPos := strings.Index(got, ">m<")
	zPos := strings.Index(got, ">z<")
	if aPos < 0 || mPos < 0 || zPos < 0 {
		t.Fatalf("missing expected keys in output: %s", got)
	}
	if !(aPos < mPos && mPos < zPos) {
		t.Errorf("map keys not sorted alphabetically: a=%d m=%d z=%d in %s", aPos, mPos, zPos, got)
	}
}

func TestEncodeMapNonStringKeyError(t *testing.T) {
	t.Parallel()

	_, err := marshal(map[int]string{1: "a"})
	if err == nil {
		t.Error("expected error for map with non-string key, got nil")
	}
}

func TestEncodeSlice(t *testing.T) {
	t.Parallel()

	b, err := marshal([]string{"a", "b"})
	if err != nil {
		t.Fatalf("marshal slice unexpected error: %v", err)
	}
	got := string(b)
	for _, want := range []string{"<array>", "<data>", "<string>a</string>", "<string>b</string>"} {
		if !strings.Contains(got, want) {
			t.Errorf("marshal slice missing %q in output: %s", want, got)
		}
	}
}
