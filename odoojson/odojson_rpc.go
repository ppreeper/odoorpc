package odoojson

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
)

// ----------------------------------------------------------------------------
// Request and Response
// ----------------------------------------------------------------------------

func (o *OdooJSON) Call(model string, method string, payload map[string]any) (any, error) {
	// Request payload structure
	body, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return nil, err
	}

	// Create the HTTP request
	req, err := http.NewRequest(
		http.MethodPost,
		o.genURL(model, method),
		bytes.NewBuffer(body),
	)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return nil, err
	}
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.apikey)
	if o.database != "" {
		req.Header.Set("X-Odoo-Database", o.database)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return nil, err
	}
	defer resp.Body.Close()

	// Decode the response
	var data any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		fmt.Println("Error decoding response:", err)
		return nil, err
	}

	// Handle non-2xx responses
	if resp.StatusCode >= 400 {
		var argument string
		if reflect.TypeOf(data).Kind() == reflect.Map {
			responseMap := data.(map[string]any)
			argument = fmt.Sprintf("%s", responseMap["arguments"].([]any)[0])
		}
		return nil, fmt.Errorf("request failed with status %d: %s %v", resp.StatusCode, resp.Status, argument)
	}

	return data, nil
}
