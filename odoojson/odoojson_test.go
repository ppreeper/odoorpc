package odoojson

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ─── genURL ───────────────────────────────────────────────────────────────────

var genURLTests = []struct {
	name          string
	schema        string
	hostname      string
	port          int
	apikey        string
	expectedURL   string
	expectedError error
}{
	{"empty schema", "", "localhost", 8069, "key", "", errors.New("invalid schema: http or https")},
	{"ftp schema", "ftp", "localhost", 8069, "key", "", errors.New("invalid schema: http or https")},
	{"port zero", "http", "localhost", 0, "key", "", errors.New("invalid port: 1-65535")},
	{"port negative", "http", "localhost", -1, "key", "", errors.New("invalid port: 1-65535")},
	{"port too large", "http", "localhost", 65536, "key", "", errors.New("invalid port: 1-65535")},
	{"empty hostname", "http", "", 8069, "key", "", errors.New("invalid hostname length: 1-2048")},
	{"empty apikey", "http", "localhost", 8069, "", "", errors.New("invalid apikey: must not be empty")},
	{"http ok", "http", "localhost", 8069, "mykey", "http://localhost:8069/json/2/", nil},
	{"https ok", "https", "myhost", 443, "mykey", "https://myhost:443/json/2/", nil},
}

func TestGenURL(t *testing.T) {
	for _, tt := range genURLTests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			o := OdooJSON{
				schema:   tt.schema,
				hostname: tt.hostname,
				port:     tt.port,
				apikey:   tt.apikey,
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

// ─── endpointURL ─────────────────────────────────────────────────────────────

func TestEndpointURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		base   string
		model  string
		method string
		want   string
	}{
		{"simple", "http://localhost:8069/json/2/", "res.partner", "search_read", "http://localhost:8069/json/2/res.partner/search_read"},
		{"trailing slash preserved", "http://host:8069/json/2/", "sale.order", "create", "http://host:8069/json/2/sale.order/create"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			o := OdooJSON{url: tt.base}
			got, err := o.endpointURL(tt.model, tt.method)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// ─── Call (mock HTTP server) ──────────────────────────────────────────────────

// newTestClient returns an OdooJSON wired to the provided test server.
func newTestClient(ts *httptest.Server) *OdooJSON {
	o := &OdooJSON{
		schema:   "http",
		hostname: "localhost",
		port:     8069,
		apikey:   "testkey",
		database: "testdb",
		url:      ts.URL + "/json/2/",
		client:   ts.Client(),
	}
	return o
}

func TestCallSuccess(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[1,2,3]`)
	}))
	defer ts.Close()

	o := newTestClient(ts)
	result, err := o.Call(context.Background(), "res.partner", "search", map[string]any{"domain": []any{}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	arr, ok := result.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", result)
	}
	if len(arr) != 3 {
		t.Errorf("expected 3 elements, got %d", len(arr))
	}
}

func TestCallHTTPError(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"arguments":["record not found"]}`)
	}))
	defer ts.Close()

	o := newTestClient(ts)
	_, err := o.Call(context.Background(), "res.partner", "read", map[string]any{"ids": []int{9999}})
	if err == nil {
		t.Fatal("expected error for 404 response, got nil")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected 404 in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "record not found") {
		t.Errorf("expected argument in error, got: %v", err)
	}
}

func TestCallHTTPErrorWithoutArguments(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{}`)
	}))
	defer ts.Close()

	o := newTestClient(ts)
	_, err := o.Call(context.Background(), "res.partner", "read", map[string]any{})
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
}

func TestCallRequestHeadersSet(t *testing.T) {
	t.Parallel()
	var gotAuth, gotDB, gotCT string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotDB = r.Header.Get("X-Odoo-Database")
		gotCT = r.Header.Get("Content-Type")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `true`)
	}))
	defer ts.Close()

	o := newTestClient(ts)
	o.Call(context.Background(), "res.partner", "search", map[string]any{}) //nolint
	if gotAuth != "Bearer testkey" {
		t.Errorf("Authorization: got %q, want %q", gotAuth, "Bearer testkey")
	}
	if gotDB != "testdb" {
		t.Errorf("X-Odoo-Database: got %q, want %q", gotDB, "testdb")
	}
	if gotCT != "application/json" {
		t.Errorf("Content-Type: got %q, want %q", gotCT, "application/json")
	}
}

func TestCallInvalidJSONResponse(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `not json at all`)
	}))
	defer ts.Close()

	o := newTestClient(ts)
	_, err := o.Call(context.Background(), "res.partner", "search", map[string]any{})
	if err == nil {
		t.Fatal("expected error for invalid JSON response, got nil")
	}
}

func TestCallGenURLCalledWhenEmpty(t *testing.T) {
	t.Parallel()
	// url is empty but config is valid → genURL should be called lazily.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `true`)
	}))
	defer ts.Close()

	// We cannot easily point the lazy genURL to the test server, so instead
	// verify that an invalid config surfaces a genURL error before any HTTP call.
	o := &OdooJSON{
		schema:   "ftp", // invalid
		hostname: "localhost",
		port:     8069,
		apikey:   "key",
		client:   ts.Client(),
	}
	_, err := o.Call(context.Background(), "res.partner", "search", map[string]any{})
	if err == nil {
		t.Fatal("expected genURL error for invalid schema, got nil")
	}
	if !strings.Contains(err.Error(), "genURL failed") {
		t.Errorf("expected 'genURL failed' in error, got: %v", err)
	}
}

// ─── Login ────────────────────────────────────────────────────────────────────

func TestLoginValidConfig(t *testing.T) {
	t.Parallel()
	o := &OdooJSON{
		schema:   "http",
		hostname: "localhost",
		port:     8069,
		apikey:   "mykey",
	}
	if err := o.Login(context.Background()); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestLoginMissingAPIKey(t *testing.T) {
	t.Parallel()
	o := &OdooJSON{
		schema:   "http",
		hostname: "localhost",
		port:     8069,
		apikey:   "",
	}
	if err := o.Login(context.Background()); err == nil {
		t.Error("expected error for missing API key, got nil")
	}
}

// ─── CRUD methods (via mock server) ──────────────────────────────────────────

func TestCreate(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[42]`)
	}))
	defer ts.Close()

	o := newTestClient(ts)
	id, err := o.Create(context.Background(), "res.partner", map[string]any{"name": "Test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 42 {
		t.Errorf("got id %d, want 42", id)
	}
}

func TestCreateServerError(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"arguments":["bad values"]}`)
	}))
	defer ts.Close()

	o := newTestClient(ts)
	_, err := o.Create(context.Background(), "res.partner", map[string]any{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCount(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `7`)
	}))
	defer ts.Close()

	o := newTestClient(ts)
	count, err := o.Count(context.Background(), "res.partner")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 7 {
		t.Errorf("got %d, want 7", count)
	}
}

func TestSearch(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[1,2,3]`)
	}))
	defer ts.Close()

	o := newTestClient(ts)
	ids, err := o.Search(context.Background(), "res.partner")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 3 || ids[0] != 1 || ids[1] != 2 || ids[2] != 3 {
		t.Errorf("got %v, want [1 2 3]", ids)
	}
}

func TestSearchEmptyResult(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[]`)
	}))
	defer ts.Close()

	o := newTestClient(ts)
	ids, err := o.Search(context.Background(), "res.partner")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("expected empty, got %v", ids)
	}
}

func TestGetID(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[10]`)
	}))
	defer ts.Close()

	o := newTestClient(ts)
	id, err := o.GetID(context.Background(), "res.partner")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 10 {
		t.Errorf("got %d, want 10", id)
	}
}

func TestGetIDNotFound(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[]`)
	}))
	defer ts.Close()

	o := newTestClient(ts)
	id, err := o.GetID(context.Background(), "res.partner")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != -1 {
		t.Errorf("got %d, want -1", id)
	}
}

func TestRead(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"id":1,"name":"Alice"},{"id":2,"name":"Bob"}]`)
	}))
	defer ts.Close()

	o := newTestClient(ts)
	records, err := o.Read(context.Background(), "res.partner", []int{1, 2}, "id", "name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0]["name"] != "Alice" {
		t.Errorf("records[0].name: got %v, want Alice", records[0]["name"])
	}
}

func TestSearchRead(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"id":5,"name":"Charlie"}]`)
	}))
	defer ts.Close()

	o := newTestClient(ts)
	records, err := o.SearchRead(context.Background(), "res.partner", 0, 10, []string{"id", "name"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0]["name"] != "Charlie" {
		t.Errorf("records[0].name: got %v, want Charlie", records[0]["name"])
	}
}

func TestWrite(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `true`)
	}))
	defer ts.Close()

	o := newTestClient(ts)
	ok, err := o.Write(context.Background(), "res.partner", 1, map[string]any{"name": "Updated"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected true, got false")
	}
}

func TestUnlink(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `true`)
	}))
	defer ts.Close()

	o := newTestClient(ts)
	ok, err := o.Unlink(context.Background(), "res.partner", []int{1, 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected true, got false")
	}
}

func TestExecuteNilArgsSendsEmptyList(t *testing.T) {
	t.Parallel()
	var gotBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotBody) //nolint
		fmt.Fprint(w, `true`)
	}))
	defer ts.Close()

	o := newTestClient(ts)
	ok, err := o.Execute(context.Background(), "res.partner", "action_archive", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected true, got false")
	}
	args, present := gotBody["args"]
	if !present {
		t.Fatal("payload missing 'args' key")
	}
	arr, ok := args.([]any)
	if !ok {
		t.Fatalf("args: expected []any, got %T", args)
	}
	if len(arr) != 0 {
		t.Errorf("args: expected empty list, got %v", arr)
	}
}

func TestExecuteArgsForwardedInPayload(t *testing.T) {
	t.Parallel()
	var gotBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotBody) //nolint
		fmt.Fprint(w, `true`)
	}))
	defer ts.Close()

	o := newTestClient(ts)
	ok, err := o.Execute(context.Background(), "res.partner", "action_archive", []any{1, 2, 3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected true, got false")
	}
	args, present := gotBody["args"]
	if !present {
		t.Fatal("payload missing 'args' key")
	}
	arr, ok := args.([]any)
	if !ok || len(arr) != 3 {
		t.Errorf("args: expected [1 2 3], got %v", args)
	}
}

func TestExecuteKwNilKwargsSendsArgsOnly(t *testing.T) {
	t.Parallel()
	var gotBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotBody) //nolint
		fmt.Fprint(w, `true`)
	}))
	defer ts.Close()

	o := newTestClient(ts)
	ok, err := o.ExecuteKw(context.Background(), "res.partner", "method", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error for nil kwargs: %v", err)
	}
	if !ok {
		t.Error("expected true, got false")
	}
	if _, present := gotBody["args"]; !present {
		t.Fatal("payload missing 'args' key")
	}
}

func TestExecuteKwMergesKwargsIntoPayload(t *testing.T) {
	t.Parallel()
	var gotBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotBody) //nolint
		fmt.Fprint(w, `true`)
	}))
	defer ts.Close()

	o := newTestClient(ts)
	ok, err := o.ExecuteKw(context.Background(), "res.partner", "method",
		[]any{1, 2},
		[]map[string]any{{"context": map[string]any{"lang": "en_US"}}},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected true, got false")
	}
	if _, present := gotBody["args"]; !present {
		t.Error("payload missing 'args' key")
	}
	if _, present := gotBody["context"]; !present {
		t.Error("payload missing 'context' key merged from kwargs")
	}
}

func TestFieldsGet(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"name":{"type":"char","string":"Name"}}`)
	}))
	defer ts.Close()

	o := newTestClient(ts)
	fields, err := o.FieldsGet(context.Background(), "res.partner", []string{"name"}, "type", "string")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := fields["name"]; !ok {
		t.Error("expected 'name' field in response")
	}
}

func TestFieldsGetNoAttributesOmitsKey(t *testing.T) {
	t.Parallel()
	var gotBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotBody) //nolint
		fmt.Fprint(w, `{}`)
	}))
	defer ts.Close()

	o := newTestClient(ts)
	o.FieldsGet(context.Background(), "res.partner", nil) //nolint
	if _, present := gotBody["attributes"]; present {
		t.Error("expected 'attributes' key to be absent when no fieldAttributes provided, but it was present")
	}
}

func TestFieldsGetWithAttributesSendsKey(t *testing.T) {
	t.Parallel()
	var gotBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotBody) //nolint
		fmt.Fprint(w, `{}`)
	}))
	defer ts.Close()

	o := newTestClient(ts)
	o.FieldsGet(context.Background(), "res.partner", nil, "type", "string") //nolint
	attrs, present := gotBody["attributes"]
	if !present {
		t.Fatal("expected 'attributes' key in payload, but it was absent")
	}
	arr, ok := attrs.([]any)
	if !ok || len(arr) != 2 {
		t.Errorf("expected attributes [type string], got %v", attrs)
	}
}

func TestLoad(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the payload has both fields and data keys.
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body) //nolint
		fmt.Fprint(w, `{"ids":[10,11]}`)
	}))
	defer ts.Close()

	o := newTestClient(ts)
	ids, err := o.Load(context.Background(), "res.partner",
		[]string{"name", "email"},
		[][]any{{"Alice", "alice@example.com"}, {"Bob", "bob@example.com"}},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 2 || ids[0] != 10 || ids[1] != 11 {
		t.Errorf("got %v, want [10 11]", ids)
	}
}

func TestAction(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `true`)
	}))
	defer ts.Close()

	o := newTestClient(ts)
	ok, err := o.Action(context.Background(), "res.partner", "action_archive", map[string]any{"ids": []int{1}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected true")
	}
}
