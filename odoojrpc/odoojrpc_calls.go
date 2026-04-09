package odoojrpc

import (
	"context"
	"fmt"

	"github.com/ppreeper/odoosearchdomain"
)

// Login
// Login to the server and return the uid
func (o *OdooJSON) Login(ctx context.Context) (err error) {
	if o.url == "" {
		if err = o.genURL(); err != nil {
			return fmt.Errorf("genURL failed in login: %w", err)
		}
	}
	// Logging in
	v, err := o.Call(ctx, "common", "login", o.database, o.username, o.password)
	if err != nil {
		return fmt.Errorf("login error: %w", err)
	}
	switch v2 := v.(type) {
	case float64:
		if v2 == 0 {
			return fmt.Errorf("login failed: invalid credentials")
		}
		o.uid = int(v2)
	default:
		return fmt.Errorf("login failed: unexpected response type %T (credentials may be invalid)", v)
	}

	return nil
}

// Create record
// Create a single record for the model and return its id
// model: model name
// values: list of field values
// Example:
//
//	values = {
//		"name": "ZExample1",
//		"email": "zexample1@   example.com",
//	}
func (o *OdooJSON) Create(ctx context.Context, model string, values map[string]any) (row int, err error) {
	v, err := o.Call(ctx, "object", "execute",
		o.database, o.uid, o.password,
		model, "create", values,
	)
	if err != nil {
		return -1, err
	}
	switch v := v.(type) {
	case float64:
		row = int(v)
	default:
		row = -1
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
	results, err := o.Call(ctx, "object", "execute",
		o.database, o.uid, o.password,
		model, "load", header, values,
	)
	if err != nil {
		return nil, err
	}
	switch restype := results.(type) {
	case map[string]any:
		idmap, ok := restype["ids"].([]any)
		if !ok {
			return nil, fmt.Errorf("ids not found in response")
		}
		for _, id := range idmap {
			v, ok := id.(float64)
			if !ok {
				return nil, fmt.Errorf("load failed: unexpected id type in response")
			}
			ids = append(ids, int(v))
		}
	case float64:
		ids = []int{int(restype)}
	default:
		return nil, fmt.Errorf("load failed: unexpected response type %T", results)
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
// domain = [[["name", "=", "ZExample1"]]]
// limit = 1
func (o *OdooJSON) Count(ctx context.Context, model string, domains ...any) (count int, err error) {
	v, err := o.Call(ctx, "object", "execute",
		o.database, o.uid, o.password,
		model, "search_count", odoosearchdomain.DomainList(domains...),
	)
	if err != nil {
		return count, err
	}
	switch v := v.(type) {
	case float64:
		count = int(v)
	default:
		return -1, fmt.Errorf("count: unexpected response type %T", v)
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
	var v any
	if len(fieldAttributes) == 0 {
		v, err = o.Call(ctx, "object", "execute",
			o.database, o.uid, o.password,
			model, "fields_get", odoosearchdomain.DomainString(fields...),
		)
		if err != nil {
			return recordFields, fmt.Errorf("fields_get failed: %w", err)
		}
	} else {
		v, err = o.Call(ctx, "object", "execute",
			o.database, o.uid, o.password,
			model, "fields_get", odoosearchdomain.DomainString(fields...),
			odoosearchdomain.DomainString(fieldAttributes...),
		)
		if err != nil {
			return recordFields, fmt.Errorf("fields_get failed: %w", err)
		}
	}
	recordFields, ok := v.(map[string]any)
	if !ok {
		return recordFields, fmt.Errorf("fields_get failed: unexpected response type %T", v)
	}
	return recordFields, nil
}

// GetID record
// Return id of the first record matching the query, or -1 if none is found
// model: model name
// domain: list of search criteria following the modified odoo domain syntax
// Example:
// domain = [[["name", "=", "ZExample1"]]]
func (o *OdooJSON) GetID(ctx context.Context, model string, domains ...any) (id int, err error) {
	v, err := o.Call(ctx, "object", "execute",
		o.database, o.uid, o.password,
		model, "search", odoosearchdomain.DomainList(domains...),
	)
	if err != nil {
		return -1, err
	}
	items, ok := v.([]any)
	if !ok || len(items) == 0 {
		return -1, nil
	}
	f, ok := items[0].(float64)
	if !ok {
		return -1, fmt.Errorf("get_id: unexpected id type %T in response", items[0])
	}
	return int(f), nil
}

// Search record
// Return ids of records matching the query
// model: model name
// domain: list of search criteria following the modified odoo domain syntax
// Example:
// domain = [[["name", "=", "ZExample1"]]]
func (o *OdooJSON) Search(ctx context.Context, model string, domains ...any) (ids []int, err error) {
	v, err := o.Call(ctx, "object", "execute",
		o.database, o.uid, o.password,
		model, "search", odoosearchdomain.DomainList(domains...),
	)
	if err != nil {
		return ids, err
	}
	switch v := v.(type) {
	case []any:
		for _, item := range v {
			f, ok := item.(float64)
			if !ok {
				return ids, fmt.Errorf("search: unexpected id type %T in response", item)
			}
			ids = append(ids, int(f))
		}
	case []float64:
		for _, f := range v {
			ids = append(ids, int(f))
		}
	default:
		return ids, fmt.Errorf("search: unexpected response type %T", v)
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
	v, err := o.Call(ctx, "object", "execute",
		o.database, o.uid, o.password,
		model, "read", ids, odoosearchdomain.DomainString(fields...),
	)
	if err != nil {
		return records, err
	}
	switch v := v.(type) {
	case []any:
		for _, item := range v {
			rec, ok := item.(map[string]any)
			if !ok {
				return records, fmt.Errorf("read: unexpected record type %T in response", item)
			}
			records = append(records, rec)
		}
	case []map[string]any:
		records = append(records, v...)
	default:
		return records, fmt.Errorf("read: unexpected response type %T", v)
	}
	return records, nil
}

// SearchRead records
// Return the requested fields of the records matching the query
// model: model name
// offset: number of records to skip
// limit: maximum number of records to return
// fields: list of field names
// domains: list of search criteria following the modified odoo domain syntax
// Example:
// offset = 0
// limit = 1
// fields = ["name", "email"]
// domains = [["name", "=", "ZExample1"]]
func (o *OdooJSON) SearchRead(ctx context.Context, model string, offset int, limit int, fields []string, domains ...any) (records []map[string]any, err error) {
	vv, err := o.Call(ctx, "object", "execute",
		o.database, o.uid, o.password,
		model, "search_read", odoosearchdomain.DomainList(domains...), fields, offset, limit,
	)
	if err != nil {
		return records, err
	}

	switch vv := vv.(type) {
	case []any:
		for _, item := range vv {
			rec, ok := item.(map[string]any)
			if !ok {
				return records, fmt.Errorf("search_read: unexpected record type %T in response", item)
			}
			records = append(records, rec)
		}
	case []map[string]any:
		records = append(records, vv...)
	default:
		return records, fmt.Errorf("search_read: unexpected response type %T", vv)
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
	v, err := o.Call(ctx, "object", "execute",
		o.database, o.uid, o.password,
		model, "write", recordID, values,
	)
	if err != nil {
		return false, err
	}
	switch v := v.(type) {
	case bool:
		result = v
	default:
		result = false
	}
	return result, nil
}

// Unlink record
// Delete the records with the given ids
// model: model name
// ids: list of record ids
// Example:
// ids = [1, 2, 3]
func (o *OdooJSON) Unlink(ctx context.Context, model string, recordIDs []int) (result bool, err error) {
	v, err := o.Call(ctx, "object", "execute",
		o.database, o.uid, o.password,
		model, "unlink", recordIDs,
	)
	if err != nil {
		return result, err
	}
	switch v := v.(type) {
	case bool:
		result = v
	default:
		result = false
	}
	return result, nil
}

// Execute
// call a method of the model
// model: model name
// method: method name
// args: list of arguments
// Example:
// model = "res.partner"
// method = "search_read"
func (o *OdooJSON) Execute(ctx context.Context, model string, method string, args []any) (result bool, err error) {
	v, err := o.Call(ctx, "object", "execute_kw",
		o.database, o.uid, o.password,
		model, method, args,
	)
	if err != nil {
		return result, err
	}
	switch v := v.(type) {
	case bool:
		result = v
	default:
		result = false
	}
	return result, nil
}

// ExecuteKw
// call a method of the model
// model: model name
// method: method name
// args: list of arguments
// kwargs: dictionary of keyword arguments
// Example:
// model = "res.partner"
// method = "search_read"
func (o *OdooJSON) ExecuteKw(ctx context.Context, model string, method string, args []any, kwargs []map[string]any) (result bool, err error) {
	// XML-RPC/JSON-RPC execute_kw expects the args and kwargs to be passed as
	// two separate positional parameters: the list of positional args followed
	// by the kwargs map. When our caller provides a []map[string]any for
	// kwargs we forward the first map if present, otherwise pass an empty
	// map to match Odoo's expectations.
	var kw any
	if len(kwargs) > 0 && kwargs[0] != nil {
		kw = kwargs[0]
	} else {
		kw = map[string]any{}
	}
	v, err := o.Call(ctx, "object", "execute_kw",
		o.database, o.uid, o.password,
		model, method, args, kw,
	)
	if err != nil {
		return result, err
	}
	switch v := v.(type) {
	case bool:
		result = v
	default:
		result = false
	}
	return result, nil
}
