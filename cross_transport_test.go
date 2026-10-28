package odoorpc_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	odoojrpc "github.com/ppreeper/odoorpc/odoojrpc"
	odoojson "github.com/ppreeper/odoorpc/odoojson"
	odooxmlrpc "github.com/ppreeper/odoorpc/odooxmlrpc"
)

// parseHostPort splits an httptest server URL into host and numeric port.
func parseHostPort(t *testing.T, raw string) (host string, port int) {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	h, p, err := net.SplitHostPort(u.Host)
	if err != nil {
		t.Fatalf("split hostport: %v", err)
	}
	pi, err := strconv.Atoi(p)
	if err != nil {
		t.Fatalf("atoi port: %v", err)
	}
	return h, pi
}

// TestCrossTransportExecuteKwConsistency calls ExecuteKw on all three transports
// and asserts that the outgoing requests contain the same logical pieces:
// model, method and kwargs (ids/vals/name). Positional arguments are omitted
// because the JSON API v2 has no positional argument list; its payload keys are
// the keyword arguments.
func TestCrossTransportExecuteKwConsistency(t *testing.T) {
	t.Parallel()

	model := "res.partner"
	method := "write"
	kwargs := []map[string]any{{"ids": []int{42}, "vals": map[string]any{"name": "Alice"}}}

	var jsonBody, jsonrpcBody, xmlBody string

	// JSON API server (odoojson)
	tsJSON := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		jsonBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `true`)
	}))
	defer tsJSON.Close()

	// JSON-RPC server (odoojrpc)
	tsJSONRPC := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		jsonrpcBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":true}`)
	}))
	defer tsJSONRPC.Close()

	// XML-RPC server (odooxmlrpc)
	tsXML := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		xmlBody = string(b)
		w.Header().Set("Content-Type", "text/xml")
		// Respond to authenticate requests with an int uid, otherwise true.
		if strings.Contains(xmlBody, "authenticate") || strings.Contains(xmlBody, "<methodName>authenticate</methodName>") {
			_, _ = fmt.Fprint(w, `<?xml version="1.0"?><methodResponse><params><param><value><int>1</int></value></param></params></methodResponse>`)
			return
		}
		_, _ = fmt.Fprint(w, `<?xml version="1.0"?><methodResponse><params><param><value><boolean>1</boolean></value></param></params></methodResponse>`)
	}))
	defer tsXML.Close()

	// --- odoojson client ---
	host, port := parseHostPort(t, tsJSON.URL)
	ojson := odoojson.NewOdoo().WithHostname(host).WithPort(port).WithDatabase("testdb").WithAPIKey("testkey")
	if _, err := ojson.ExecuteKw(context.Background(), model, method, nil, kwargs); err != nil {
		t.Fatalf("odoojson ExecuteKw failed: %v", err)
	}

	// --- odoojrpc client ---
	host, port = parseHostPort(t, tsJSONRPC.URL)
	orpc := odoojrpc.NewOdoo().WithHostname(host).WithPort(port).WithDatabase("testdb").WithUsername("admin").WithPassword("secret")
	if _, err := orpc.ExecuteKw(context.Background(), model, method, nil, kwargs); err != nil {
		t.Fatalf("odoojrpc ExecuteKw failed: %v", err)
	}

	// --- odooxmlrpc client ---
	host, port = parseHostPort(t, tsXML.URL)
	oxml := odooxmlrpc.NewOdoo().WithHostname(host).WithPort(port).WithDatabase("testdb").WithUsername("admin").WithPassword("secret")
	if err := oxml.Login(context.Background()); err != nil {
		t.Fatalf("odooxmlrpc login failed: %v", err)
	}
	if _, err := oxml.ExecuteKw(context.Background(), model, method, nil, kwargs); err != nil {
		t.Fatalf("odooxmlrpc ExecuteKw failed: %v", err)
	}

	// --- Inspect JSON API body ---
	var jpayload map[string]any
	if err := json.Unmarshal([]byte(jsonBody), &jpayload); err != nil {
		t.Fatalf("failed to decode odoojson body: %v; raw=%s", err, jsonBody)
	}
	if _, ok := jpayload["args"]; ok {
		t.Fatalf("odoojson payload must not contain 'args': %v", jpayload)
	}
	if _, ok := jpayload["ids"]; !ok {
		t.Fatalf("odoojson payload missing 'ids' merged from kwargs: %v", jpayload)
	}
	if _, ok := jpayload["vals"]; !ok {
		t.Fatalf("odoojson payload missing 'vals' merged from kwargs: %v", jpayload)
	}

	// --- Inspect JSON-RPC body ---
	var rpcmap map[string]any
	if err := json.Unmarshal([]byte(jsonrpcBody), &rpcmap); err != nil {
		t.Fatalf("failed to decode odoojrpc body: %v; raw=%s", err, jsonrpcBody)
	}
	params, ok := rpcmap["params"].(map[string]any)
	if !ok {
		t.Fatalf("odoojrpc missing params: %v", rpcmap)
	}
	// service/method sanity
	if params["service"] != "object" || params["method"] != "execute_kw" {
		t.Fatalf("odoojrpc unexpected service/method: %v", params)
	}
	parr, ok := params["args"].([]any)
	if !ok || len(parr) < 7 {
		t.Fatalf("odoojrpc params.args malformed: %v", params["args"])
	}
	// model and method positions
	if parr[3] != model || parr[4] != method {
		t.Fatalf("odoojrpc model/method mismatch: got %v/%v, want %s/%s", parr[3], parr[4], model, method)
	}
	// positional args list should be at index 5 and empty (nil encodes as null)
	if parr[5] != nil {
		if pArgs, ok := parr[5].([]any); !ok || len(pArgs) != 0 {
			t.Fatalf("odoojrpc positional args: got %v, want empty", parr[5])
		}
	}
	// kwargs map should be at index 6
	pKw, ok := parr[6].(map[string]any)
	if !ok {
		t.Fatalf("odoojrpc kwargs malformed: %v", parr[6])
	}
	if _, ok := pKw["ids"]; !ok {
		t.Fatalf("odoojrpc kwargs missing ids: %v", pKw)
	}
	if _, ok := pKw["vals"]; !ok {
		t.Fatalf("odoojrpc kwargs missing vals: %v", pKw)
	}

	// --- Inspect XML-RPC body ---
	if !strings.Contains(xmlBody, "<methodName>execute_kw</methodName>") {
		t.Fatalf("xmlrpc body missing execute_kw method name: %s", xmlBody)
	}
	if !strings.Contains(xmlBody, model) || !strings.Contains(xmlBody, method) {
		t.Fatalf("xmlrpc body missing model/method: model=%s method=%s body=%s", model, method, xmlBody)
	}
	if !strings.Contains(xmlBody, "<name>ids</name>") || !strings.Contains(xmlBody, "<name>vals</name>") {
		t.Fatalf("xmlrpc kwargs missing ids/vals names: %s", xmlBody)
	}
	if !strings.Contains(xmlBody, "Alice") {
		t.Fatalf("xmlrpc body missing value Alice: %s", xmlBody)
	}
}

// TestCrossTransportWriteConsistency pins the canonical Odoo write payload
// shape across all three transports: write([ids], values) — the record id
// as a list followed by the values, with no "vals" wrapper key.
func TestCrossTransportWriteConsistency(t *testing.T) {
	t.Parallel()

	model := "res.partner"
	recordID := 42
	values := map[string]any{"name": "Alice"}

	var jsonBody, jsonrpcBody, xmlBody string

	// JSON API server (odoojson)
	tsJSON := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		jsonBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `true`)
	}))
	defer tsJSON.Close()

	// JSON-RPC server (odoojrpc)
	tsJSONRPC := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		jsonrpcBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":true}`)
	}))
	defer tsJSONRPC.Close()

	// XML-RPC server (odooxmlrpc)
	tsXML := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		xmlBody = string(b)
		w.Header().Set("Content-Type", "text/xml")
		if strings.Contains(xmlBody, "authenticate") {
			_, _ = fmt.Fprint(w, `<?xml version="1.0"?><methodResponse><params><param><value><int>1</int></value></param></params></methodResponse>`)
			return
		}
		_, _ = fmt.Fprint(w, `<?xml version="1.0"?><methodResponse><params><param><value><boolean>1</boolean></value></param></params></methodResponse>`)
	}))
	defer tsXML.Close()

	// --- odoojson client ---
	host, port := parseHostPort(t, tsJSON.URL)
	ojson := odoojson.NewOdoo().WithHostname(host).WithPort(port).WithDatabase("testdb").WithAPIKey("testkey")
	if _, err := ojson.Write(context.Background(), model, recordID, values); err != nil {
		t.Fatalf("odoojson Write failed: %v", err)
	}

	// --- odoojrpc client ---
	host, port = parseHostPort(t, tsJSONRPC.URL)
	orpc := odoojrpc.NewOdoo().WithHostname(host).WithPort(port).WithDatabase("testdb").WithUsername("admin").WithPassword("secret")
	if _, err := orpc.Write(context.Background(), model, recordID, values); err != nil {
		t.Fatalf("odoojrpc Write failed: %v", err)
	}

	// --- odooxmlrpc client ---
	host, port = parseHostPort(t, tsXML.URL)
	oxml := odooxmlrpc.NewOdoo().WithHostname(host).WithPort(port).WithDatabase("testdb").WithUsername("admin").WithPassword("secret")
	if err := oxml.Login(context.Background()); err != nil {
		t.Fatalf("odooxmlrpc login failed: %v", err)
	}
	if _, err := oxml.Write(context.Background(), model, recordID, values); err != nil {
		t.Fatalf("odooxmlrpc Write failed: %v", err)
	}

	// --- Inspect JSON API body ---
	var jpayload map[string]any
	if err := json.Unmarshal([]byte(jsonBody), &jpayload); err != nil {
		t.Fatalf("failed to decode odoojson body: %v; raw=%s", err, jsonBody)
	}
	jids, ok := jpayload["ids"].([]any)
	if !ok || len(jids) != 1 {
		t.Fatalf("odoojson payload missing 'ids' list: %v", jpayload)
	}
	if jids[0] != float64(recordID) {
		t.Errorf("odoojson ids: got %v, want %d", jids[0], recordID)
	}
	jvals, ok := jpayload["vals"].(map[string]any)
	if !ok || jvals["name"] != "Alice" {
		t.Fatalf("odoojson payload missing 'vals' name: %v", jpayload)
	}

	// --- Inspect JSON-RPC body ---
	var rpcmap map[string]any
	if err := json.Unmarshal([]byte(jsonrpcBody), &rpcmap); err != nil {
		t.Fatalf("failed to decode odoojrpc body: %v; raw=%s", err, jsonrpcBody)
	}
	params, ok := rpcmap["params"].(map[string]any)
	if !ok {
		t.Fatalf("odoojrpc missing params: %v", rpcmap)
	}
	parr, ok := params["args"].([]any)
	if !ok || len(parr) < 7 {
		t.Fatalf("odoojrpc params.args malformed: %v", params["args"])
	}
	// write positional args: db, uid, pwd, model, method, [id], values
	if parr[3] != model || parr[4] != "write" {
		t.Fatalf("odoojrpc model/method mismatch: got %v/%v", parr[3], parr[4])
	}
	pIDs, ok := parr[5].([]any)
	if !ok || len(pIDs) != 1 || pIDs[0] != float64(recordID) {
		t.Fatalf("odoojrpc write ids malformed: %v", parr[5])
	}
	pVals, ok := parr[6].(map[string]any)
	if !ok || pVals["name"] != "Alice" {
		t.Fatalf("odoojrpc write values malformed: %v", parr[6])
	}

	// --- Inspect XML-RPC body ---
	if !strings.Contains(xmlBody, "<methodName>execute_kw</methodName>") {
		t.Fatalf("xmlrpc body missing execute_kw method name: %s", xmlBody)
	}
	if !strings.Contains(xmlBody, "<int>42</int>") {
		t.Fatalf("xmlrpc body missing record id 42: %s", xmlBody)
	}
	if !strings.Contains(xmlBody, "<name>name</name>") {
		t.Fatalf("xmlrpc body missing direct values member 'name': %s", xmlBody)
	}
	if strings.Contains(xmlBody, "<name>vals</name>") {
		t.Fatalf("xmlrpc values must not be wrapped in a 'vals' member: %s", xmlBody)
	}
}
