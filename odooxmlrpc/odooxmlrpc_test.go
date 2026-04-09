package odooxmlrpc

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ppreeper/odoorpc/xmlrpc"
)

var genURLTests = []struct {
	name          string
	schema        string
	hostname      string
	port          int
	expectedURL   string
	expectedError error
}{
	{"empty schema", "", "localhost", 8069, "", errors.New("invalid schema: http or https")},
	{"ftp schema", "ftp", "localhost", 8069, "", errors.New("invalid schema: http or https")},
	{"no schema no host no port", "", "", 0, "", errors.New("invalid schema: http or https")},
	{"port zero", "http", "localhost", 0, "", errors.New("invalid port: 1-65535")},
	{"port negative", "http", "localhost", -1, "", errors.New("invalid port: 1-65535")},
	{"port too large", "https", "localhost", 65536, "", errors.New("invalid port: 1-65535")},
	{"empty hostname", "http", "", 8069, "", errors.New("invalid hostname length: 1-2048")},
	{"http ok", "http", "localhost", 8069, "http://localhost:8069/xmlrpc/2/", nil},
	{"https ok", "https", "myhost", 443, "https://myhost:443/xmlrpc/2/", nil},
}

func TestGenURL(t *testing.T) {
	for _, tt := range genURLTests {
		tt := tt
		name := fmt.Sprintf("%s://%s:%d", tt.schema, tt.hostname, tt.port)
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			o := OdooXML{
				schema:   tt.schema,
				hostname: tt.hostname,
				port:     tt.port,
			}
			err := o.genURL()
			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("expected error %q, got nil", tt.expectedError)
					return
				}
				if err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error %q, got %q", tt.expectedError, err)
				}
				if o.url != "" {
					t.Errorf("url should be empty on error, got %q", o.url)
				}
				return
			}
			if err != nil {
				t.Errorf("expected no error, got %q", err)
				return
			}
			if o.url != tt.expectedURL {
				t.Errorf("expected url %q, got %q", tt.expectedURL, o.url)
			}
		})
	}
}

// xmlrpcResponse wraps innerXML in a minimal XML-RPC methodResponse envelope.
func xmlrpcResponse(innerXML string) string {
	return `<?xml version="1.0"?><methodResponse><params><param><value>` +
		innerXML + `</value></param></params></methodResponse>`
}

// newXMLTestServer creates an httptest.Server that always returns the provided
// XML-RPC response body, and returns an OdooXML pre-wired to it.
func newXMLTestServer(t *testing.T, body string) (*httptest.Server, *OdooXML) {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml")
		fmt.Fprint(w, body)
	}))
	o := &OdooXML{
		schema:   "http",
		hostname: "localhost",
		port:     8069,
		database: "testdb",
		username: "admin",
		password: "secret",
		url:      ts.URL + "/xmlrpc/2/",
	}
	return ts, o
}

// ─── Login ────────────────────────────────────────────────────────────────────

func TestLoginSuccess(t *testing.T) {
	t.Parallel()
	// Odoo returns the uid (e.g. 7) on successful authenticate.
	ts, o := newXMLTestServer(t, xmlrpcResponse("<int>7</int>"))
	defer ts.Close()

	if err := o.Login(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if o.uid != 7 {
		t.Errorf("uid: got %d, want 7", o.uid)
	}
}

func TestLoginBadCredentials(t *testing.T) {
	t.Parallel()
	// Odoo returns 0 when credentials are wrong.
	ts, o := newXMLTestServer(t, xmlrpcResponse("<int>0</int>"))
	defer ts.Close()

	err := o.Login(context.Background())
	if err == nil {
		t.Fatal("expected error for bad credentials, got nil")
	}
	if !strings.Contains(err.Error(), "invalid credentials") {
		t.Errorf("expected 'invalid credentials' in error, got: %v", err)
	}
}

func TestLoginGenURLFailure(t *testing.T) {
	t.Parallel()
	o := &OdooXML{
		schema:   "ftp",
		hostname: "localhost",
		port:     8069,
	}
	err := o.Login(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "genURL failed") {
		t.Errorf("expected 'genURL failed' in error, got: %v", err)
	}
}

// ─── CRUD helpers ─────────────────────────────────────────────────────────────

// newQueueServer creates an httptest.Server that serves responses from a
// pre-populated queue (one per request, in order). It also captures the raw
// request body of each call in requestBodies.
func newQueueServer(t *testing.T, responses []string) (ts *httptest.Server, requestBodies *[]string) {
	t.Helper()
	var idx atomic.Int32
	bodies := make([]string, 0, len(responses))
	requestBodies = &bodies

	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Capture request body for inspection.
		b, err := io.ReadAll(r.Body)
		if err == nil {
			bodies = append(bodies, string(b))
		}

		i := int(idx.Add(1)) - 1
		if i >= len(responses) {
			http.Error(w, "no more responses", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/xml")
		fmt.Fprint(w, responses[i])
	}))
	return ts, requestBodies
}

// newXMLCRUDClient builds an OdooXML with common and models clients already
// wired to ts, bypassing Login. uid is set to 1.
func newXMLCRUDClient(t *testing.T, ts *httptest.Server) *OdooXML {
	t.Helper()
	transport := ts.Client().Transport
	common, err := xmlrpc.NewClient(ts.URL+"/xmlrpc/2/common", transport)
	if err != nil {
		t.Fatalf("newXMLCRUDClient common: %v", err)
	}
	models, err := xmlrpc.NewClient(ts.URL+"/xmlrpc/2/object", transport)
	if err != nil {
		t.Fatalf("newXMLCRUDClient models: %v", err)
	}
	return &OdooXML{
		schema:   "http",
		hostname: "localhost",
		port:     8069,
		database: "testdb",
		username: "admin",
		password: "secret",
		uid:      1,
		url:      ts.URL + "/xmlrpc/2/",
		common:   common,
		models:   models,
	}
}

// ─── Load (#4) ────────────────────────────────────────────────────────────────

func TestLoadUnexpectedResponseReturnsError(t *testing.T) {
	t.Parallel()
	// Return a plain boolean — not a struct or int64 — to trigger the default
	// branch that previously returned []int{-1} silently.
	ts, _ := newQueueServer(t, []string{xmlrpcResponse("<boolean>0</boolean>")})
	defer ts.Close()

	o := newXMLCRUDClient(t, ts)
	ids, err := o.Load(context.Background(), "res.partner",
		[]string{"name"}, [][]any{{"Alice"}})
	if err == nil {
		t.Fatalf("expected error for unexpected response type, got ids=%v", ids)
	}
	if !strings.Contains(err.Error(), "unexpected response type") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// ─── SearchRead (#3) ──────────────────────────────────────────────────────────

func TestSearchReadSendsZeroOffsetAndLimit(t *testing.T) {
	t.Parallel()
	// Return an empty array so SearchRead succeeds with no records.
	ts, reqBodies := newQueueServer(t, []string{
		xmlrpcResponse("<array><data></data></array>"),
	})
	defer ts.Close()

	o := newXMLCRUDClient(t, ts)
	_, err := o.SearchRead(context.Background(), "res.partner", 0, 0, []string{"name"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(*reqBodies) == 0 {
		t.Fatal("no request body captured")
	}
	body := (*reqBodies)[0]
	// The request XML must contain both offset and limit values.
	if !strings.Contains(body, "offset") {
		t.Errorf("expected 'offset' in request body, got:\n%s", body)
	}
	if !strings.Contains(body, "limit") {
		t.Errorf("expected 'limit' in request body, got:\n%s", body)
	}
}

// ─── Write (#5) ───────────────────────────────────────────────────────────────

func TestWriteSendsCorrectStructure(t *testing.T) {
	t.Parallel()
	ts, reqBodies := newQueueServer(t, []string{xmlrpcResponse("<boolean>1</boolean>")})
	defer ts.Close()

	o := newXMLCRUDClient(t, ts)
	result, err := o.Write(context.Background(), "res.partner", 42, map[string]any{"name": "Alice"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result {
		t.Error("expected true, got false")
	}

	if len(*reqBodies) == 0 {
		t.Fatal("no request body captured")
	}
	body := (*reqBodies)[0]

	// The call must NOT wrap ids and vals together in a single extra array.
	// Verify the request XML is parseable and contains "vals" as a named member.
	if _, err := xml.NewDecoder(strings.NewReader(body)).Token(); err != nil {
		t.Errorf("request body is not valid XML: %v", err)
	}
	if !strings.Contains(body, "vals") {
		t.Errorf("expected 'vals' keyword in request body, got:\n%s", body)
	}
}
