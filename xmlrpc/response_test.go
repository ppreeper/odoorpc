package xmlrpc

import (
	"strings"
	"testing"
)

// validMethodResponse wraps a value in a full XML-RPC methodResponse envelope.
func validMethodResponse(inner string) Response {
	return Response(`<?xml version="1.0"?><methodResponse><params><param><value>` +
		inner + `</value></param></params></methodResponse>`)
}

// faultResponse builds a fault XML-RPC response.
func faultResponse(code int, msg string) Response {
	return Response(`<?xml version="1.0"?><methodResponse><fault><value><struct>` +
		`<member><name>faultCode</name><value><int>` + itoa(code) + `</int></value></member>` +
		`<member><name>faultString</name><value><string>` + msg + `</string></value></member>` +
		`</struct></value></fault></methodResponse>`)
}

// itoa is a tiny helper to avoid importing strconv/fmt just for int formatting.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	buf := make([]byte, 0, 10)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}

// ─── FaultError ──────────────────────────────────────────────────────────────

func TestFaultErrorMessage(t *testing.T) {
	t.Parallel()
	e := FaultError{Code: 404, String: "not found"}
	want := "Fault(404): not found"
	if e.Error() != want {
		t.Errorf("got %q, want %q", e.Error(), want)
	}
}

func TestFaultErrorZeroCode(t *testing.T) {
	t.Parallel()
	e := FaultError{Code: 0, String: "something"}
	if !strings.HasPrefix(e.Error(), "Fault(0)") {
		t.Errorf("unexpected format: %q", e.Error())
	}
}

// ─── Response.Err ────────────────────────────────────────────────────────────

func TestResponseErrNoFault(t *testing.T) {
	t.Parallel()
	r := validMethodResponse("<string>ok</string>")
	if err := r.Err(); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestResponseErrWithFault(t *testing.T) {
	t.Parallel()
	r := faultResponse(42, "something went wrong")
	err := r.Err()
	if err == nil {
		t.Fatal("expected fault error, got nil")
	}
	fe, ok := err.(FaultError)
	if !ok {
		t.Fatalf("expected FaultError, got %T: %v", err, err)
	}
	if fe.Code != 42 {
		t.Errorf("Code: got %d, want 42", fe.Code)
	}
	if fe.String != "something went wrong" {
		t.Errorf("String: got %q, want %q", fe.String, "something went wrong")
	}
}

func TestResponseErrFaultImplementsError(t *testing.T) {
	t.Parallel()
	r := faultResponse(1, "oops")
	err := r.Err()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Ensure the error message contains useful info.
	if !strings.Contains(err.Error(), "oops") {
		t.Errorf("error message %q does not contain fault string", err.Error())
	}
}

// ─── Response.Unmarshal ───────────────────────────────────────────────────────

func TestResponseUnmarshalString(t *testing.T) {
	t.Parallel()
	r := validMethodResponse("<string>hello</string>")
	var v string
	if err := r.Unmarshal(&v); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "hello" {
		t.Errorf("got %q, want %q", v, "hello")
	}
}

func TestResponseUnmarshalInt(t *testing.T) {
	t.Parallel()
	r := validMethodResponse("<int>123</int>")
	var v int64
	if err := r.Unmarshal(&v); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != 123 {
		t.Errorf("got %d, want 123", v)
	}
}

func TestResponseUnmarshalStruct(t *testing.T) {
	t.Parallel()

	type result struct {
		Name string `xmlrpc:"name"`
	}

	xml := `<struct><member><name>name</name><value><string>bar</string></value></member></struct>`
	r := validMethodResponse(xml)
	var v result
	if err := r.Unmarshal(&v); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Name != "bar" {
		t.Errorf("got %q, want %q", v.Name, "bar")
	}
}

func TestResponseUnmarshalNonPointerError(t *testing.T) {
	t.Parallel()
	r := validMethodResponse("<string>x</string>")
	var v string
	// Pass non-pointer
	err := r.Unmarshal(v)
	if err == nil {
		t.Fatal("expected error for non-pointer, got nil")
	}
}

func TestResponseUnmarshalInvalidXML(t *testing.T) {
	t.Parallel()
	r := Response([]byte("not xml at all"))
	var v string
	err := r.Unmarshal(&v)
	if err == nil {
		t.Fatal("expected error for invalid XML, got nil")
	}
}
