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

// TestCrossTransportExecuteKwConsistency calls ExecuteKw on all three transports
// and asserts that the outgoing requests contain the same logical pieces:
// model, method, positional args and kwargs (ids/vals/name).
func TestCrossTransportExecuteKwConsistency(t *testing.T) {
	t.Parallel()

	model := "res.partner"
	method := "write"
	args := []any{1, 2}
	kwargs := []map[string]any{{"ids": []int{42}, "vals": map[string]any{"name": "Alice"}}}

	var jsonBody, jsonrpcBody, xmlBody string

	// JSON API server (odoojson)
	tsJSON := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		jsonBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `true`)
	}))
	defer tsJSON.Close()

	// JSON-RPC server (odoojrpc)
	tsJSONRPC := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		jsonrpcBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":true}`)
	}))
	defer tsJSONRPC.Close()

	// XML-RPC server (odooxmlrpc)
	tsXML := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		xmlBody = string(b)
		w.Header().Set("Content-Type", "text/xml")
		// Respond to authenticate requests with an int uid, otherwise true.
		if strings.Contains(xmlBody, "authenticate") || strings.Contains(xmlBody, "<methodName>authenticate</methodName>") {
			fmt.Fprint(w, `<?xml version="1.0"?><methodResponse><params><param><value><int>1</int></value></param></params></methodResponse>`)
			return
		}
		fmt.Fprint(w, `<?xml version="1.0"?><methodResponse><params><param><value><boolean>1</boolean></value></param></params></methodResponse>`)
	}))
	defer tsXML.Close()

	// Helper to split host/port from ts.URL
	parseHostPort := func(raw string) (host string, port int) {
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

	// --- odoojson client ---
	host, port := parseHostPort(tsJSON.URL)
	ojson := odoojson.NewOdoo().WithHostname(host).WithPort(port).WithDatabase("testdb").WithAPIKey("testkey")
	if _, err := ojson.ExecuteKw(context.Background(), model, method, args, kwargs); err != nil {
		t.Fatalf("odoojson ExecuteKw failed: %v", err)
	}

	// --- odoojrpc client ---
	host, port = parseHostPort(tsJSONRPC.URL)
	orpc := odoojrpc.NewOdoo().WithHostname(host).WithPort(port).WithDatabase("testdb").WithUsername("admin").WithPassword("secret")
	if _, err := orpc.ExecuteKw(context.Background(), model, method, args, kwargs); err != nil {
		t.Fatalf("odoojrpc ExecuteKw failed: %v", err)
	}

	// --- odooxmlrpc client ---
	host, port = parseHostPort(tsXML.URL)
	oxml := odooxmlrpc.NewOdoo().WithHostname(host).WithPort(port).WithDatabase("testdb").WithUsername("admin").WithPassword("secret")
	if err := oxml.Login(context.Background()); err != nil {
		t.Fatalf("odooxmlrpc login failed: %v", err)
	}
	if _, err := oxml.ExecuteKw(context.Background(), model, method, args, kwargs); err != nil {
		t.Fatalf("odooxmlrpc ExecuteKw failed: %v", err)
	}

	// --- Inspect JSON API body ---
	var jpayload map[string]any
	if err := json.Unmarshal([]byte(jsonBody), &jpayload); err != nil {
		t.Fatalf("failed to decode odoojson body: %v; raw=%s", err, jsonBody)
	}
	if _, ok := jpayload["args"]; !ok {
		t.Fatalf("odoojson payload missing 'args': %v", jpayload)
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
	// positional args list should be at index 5
	pArgs, ok := parr[5].([]any)
	if !ok || len(pArgs) != 2 {
		t.Fatalf("odoojrpc positional args malformed: %v", parr[5])
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
