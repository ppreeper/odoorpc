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
		_, _ = fmt.Fprint(w, jsonrpcResponse([]float64{1, 2, 3}))
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
		_, _ = fmt.Fprint(w, jsonrpcErrorResponse(200, "Odoo server error"))
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
		_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":null}`)
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
		_, _ = fmt.Fprint(w, `not json`)
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
		_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":"`)
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
		_, _ = fmt.Fprint(w, jsonrpcResponse(true))
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
		_, _ = fmt.Fprint(w, jsonrpcResponse(float64(7)))
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
		_, _ = fmt.Fprint(w, jsonrpcResponse(float64(0)))
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
		_, _ = fmt.Fprint(w, jsonrpcErrorResponse(200, "auth error"))
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
		_, _ = fmt.Fprint(w, jsonrpcResponse(float64(42)))
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
		_, _ = fmt.Fprint(w, jsonrpcErrorResponse(200, "create failed"))
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
		_, _ = fmt.Fprint(w, jsonrpcResponse(float64(5)))
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
		_, _ = fmt.Fprint(w, jsonrpcResponse([]float64{1, 2, 3}))
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
		_, _ = fmt.Fprint(w, jsonrpcResponse([]float64{}))
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
		_, _ = fmt.Fprint(w, jsonrpcResponse([]float64{10}))
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
		_, _ = fmt.Fprint(w, jsonrpcResponse([]float64{}))
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
		_, _ = fmt.Fprint(w, jsonrpcResponse([]map[string]any{{"id": 1.0, "name": "Alice"}}))
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
		_, _ = fmt.Fprint(w, jsonrpcResponse([]map[string]any{{"id": 5.0, "name": "Bob"}}))
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
		_, _ = fmt.Fprint(w, jsonrpcResponse(true))
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
		_, _ = fmt.Fprint(w, jsonrpcResponse(true))
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
		_, _ = fmt.Fprint(w, jsonrpcResponse(true))
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

// TestExecuteSendsArgsAndEmptyKwargs pins the execute_kw layout for Execute:
// db, uid, password, model, method, args, {} — with the caller's args list
// forwarded unchanged and an empty kwargs map appended.
func TestExecuteSendsArgsAndEmptyKwargs(t *testing.T) {
	t.Parallel()

	var gotArgs []any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Params struct {
				Args []any `json:"args"`
			} `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
			return
		}
		gotArgs = req.Params.Args
		_, _ = fmt.Fprint(w, jsonrpcResponse(true))
	}))
	defer ts.Close()

	o := newJRPCTestClient(ts)
	args := []any{[]int{2}, "base.group_user"}
	if _, err := o.Execute(context.Background(), "res.users", "has_group", args); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gotArgs) != 7 {
		t.Fatalf("got %d args, want 7: %v", len(gotArgs), gotArgs)
	}
	if gotArgs[3] != "res.users" || gotArgs[4] != "has_group" {
		t.Errorf("model/method: got %v/%v", gotArgs[3], gotArgs[4])
	}
	pos, ok := gotArgs[5].([]any)
	if !ok || len(pos) != 2 {
		t.Fatalf("positional args malformed: %v", gotArgs[5])
	}
	ids, ok := pos[0].([]any)
	if !ok || len(ids) != 1 || ids[0] != float64(2) {
		t.Errorf("ids: got %v, want [2]", pos[0])
	}
	if pos[1] != "base.group_user" {
		t.Errorf("second arg: got %v, want base.group_user", pos[1])
	}
	kw, ok := gotArgs[6].(map[string]any)
	if !ok || len(kw) != 0 {
		t.Errorf("kwargs: got %v, want empty map", gotArgs[6])
	}
}

// TestExecuteKwNilArgsSendsEmptyList pins that a nil args slice is marshalled
// as [] rather than null; Odoo rejects null with "Value after * must be an
// iterable, not NoneType".
func TestExecuteKwNilArgsSendsEmptyList(t *testing.T) {
	t.Parallel()

	var gotArgs []any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Params struct {
				Args []any `json:"args"`
			} `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
			return
		}
		gotArgs = req.Params.Args
		_, _ = fmt.Fprint(w, jsonrpcResponse(true))
	}))
	defer ts.Close()

	o := newJRPCTestClient(ts)
	if _, err := o.ExecuteKw(context.Background(), "res.users", "has_group", nil, []map[string]any{{"group_ext_id": "base.group_user"}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gotArgs) != 7 {
		t.Fatalf("got %d args, want 7: %v", len(gotArgs), gotArgs)
	}
	if gotArgs[5] == nil {
		t.Fatalf("args[5] is null, want empty list")
	}
	if pos, ok := gotArgs[5].([]any); !ok || len(pos) != 0 {
		t.Errorf("positional args: got %v, want empty list", gotArgs[5])
	}
}

func TestExecuteKw(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, jsonrpcResponse(true))
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
		_, _ = fmt.Fprint(w, jsonrpcResponse(map[string]any{"name": map[string]any{"type": "char"}}))
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
		_, _ = fmt.Fprint(w, jsonrpcResponse(map[string]any{"ids": []any{10.0, 11.0}}))
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

// ─── response validation ──────────────────────────────────────────────────────

// newUnexpectedResponseClient returns a client whose server replies with a
// result of the wrong type, so callers can assert they get an error rather
// than a silent zero value.
func newUnexpectedResponseClient(t *testing.T) *OdooJSON {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, jsonrpcResponse("unexpected"))
	}))
	t.Cleanup(ts.Close)
	return newJRPCTestClient(ts)
}

func TestCreateUnexpectedResponseReturnsError(t *testing.T) {
	t.Parallel()
	o := newUnexpectedResponseClient(t)
	if _, err := o.Create(context.Background(), "res.partner", map[string]any{"name": "x"}); err == nil {
		t.Fatal("expected error for non-numeric create response, got nil")
	}
}

func TestCountUnexpectedResponseReturnsError(t *testing.T) {
	t.Parallel()
	o := newUnexpectedResponseClient(t)
	if _, err := o.Count(context.Background(), "res.partner"); err == nil {
		t.Fatal("expected error for non-numeric count response, got nil")
	}
}

func TestGetIDUnexpectedResponseReturnsError(t *testing.T) {
	t.Parallel()
	o := newUnexpectedResponseClient(t)
	if _, err := o.GetID(context.Background(), "res.partner"); err == nil {
		t.Fatal("expected error for non-list get_id response, got nil")
	}
}

func TestWriteUnexpectedResponseReturnsError(t *testing.T) {
	t.Parallel()
	o := newUnexpectedResponseClient(t)
	if _, err := o.Write(context.Background(), "res.partner", 1, map[string]any{"name": "x"}); err == nil {
		t.Fatal("expected error for non-boolean write response, got nil")
	}
}

func TestUnlinkUnexpectedResponseReturnsError(t *testing.T) {
	t.Parallel()
	o := newUnexpectedResponseClient(t)
	if _, err := o.Unlink(context.Background(), "res.partner", []int{1}); err == nil {
		t.Fatal("expected error for non-boolean unlink response, got nil")
	}
}

func TestExecuteUnexpectedResponseReturnsError(t *testing.T) {
	t.Parallel()
	o := newUnexpectedResponseClient(t)
	if _, err := o.Execute(context.Background(), "res.partner", "action_archive", []any{1}); err == nil {
		t.Fatal("expected error for non-boolean execute response, got nil")
	}
}

func TestExecuteKwUnexpectedResponseReturnsError(t *testing.T) {
	t.Parallel()
	o := newUnexpectedResponseClient(t)
	if _, err := o.ExecuteKw(context.Background(), "res.partner", "write", []any{1}, nil); err == nil {
		t.Fatal("expected error for non-boolean execute_kw response, got nil")
	}
}

// TestWriteSendsIDList pins the old-style write([ids], values) payload: the
// record id must be a list, and the values must be the next positional
// argument (no "vals" wrapper).
func TestWriteSendsIDList(t *testing.T) {
	t.Parallel()

	var gotArgs []any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Params struct {
				Args []any `json:"args"`
			} `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
			return
		}
		gotArgs = req.Params.Args
		_, _ = fmt.Fprint(w, jsonrpcResponse(true))
	}))
	defer ts.Close()

	o := newJRPCTestClient(ts)
	if _, err := o.Write(context.Background(), "res.partner", 42, map[string]any{"name": "Alice"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gotArgs) != 7 {
		t.Fatalf("got %d args, want 7: %v", len(gotArgs), gotArgs)
	}
	ids, ok := gotArgs[5].([]any)
	if !ok || len(ids) != 1 || ids[0] != float64(42) {
		t.Errorf("write ids: got %v, want [42]", gotArgs[5])
	}
	vals, ok := gotArgs[6].(map[string]any)
	if !ok || vals["name"] != "Alice" {
		t.Errorf("write values: got %v, want name=Alice", gotArgs[6])
	}
}
