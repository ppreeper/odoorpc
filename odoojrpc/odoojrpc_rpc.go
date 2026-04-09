package odoojrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
)

// maxResponseBytes is the maximum number of bytes read from a JSON-RPC response
// body. It prevents a misbehaving or malicious server from exhausting memory.
const maxResponseBytes = 32 << 20 // 32 MiB

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
	Error  *rpcError        `json:"error"`
	ID     uint64           `json:"id"`
}

// rpcError is the structured error object returned by the Odoo JSON-RPC server.
type rpcError struct {
	Code    int            `json:"code"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data"`
}

func (e *rpcError) Error() string {
	if e == nil {
		return ""
	}
	if msg, ok := e.Data["message"].(string); ok && msg != "" {
		return fmt.Sprintf("rpc error %d: %s: %s", e.Code, e.Message, msg)
	}
	return fmt.Sprintf("rpc error %d: %s", e.Code, e.Message)
}

// EncodeClientRequest encodes parameters for a JSON-RPC client request.
func encodeClientRequest(service, method string, args any) ([]byte, error) {
	// Use a non-cryptographic PRNG for JSON-RPC request IDs. The ID is only
	// used for matching requests and responses and does not require
	// cryptographic unpredictability. math/rand is fast and sufficient for
	// this purpose; callers that need different seeding can call
	// rand.Seed(...) if deterministic IDs are undesirable.
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
		return c.Error
	}
	if c.Result == nil {
		return errors.New("result is null")
	}
	return json.Unmarshal(*c.Result, reply)
}

func (o *OdooJSON) Call(ctx context.Context, service string, method string, args ...any) (res any, err error) {
	if o.url == "" {
		if err = o.genURL(); err != nil {
			return nil, fmt.Errorf("genURL failed: %w", err)
		}
	}

	req, err := encodeClientRequest(service, method, args)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.url, bytes.NewBuffer(req))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := decodeClientResponse(io.LimitReader(resp.Body, maxResponseBytes), &res); err != nil {
		return nil, err
	}
	return res, nil
}
