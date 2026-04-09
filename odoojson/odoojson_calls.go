package odoojson

import (
	"context"
	"fmt"

	"github.com/ppreeper/odoosearchdomain"
)

// Login validates the connection configuration and initialises the HTTP client.
// Unlike the JSON-RPC and XML-RPC transports, the Odoo JSON API v2 uses a
// per-request Bearer token rather than a session UID, so no network handshake
// is performed here. Configuration errors (missing API key, invalid schema,
// port, or hostname) are surfaced at this point rather than on the first call.
func (o *OdooJSON) Login(ctx context.Context) error {
	return o.genURL()
}

// Create
// Create a single record for the model and return its id
// model: model name
// values: list of field values
// Example:
//
//	values = {
//		"name": "ZExample1",
//		"email": "zexample1@example.com",
//	}
func (o *OdooJSON) Create(ctx context.Context, model string, values map[string]any) (row int, err error) {
	rowID, err := o.Call(ctx, model, "create", map[string]any{
		"vals_list": []map[string]any{values},
	})
	if err != nil {
		return -1, fmt.Errorf("create failed: %w", err)
	}

	switch v := rowID.(type) {
	case []any:
		if len(v) >= 1 {
			id, ok := v[0].(float64)
			if !ok {
				return -1, fmt.Errorf("create failed: unexpected id type in response")
			}
			row = int(id)
		}
	default:
		return -1, fmt.Errorf("create failed: unexpected response type")
	}

	return row, nil
}

// Load
// Create multiple records using a datamatrix and return their ids
// model: model name
// header: list of field names
// values: list of lists of field values
//
// Example:
//
// header = ["name", "email"]
// values = [
//
//	["ZExample1", "zexample1@example.com"],
//	["ZExample2", "zexample1@example.com"],
//	]
func (o *OdooJSON) Load(ctx context.Context, model string, header []string, values [][]any) (ids []int, err error) {
	data, err := o.Call(ctx, model, "load", map[string]any{
		"fields": header,
		"data":   values,
	})
	if err != nil {
		return nil, fmt.Errorf("load failed: %w", err)
	}

	responseMap, ok := data.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("load failed: unexpected response type")
	}
	idmap, ok := responseMap["ids"].([]any)
	if !ok {
		return nil, fmt.Errorf("load failed: ids not found in response")
	}
	for _, id := range idmap {
		v, ok := id.(float64)
		if !ok {
			return nil, fmt.Errorf("load failed: unexpected id type in response")
		}
		ids = append(ids, int(v))
	}
	return ids, nil
}

// Count record
// Returns the number of record in the current model
// matching the :domain with a maximum of :limit records
// model: model name
// domain: list of search criteria following the modified odoo domain syntax
// limit: maximum number of records to return
// Example:
//
// filters = [[["name", "=", "ZExample1"]]]
// limit = 1
func (o *OdooJSON) Count(ctx context.Context, model string, filters ...any) (count int, err error) {
	searchCount, err := o.Call(ctx, model, "search_count", map[string]any{
		"domain": odoosearchdomain.DomainList(filters...),
		"limit":  0,
	})
	if err != nil {
		return -1, fmt.Errorf("count failed: %w", err)
	}
	switch v := searchCount.(type) {
	case float64:
		count = int(v)
	default:
		return -1, fmt.Errorf("count failed: unexpected response type")
	}
	return count, nil
}

// FieldsGet record
// Returns the definition of the fields of the model
// model: model name
// attributes: list of field attributes to return
// Example:
// attributes = ["string", "help", "type"]
func (o *OdooJSON) FieldsGet(ctx context.Context, model string, fields []string, fieldAttributes ...string) (recordFields map[string]any, err error) {
	payload := map[string]any{
		"allfields": fields,
	}
	if len(fieldAttributes) > 0 {
		payload["attributes"] = fieldAttributes
	}
	fieldSet, err := o.Call(ctx, model, "fields_get", payload)
	if err != nil {
		return nil, fmt.Errorf("fields_get failed: %w", err)
	}
	recordFields, ok := fieldSet.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("fields_get failed: unexpected response type")
	}
	return recordFields, nil
}

// GetID record
// Return id of the first record matching the query, or -1 if none is found
// model: model name
// domain: list of search criteria following the modified odoo domain syntax
// Example:
// domain = [[["name", "=", "ZExample1"]]]
func (o *OdooJSON) GetID(ctx context.Context, model string, filters ...any) (id int, err error) {
	data, err := o.Call(ctx, model, "search", map[string]any{
		"domain": odoosearchdomain.DomainList(filters...),
	})
	if err != nil {
		return -1, fmt.Errorf("get_id failed: %w", err)
	}

	ids, ok := data.([]any)
	if !ok || len(ids) == 0 {
		return -1, nil
	}
	v, ok := ids[0].(float64)
	if !ok {
		return -1, fmt.Errorf("get_id failed: unexpected id type in response")
	}
	return int(v), nil
}

// GetIDWithOrder is like GetID but allows an explicit order string to be
// supplied. The order is only included in the request when non-empty.
func (o *OdooJSON) GetIDWithOrder(ctx context.Context, model string, order string, filters ...any) (id int, err error) {
	payload := map[string]any{
		"domain": odoosearchdomain.DomainList(filters...),
	}
	if order != "" {
		payload["order"] = order
	}
	data, err := o.Call(ctx, model, "search", payload)
	if err != nil {
		return -1, fmt.Errorf("get_id failed: %w", err)
	}

	ids, ok := data.([]any)
	if !ok || len(ids) == 0 {
		return -1, nil
	}
	v, ok := ids[0].(float64)
	if !ok {
		return -1, fmt.Errorf("get_id failed: unexpected id type in response")
	}
	return int(v), nil
}

// Search record
// Return ids of records matching the query
// model: model name
// domain: list of search criteria following the modified odoo domain syntax
// Example:
// domain = [[["name", "=", "ZExample1"]]]
func (o *OdooJSON) Search(ctx context.Context, model string, filters ...any) (ids []int, err error) {
	data, err := o.Call(ctx, model, "search", map[string]any{
		"domain": odoosearchdomain.DomainList(filters...),
	})
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	raw, ok := data.([]any)
	if !ok {
		return nil, fmt.Errorf("search failed: unexpected response type")
	}
	for _, item := range raw {
		v, ok := item.(float64)
		if !ok {
			return nil, fmt.Errorf("search failed: unexpected id type in response")
		}
		ids = append(ids, int(v))
	}
	return ids, nil
}

// SearchWithOrder is like Search but accepts an explicit order string. If the
// order is empty the request omits the order key.
func (o *OdooJSON) SearchWithOrder(ctx context.Context, model string, order string, filters ...any) (ids []int, err error) {
	payload := map[string]any{
		"domain": odoosearchdomain.DomainList(filters...),
	}
	if order != "" {
		payload["order"] = order
	}
	data, err := o.Call(ctx, model, "search", payload)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	raw, ok := data.([]any)
	if !ok {
		return nil, fmt.Errorf("search failed: unexpected response type")
	}
	for _, item := range raw {
		v, ok := item.(float64)
		if !ok {
			return nil, fmt.Errorf("search failed: unexpected id type in response")
		}
		ids = append(ids, int(v))
	}
	return ids, nil
}

// Read record
// Read the requested fields of the records with the given ids
// model: model name
// ids: list of record ids
// fields: list of field names
// Example:
// ids = [1, 2, 3]
// fields = ["name", "email"]
func (o *OdooJSON) Read(ctx context.Context, model string, ids []int, fields ...string) (records []map[string]any, err error) {
	data, err := o.Call(ctx, model, "read", map[string]any{
		"ids":    ids,
		"fields": fields,
		"load":   "_classic_read",
	})
	if err != nil {
		return nil, fmt.Errorf("read failed: %w", err)
	}

	raw, ok := data.([]any)
	if !ok {
		return nil, fmt.Errorf("read failed: unexpected response type")
	}
	for _, item := range raw {
		v, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("read failed: unexpected record type in response")
		}
		records = append(records, v)
	}
	return records, nil
}

// SearchRead records
// Return the requested fields of the records matching the query
// model: model name
// domain: list of search criteria following the modified odoo domain syntax
// offset: number of records to skip
// limit: maximum number of records to return
// fields: list of field names
// Example:
// domain = [[["name", "=", "ZExample1"]]]
// offset = 0
// limit = 1
// fields = ["name", "email"]
func (o *OdooJSON) SearchRead(ctx context.Context, model string, offset int, limit int, fields []string, filters ...any) (records []map[string]any, err error) {
	data, err := o.Call(ctx, model, "search_read", map[string]any{
		"domain": odoosearchdomain.DomainList(filters...),
		"offset": offset,
		"limit":  limit,
		"fields": fields,
	})
	if err != nil {
		return nil, fmt.Errorf("search_read failed: %w", err)
	}

	raw, ok := data.([]any)
	if !ok {
		return nil, fmt.Errorf("search_read failed: unexpected response type")
	}
	for _, item := range raw {
		v, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("search_read failed: unexpected record type in response")
		}
		records = append(records, v)
	}
	return records, nil
}

// SearchReadWithOrder is like SearchRead but accepts an explicit order string.
// If order is empty the request omits the order key.
func (o *OdooJSON) SearchReadWithOrder(ctx context.Context, model string, offset int, limit int, fields []string, order string, filters ...any) (records []map[string]any, err error) {
	payload := map[string]any{
		"domain": odoosearchdomain.DomainList(filters...),
		"offset": offset,
		"limit":  limit,
		"fields": fields,
	}
	if order != "" {
		payload["order"] = order
	}
	data, err := o.Call(ctx, model, "search_read", payload)
	if err != nil {
		return nil, fmt.Errorf("search_read failed: %w", err)
	}

	raw, ok := data.([]any)
	if !ok {
		return nil, fmt.Errorf("search_read failed: unexpected response type")
	}
	for _, item := range raw {
		v, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("search_read failed: unexpected record type in response")
		}
		records = append(records, v)
	}
	return records, nil
}

// Write record
// Update all the fields of the records with the given ids with the provided values
// model: model name
// id: record id
// values: list of field values
// Example:
// id = 1
//
//	values = {
//		"name": "ZExample1",
//		"email": "zexample1_1@example.com",
//	}
func (o *OdooJSON) Write(ctx context.Context, model string, recordID int, values map[string]any) (result bool, err error) {
	res, err := o.Call(ctx, model, "write", map[string]any{
		"ids":  []int{recordID},
		"vals": values,
	})
	if err != nil {
		return false, fmt.Errorf("write failed: %w", err)
	}
	result, _ = res.(bool)
	return result, nil
}

// Unlink record
// Delete the records with the given ids
// model: model name
// ids: list of record ids
// Example:
// ids = [1, 2, 3]
func (o *OdooJSON) Unlink(ctx context.Context, model string, recordIDs []int) (result bool, err error) {
	res, err := o.Call(ctx, model, "unlink", map[string]any{
		"ids": recordIDs,
	})
	if err != nil {
		return false, fmt.Errorf("unlink failed: %w", err)
	}
	result, _ = res.(bool)
	return result, nil
}

// Execute calls the given method on model, passing args as a positional argument
// list in the JSON API v2 payload. A nil or empty args slice sends an empty list.
func (o *OdooJSON) Execute(ctx context.Context, model string, method string, args []any) (result bool, err error) {
	if args == nil {
		args = []any{}
	}
	payload := map[string]any{"args": args}
	_, err = o.Call(ctx, model, method, payload)
	if err != nil {
		return false, fmt.Errorf("execute failed: %w", err)
	}
	return true, nil
}

// ExecuteKw calls the given method on model, merging args and all keys from the
// first kwargs map into the JSON API v2 payload. A nil kwargs slice is valid and
// results in a payload that contains only the "args" key.
func (o *OdooJSON) ExecuteKw(ctx context.Context, model string, method string, args []any, kwargs []map[string]any) (result bool, err error) {
	if args == nil {
		args = []any{}
	}
	payload := map[string]any{"args": args}
	if len(kwargs) > 0 {
		for k, v := range kwargs[0] {
			payload[k] = v
		}
	}
	res, err := o.Call(ctx, model, method, payload)
	if err != nil {
		return false, fmt.Errorf("execute_kw failed: %w", err)
	}
	result, _ = res.(bool)
	return result, nil
}

// Action
// stub for Action
func (o *OdooJSON) Action(ctx context.Context, model string, action string, params map[string]any) (result bool, err error) {
	res, err := o.Call(ctx, model, action, params)
	if err != nil {
		return false, fmt.Errorf("action failed: %w", err)
	}
	result, _ = res.(bool)
	return result, nil
}
