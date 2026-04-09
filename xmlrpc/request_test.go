package xmlrpc

import (
	"context"
	"io"
	"strings"
	"testing"
)

// ─── EncodeMethodCall ─────────────────────────────────────────────────────────

func TestEncodeMethodCallNoArgs(t *testing.T) {
	t.Parallel()
	b, err := EncodeMethodCall("system.listMethods")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := string(b)
	for _, want := range []string{
		`<?xml version="1.0"`,
		`<methodCall>`,
		`<methodName>system.listMethods</methodName>`,
		`</methodCall>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in output: %s", want, got)
		}
	}
	// No <params> element when there are no args.
	if strings.Contains(got, "<params>") {
		t.Errorf("expected no <params> for no-arg call, got: %s", got)
	}
}

func TestEncodeMethodCallSingleArg(t *testing.T) {
	t.Parallel()
	b, err := EncodeMethodCall("myMethod", "argValue")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := string(b)
	for _, want := range []string{
		`<methodName>myMethod</methodName>`,
		`<params>`,
		`<param>`,
		`<string>argValue</string>`,
		`</params>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in output: %s", want, got)
		}
	}
}

func TestEncodeMethodCallMultipleArgs(t *testing.T) {
	t.Parallel()
	b, err := EncodeMethodCall("execute", "db", int64(1), "secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := string(b)
	for _, want := range []string{
		`<string>db</string>`,
		`<int>1</int>`,
		`<string>secret</string>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in output: %s", want, got)
		}
	}
}

func TestEncodeMethodCallUnsupportedArgReturnsError(t *testing.T) {
	t.Parallel()
	_, err := EncodeMethodCall("method", make(chan int))
	if err == nil {
		t.Error("expected error for unsupported arg type, got nil")
	}
}

// ─── NewRequest ───────────────────────────────────────────────────────────────

func TestNewRequestHeaders(t *testing.T) {
	t.Parallel()
	req, err := NewRequest(context.Background(), "http://localhost:8069/xmlrpc/2/common", "login", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Method != "POST" {
		t.Errorf("Method: got %q, want POST", req.Method)
	}
	ct := req.Header.Get("Content-Type")
	if ct != "text/xml" {
		t.Errorf("Content-Type: got %q, want text/xml", ct)
	}
}

func TestNewRequestURLAndMethod(t *testing.T) {
	t.Parallel()
	const rawURL = "http://localhost:8069/xmlrpc/2/common"
	req, err := NewRequest(context.Background(), rawURL, "authenticate", []interface{}{"db", "admin", "pass", map[string]interface{}{}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.URL.String() != rawURL {
		t.Errorf("URL: got %q, want %q", req.URL.String(), rawURL)
	}
}

func TestNewRequestBodyContainsMethodName(t *testing.T) {
	t.Parallel()
	req, err := NewRequest(context.Background(), "http://localhost:8069/xmlrpc/2/common", "authenticate", "singleArg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	if !strings.Contains(string(body), "authenticate") {
		t.Errorf("body does not contain method name 'authenticate': %s", body)
	}
}

func TestNewRequestNilArgsProducesNoParams(t *testing.T) {
	t.Parallel()
	req, err := NewRequest(context.Background(), "http://localhost:8069/xmlrpc/2/common", "listMethods", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	if strings.Contains(string(body), "<params>") {
		t.Errorf("expected no <params> for nil args, got: %s", body)
	}
}

func TestNewRequestContextCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled
	// NewRequest itself should still succeed — context is only checked on Do.
	_, err := NewRequest(ctx, "http://localhost:8069/xmlrpc/2/common", "method", nil)
	if err != nil {
		t.Fatalf("unexpected error constructing request with cancelled context: %v", err)
	}
}

func TestNewRequestInvalidURL(t *testing.T) {
	t.Parallel()
	_, err := NewRequest(context.Background(), "://bad url", "method", nil)
	if err == nil {
		t.Error("expected error for invalid URL, got nil")
	}
}

func TestEncodeMethodCallEscapesMethodName(t *testing.T) {
	t.Parallel()
	// A method name containing XML special characters must be escaped, not
	// emitted literally, so the resulting document remains well-formed.
	b, err := EncodeMethodCall("a<b>&c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	body := string(b)
	if strings.Contains(body, "<b>") {
		t.Errorf("method name was not escaped: raw '<b>' found in output:\n%s", body)
	}
	if !strings.Contains(body, "&lt;b&gt;") {
		t.Errorf("expected escaped method name in output, got:\n%s", body)
	}
}
