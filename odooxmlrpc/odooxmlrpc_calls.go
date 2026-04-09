package odooxmlrpc

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ppreeper/odoorpc/xmlrpc"
	"github.com/ppreeper/odoosearchdomain"
)

// Login
// Login to the server and return the uid
func (o *OdooXML) Login(ctx context.Context) (err error) {
	if o.url == "" {
		if err = o.genURL(); err != nil {
			return fmt.Errorf("genURL failed in login: %w", err)
		}
	}
	// rpc clients — use a transport with the configured timeout so hung
	// servers cannot block indefinitely.
	transport := &http.Transport{ResponseHeaderTimeout: o.timeout}
	o.common, err = xmlrpc.NewClient(o.url+"common", transport)
	if err != nil {
		return fmt.Errorf("failed to create common client: %w", err)
	}
	o.models, err = xmlrpc.NewClient(o.url+"object", transport)
	if err != nil {
		return fmt.Errorf("failed to create models client: %w", err)
	}

	// Logging in
	if err := o.common.CallContext(ctx, "authenticate", []any{
		o.database, o.username, o.password,
		map[string]any{},
	}, &o.uid); err != nil {
		return fmt.Errorf("login failed: %w", err)
	}
	if o.uid == 0 {
		return fmt.Errorf("login failed: invalid credentials")
	}
	return nil
}

// Create
// Create a single record for the model and return its id
// model: model name
// values: list of field values
// Example:
//
//	values = {
//		"name": "ZExample1",
//		"email": "zexample1@   example.com",
//	}
func (o *OdooXML) Create(ctx context.Context, model string, values map[string]any) (row int, err error) {
	if err := o.models.CallContext(ctx, "execute_kw", []any{
		o.database, o.uid, o.password,
		model, "create",
		[]any{values},
	}, &row); err != nil {
		return -1, fmt.Errorf("create failed: %w", err)
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
func (o *OdooXML) Load(ctx context.Context, model string, header []string, values [][]any) (ids []int, err error) {
	var results any
	// Use execute_kw with the method args provided as a single positional
	// argument (a list) containing header and values. This matches the
	// execute_kw signature: execute_kw(db, uid, pwd, model, method, args, kwargs).
	err = o.models.CallContext(ctx, "execute_kw", []any{
		o.database, o.uid, o.password,
		model, "load",
		[]any{header, values},
	}, &results)
	if err != nil {
		return nil, fmt.Errorf("load failed: %w", err)
	}
	switch restype := results.(type) {
	case map[string]any:
		idmap, ok := restype["ids"].([]any)
		if !ok {
			return nil, fmt.Errorf("load failed: ids not found in response")
		}
		for _, id := range idmap {
			v, ok := id.(int64)
			if !ok {
				return nil, fmt.Errorf("load failed: unexpected id type %T in response", id)
			}
			ids = append(ids, int(v))
		}
	case int64:
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
func (o *OdooXML) Count(ctx context.Context, model string, domains ...any) (count int, err error) {
	if err := o.models.CallContext(ctx, "execute_kw", []any{
		o.database, o.uid, o.password,
		model, "search_count",
		[]any{odoosearchdomain.DomainList(domains...)},
	}, &count); err != nil {
		return -1, fmt.Errorf("count failed: %w", err)
	}
	return count, nil
}

// FieldsGet record
// Returns the definition of the fields of the model
// model: model name
// attributes: list of field attributes to return
// Example:
// attributes = ["string", "help", "type"]
func (o *OdooXML) FieldsGet(ctx context.Context, model string, fields []string, fieldAttributes ...string) (recordFields map[string]any, err error) {
	// Call fields_get using execute_kw and pass the method args as a list.
	if err := o.models.CallContext(ctx, "execute_kw", []any{
		o.database, o.uid, o.password,
		model, "fields_get",
		[]any{fields, odoosearchdomain.DomainString(fieldAttributes...)},
	}, &recordFields); err != nil {
		return nil, fmt.Errorf("fields_get failed: %w", err)
	}
	return
}

// GetID record
// Return id of the first record matching the query, or -1 if none is found
// model: model name
// domain: list of search criteria following the modified odoo domain syntax
// Example:
// domain = [[["name", "=", "ZExample1"]]]
func (o *OdooXML) GetID(ctx context.Context, model string, domains ...any) (id int, err error) {
	var ids []int
	if err := o.models.CallContext(ctx, "execute_kw", []any{
		o.database, o.uid, o.password,
		model, "search",
		odoosearchdomain.DomainList(domains...),
		map[string]any{"limit": 1},
	}, &ids); err != nil {
		return -1, fmt.Errorf("get_id failed: %w", err)
	}
	if len(ids) == 0 {
		return -1, nil
	}
	return ids[0], nil
}

// Search record
// Return ids of records matching the query
// model: model name
// domain: list of search criteria following the modified odoo domain syntax
// Example:
// domain = [[["name", "=", "ZExample1"]]]
func (o *OdooXML) Search(ctx context.Context, model string, domains ...any) (ids []int, err error) {
	if err := o.models.CallContext(ctx, "execute_kw", []any{
		o.database, o.uid, o.password,
		model, "search",
		[]any{odoosearchdomain.DomainList(domains...)},
	}, &ids); err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
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
func (o *OdooXML) Read(ctx context.Context, model string, ids []int, fields ...string) (records []map[string]any, err error) {
	if err := o.models.CallContext(ctx, "execute_kw", []any{
		o.database, o.uid, o.password,
		model, "read",
		[]any{ids, odoosearchdomain.DomainString(fields...)},
	}, &records); err != nil {
		return records, fmt.Errorf("read failed: %w", err)
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
func (o *OdooXML) SearchRead(ctx context.Context, model string, offset int, limit int, fields []string, domains ...any) (records []map[string]any, err error) {
	options := map[string]any{
		"offset": offset,
		"limit":  limit,
		"fields": odoosearchdomain.DomainString(fields...),
	}

	if err := o.models.CallContext(ctx, "execute_kw", []any{
		o.database, o.uid, o.password,
		model, "search_read",
		[]any{odoosearchdomain.DomainList(domains...), options},
	}, &records); err != nil {
		return nil, fmt.Errorf("search_read failed: %w", err)
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
func (o *OdooXML) Write(ctx context.Context, model string, recordID int, values map[string]any) (result bool, err error) {
	if err := o.models.CallContext(ctx, "execute_kw", []any{
		o.database, o.uid, o.password,
		model, "write",
		[]any{[]int{recordID}, map[string]any{"vals": values}},
	}, &result); err != nil {
		return result, fmt.Errorf("write failed: %w", err)
	}
	return result, nil
}

// Unlink record
// Delete the records with the given ids
// model: model name
// ids: list of record ids
// Example:
// ids = [1, 2, 3]
func (o *OdooXML) Unlink(ctx context.Context, model string, recordIDs []int) (result bool, err error) {
	if err := o.models.CallContext(ctx, "execute_kw", []any{
		o.database, o.uid, o.password,
		model, "unlink",
		[]any{recordIDs},
	}, &result); err != nil {
		return result, fmt.Errorf("unlink failed: %w", err)
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
func (o *OdooXML) Execute(ctx context.Context, model string, method string, args []any) (result bool, err error) {
	// execute should call execute_kw for consistency with the XML-RPC
	// transport's expectations. The args are provided as the single positional
	// argument to execute_kw.
	if err := o.models.CallContext(ctx, "execute_kw", []any{
		o.database, o.uid, o.password,
		model, method, []any{args},
	}, &result); err != nil {
		return false, fmt.Errorf("execute failed: %w", err)
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
func (o *OdooXML) ExecuteKw(ctx context.Context, model string, method string, args []any, kwargs []map[string]any) (result bool, err error) {
	var kw any
	if len(kwargs) > 0 && kwargs[0] != nil {
		kw = kwargs[0]
	} else {
		kw = map[string]any{}
	}
	if err := o.models.CallContext(ctx, "execute_kw", []any{
		o.database, o.uid, o.password,
		model, method, args, kw,
	}, &result); err != nil {
		return false, fmt.Errorf("execute_kw failed: %w", err)
	}
	return result, nil
}
