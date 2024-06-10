package odoojrpc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
)

// func filterList(filters ...[]any) []any {
// 	filter := []any{}
// 	for _, f := range filters {
// 		filter = append(filter, f)
// 	}
// 	return filter
// }

// ----------------------------------------------------------------------------
// Request and Response
// ----------------------------------------------------------------------------

type params struct {
	Service string `json:"service"`
	Method  string `json:"method"`
	Args    any    `json:"args"`
}

// clientRequest represents a JSON-RPC request sent by a client.
type clientRequest struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	ID      uint64 `json:"id"`
	Params  params `json:"params"`
}

// clientResponse represents a JSON-RPC response returned to a client.
type clientResponse struct {
	Result *json.RawMessage `json:"result"`
	Error  any              `json:"error"`
	ID     uint64           `json:"id"`
}

// EncodeClientRequest encodes parameters for a JSON-RPC client request.
func encodeClientRequest(service, method string, args any) ([]byte, error) {
	req := &clientRequest{
		JSONRPC: "2.0",
		Method:  "call",
		ID:      rand.Uint64(),
		Params:  params{Service: service, Method: method, Args: args},
	}

	rr, _ := json.Marshal(req)
	fmt.Println(string(rr))

	return json.Marshal(req)
}

// decodeClientResponse decodes the response body of a client request into
// the interface reply.
func decodeClientResponse(r io.Reader, reply any) error {
	var c clientResponse
	if err := json.NewDecoder(r).Decode(&c); err != nil {
		return err
	}
	if c.Error != nil {
		return fmt.Errorf("%v", c.Error)
	}
	if c.Result == nil {
		return errors.New("result is null")
	}
	return json.Unmarshal(*c.Result, reply)
}

func (o *OdooJSON) Call(service string, method string, args ...any) (res any, err error) {
	req, err := encodeClientRequest(service, method, args)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(o.url, "application/json", bytes.NewBuffer(req))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := decodeClientResponse(resp.Body, &res); err != nil {
		return nil, err
	}
	return res, nil
}

// Call sends a request
// func (o *Odoo) Call(service string, method string, args ...any) (res any, err error) {
// 	params := map[string]any{
// 		"service": service,
// 		"method":  method,
// 		"args":    args,
// 	}
// 	res, err = o.JSONRPC(params)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return res, nil
// }

// JSONRPC json request
// func (o *Odoo) JSONRPC(params map[string]any) (res any, err error) {
// 	message := map[string]any{
// 		"jsonrpc": "2.0",
// 		"method":  "call",
// 		"id":      rand.Intn(100000000),
// 		"params":  params,
// 	}

// 	bytesRepresentation, err := json.Marshal(message)
// 	if err != nil {
// 		return nil, fmt.Errorf("json marshall error: %w", err)
// 	}
// 	fmt.Println(string(bytesRepresentation))

// 	// TODO: refactor timeout
// 	client := &http.Client{
// 		Timeout: 5 * time.Second,
// 	}

// 	// TODO: refactor insecure skip verify
// 	if o.schema == "https" && (o.hostname == "localhost" || strings.HasSuffix(o.hostname, ".local")) {
// 		transCfg := &http.Transport{}
// 		transCfg.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
// 		client.Transport = transCfg
// 	}

// 	resp, err := client.Post(o.url, "application/json", bytes.NewBuffer(bytesRepresentation))
// 	if err != nil {
// 		return nil, fmt.Errorf("http post error: %w", err)
// 	}

// 	var result map[string]any
// 	if resp != nil {
// 		json.NewDecoder(resp.Body).Decode(&result)
// 	} else {
// 		return nil, fmt.Errorf("no response returned")
// 	}

// 	if _, ok := result["error"]; ok {
// 		resultError := ""
// 		if errorMessage, ok := result["error"].(map[string]any)["message"].(string); ok {
// 			resultError += errorMessage
// 		}
// 		if dataMessage, ok := result["error"].(map[string]any)["data"].(map[string]any)["message"].(string); ok {
// 			resultError += ": " + dataMessage
// 		}
// 		return nil, fmt.Errorf(resultError)
// 	}

// 	res = result["result"]
// 	return res, nil
// }
