package odoojson

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// maxResponseBytes is the maximum number of bytes read from an API response
// body. It prevents a misbehaving or malicious server from exhausting memory.
const maxResponseBytes = 32 << 20 // 32 MiB

// ----------------------------------------------------------------------------
// Request and Response
// ----------------------------------------------------------------------------

func (o *OdooJSON) Call(ctx context.Context, model string, method string, payload map[string]any) (any, error) {
	if o.url == "" {
		if err := o.genURL(); err != nil {
			return nil, fmt.Errorf("genURL failed: %w", err)
		}
	}

	// Request payload structure
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	// Create the HTTP request
	endpoint, err := o.endpointURL(model, method)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint,
		bytes.NewBuffer(body),
	)
	if err != nil {
		return nil, err
	}
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if o.apikey != "" {
		req.Header.Set("Authorization", "Bearer "+o.apikey)
	}
	if o.database != "" {
		req.Header.Set("X-Odoo-Database", o.database)
	}

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Handle non-2xx responses before attempting to decode the body.
	// Non-2xx bodies may not be JSON (e.g. HTML from a reverse proxy).
	if resp.StatusCode >= 400 {
		var data any
		var argument string
		if decErr := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(&data); decErr == nil {
			if responseMap, ok := data.(map[string]any); ok {
				if args, ok := responseMap["arguments"].([]any); ok && len(args) > 0 {
					argument = fmt.Sprint(args[0])
				}
			}
		}
		return nil, fmt.Errorf("request failed with status %d: %s %v", resp.StatusCode, resp.Status, argument)
	}

	// Decode the response
	var data any
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(&data); err != nil {
		return nil, err
	}

	return data, nil
}
