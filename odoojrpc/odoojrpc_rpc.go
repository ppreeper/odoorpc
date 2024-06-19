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
