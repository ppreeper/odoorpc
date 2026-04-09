package odoojrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

// jsonrpcResponse builds a minimal Odoo JSON-RPC success response body.
func jsonrpcResponse(result any) string {
	b, _ := json.Marshal(result)
	return fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"result":%s}`, string(b))
}

// jsonrpcErrorResponse builds a minimal Odoo JSON-RPC error response body.
func jsonrpcErrorResponse(code int, msg string) string {
	return fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"error":{"code":%d,"message":%q,"data":{"message":"detail"}}}`, code, msg)
}

// newJRPCTestClient returns an OdooJSON wired to the provided test server.
func newJRPCTestClient(ts *httptest.Server) *OdooJSON {
	return &OdooJSON{
		schema:   "http",
		hostname: "localhost",
		port:     8069,
		database: "testdb",
		username: "admin",
		password: "secret",
		uid:      1,
		url:      ts.URL + "/jsonrpc/",
		client:   ts.Client(),
	}
}

// ─── Call ─────────────────────────────────────────────────────────────────────

func TestCallSuccess(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, jsonrpcResponse([]float64{1, 2, 3}))
	}))
	defer ts.Close()

	o := newJRPCTestClient(ts)
	result, err := o.Call(context.Background(), "object", "execute", "db", 1, "pass")
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

func TestCallRPCError(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, jsonrpcErrorResponse(200, "Odoo server error"))
	}))
	defer ts.Close()

	o := newJRPCTestClient(ts)
	_, err := o.Call(context.Background(), "object", "execute")
	if err == nil {
		t.Fatal("expected RPC error, got nil")
	}
	if !strings.Contains(err.Error(), "Odoo server error") {
		t.Errorf("expected server error in message, got: %v", err)
	}
}

func TestCallNullResult(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// result=null is treated as an error by decodeClientResponse
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":null}`)
	}))
	defer ts.Close()

	o := newJRPCTestClient(ts)
	_, err := o.Call(context.Background(), "object", "execute")
	if err == nil {
		t.Fatal("expected error for null result, got nil")
	}
}

func TestCallInvalidJSON(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `not json`)
	}))
	defer ts.Close()

	o := newJRPCTestClient(ts)
	_, err := o.Call(context.Background(), "object", "execute")
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestCallResponseSizeCap(t *testing.T) {
	t.Parallel()
	// Serve a response larger than maxResponseBytes; the LimitReader truncates
	// it so the JSON decoder sees an incomplete document and returns an error.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Write a valid-looking prefix, then flood with bytes beyond the cap.
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":"`)
		buf := make([]byte, 1024)
		for i := range buf {
			buf[i] = 'a'
		}
		written := 0
		for written < maxResponseBytes+1 {
			n, err := w.Write(buf)
			written += n
			if err != nil {
				break
			}
		}
	}))
	defer ts.Close()

	o := newJRPCTestClient(ts)
	_, err := o.Call(context.Background(), "object", "execute")
	if err == nil {
		t.Fatal("expected error for oversized response, got nil")
	}
}

func TestCallGenURLFailure(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer ts.Close()

	// url empty, schema invalid → genURL should fail
	o := &OdooJSON{
		schema:   "ftp",
		hostname: "localhost",
		port:     8069,
		client:   ts.Client(),
	}
	_, err := o.Call(context.Background(), "common", "login")
	if err == nil {
		t.Fatal("expected genURL error, got nil")
	}
	if !strings.Contains(err.Error(), "genURL failed") {
		t.Errorf("expected 'genURL failed' in error, got: %v", err)
	}
}

func TestCallSetsContentTypeHeader(t *testing.T) {
	t.Parallel()
	var gotCT string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		fmt.Fprint(w, jsonrpcResponse(true))
	}))
	defer ts.Close()

	o := newJRPCTestClient(ts)
	o.Call(context.Background(), "object", "execute") //nolint
	if gotCT != "application/json" {
		t.Errorf("Content-Type: got %q, want %q", gotCT, "application/json")
	}
}

// ─── Login ────────────────────────────────────────────────────────────────────

func TestLoginSuccess(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, jsonrpcResponse(float64(7)))
	}))
	defer ts.Close()

	o := newJRPCTestClient(ts)
	o.uid = 0 // reset so Login actually sets it
	if err := o.Login(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if o.uid != 7 {
		t.Errorf("uid: got %d, want 7", o.uid)
	}
}

func TestLoginBadCredentials(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Odoo returns false (0.0) when credentials are wrong.
		fmt.Fprint(w, jsonrpcResponse(float64(0)))
	}))
	defer ts.Close()

	o := newJRPCTestClient(ts)
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
	o := &OdooJSON{
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

func TestLoginRPCError(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, jsonrpcErrorResponse(200, "auth error"))
	}))
	defer ts.Close()

	o := newJRPCTestClient(ts)
	err := o.Login(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ─── CRUD methods ─────────────────────────────────────────────────────────────

func TestCreate(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, jsonrpcResponse(float64(42)))
	}))
	defer ts.Close()

	o := newJRPCTestClient(ts)
	id, err := o.Create(context.Background(), "res.partner", map[string]any{"name": "Test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 42 {
		t.Errorf("got %d, want 42", id)
	}
}

func TestCreateRPCError(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, jsonrpcErrorResponse(200, "create failed"))
	}))
	defer ts.Close()

	o := newJRPCTestClient(ts)
	id, err := o.Create(context.Background(), "res.partner", map[string]any{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if id != -1 {
		t.Errorf("expected -1 on error, got %d", id)
	}
}

func TestCount(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, jsonrpcResponse(float64(5)))
	}))
	defer ts.Close()

	o := newJRPCTestClient(ts)
	count, err := o.Count(context.Background(), "res.partner")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 5 {
		t.Errorf("got %d, want 5", count)
	}
}

func TestSearch(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, jsonrpcResponse([]float64{1, 2, 3}))
	}))
	defer ts.Close()

	o := newJRPCTestClient(ts)
	ids, err := o.Search(context.Background(), "res.partner")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 3 || ids[0] != 1 || ids[1] != 2 || ids[2] != 3 {
		t.Errorf("got %v, want [1 2 3]", ids)
	}
}

func TestSearchEmpty(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, jsonrpcResponse([]float64{}))
	}))
	defer ts.Close()

	o := newJRPCTestClient(ts)
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
		fmt.Fprint(w, jsonrpcResponse([]float64{10}))
	}))
	defer ts.Close()

	o := newJRPCTestClient(ts)
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
		fmt.Fprint(w, jsonrpcResponse([]float64{}))
	}))
	defer ts.Close()

	o := newJRPCTestClient(ts)
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
		fmt.Fprint(w, jsonrpcResponse([]map[string]any{{"id": 1.0, "name": "Alice"}}))
	}))
	defer ts.Close()

	o := newJRPCTestClient(ts)
	records, err := o.Read(context.Background(), "res.partner", []int{1}, "id", "name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0]["name"] != "Alice" {
		t.Errorf("name: got %v, want Alice", records[0]["name"])
	}
}

func TestSearchRead(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, jsonrpcResponse([]map[string]any{{"id": 5.0, "name": "Bob"}}))
	}))
	defer ts.Close()

	o := newJRPCTestClient(ts)
	records, err := o.SearchRead(context.Background(), "res.partner", 0, 10, []string{"id", "name"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0]["name"] != "Bob" {
		t.Errorf("name: got %v, want Bob", records[0]["name"])
	}
}

func TestWrite(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, jsonrpcResponse(true))
	}))
	defer ts.Close()

	o := newJRPCTestClient(ts)
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
		fmt.Fprint(w, jsonrpcResponse(true))
	}))
	defer ts.Close()

	o := newJRPCTestClient(ts)
	ok, err := o.Unlink(context.Background(), "res.partner", []int{1, 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected true, got false")
	}
}

func TestExecute(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, jsonrpcResponse(true))
	}))
	defer ts.Close()

	o := newJRPCTestClient(ts)
	ok, err := o.Execute(context.Background(), "res.partner", "action_archive", []any{1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected true, got false")
	}
}

func TestExecuteKw(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, jsonrpcResponse(true))
	}))
	defer ts.Close()

	o := newJRPCTestClient(ts)
	ok, err := o.ExecuteKw(context.Background(), "res.partner", "search_read", nil, []map[string]any{{"domain": []any{}}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected true, got false")
	}
}

func TestFieldsGet(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, jsonrpcResponse(map[string]any{"name": map[string]any{"type": "char"}}))
	}))
	defer ts.Close()

	o := newJRPCTestClient(ts)
	fields, err := o.FieldsGet(context.Background(), "res.partner", []string{"name"}, "type")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := fields["name"]; !ok {
		t.Error("expected 'name' in fields response")
	}
}

func TestLoad(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, jsonrpcResponse(map[string]any{"ids": []any{10.0, 11.0}}))
	}))
	defer ts.Close()

	o := newJRPCTestClient(ts)
	ids, err := o.Load(context.Background(), "res.partner",
		[]string{"name"},
		[][]any{{"Alice"}, {"Bob"}},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 2 || ids[0] != 10 || ids[1] != 11 {
		t.Errorf("got %v, want [10 11]", ids)
	}
}
