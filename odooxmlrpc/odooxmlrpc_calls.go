package odooxmlrpc

import (
	"fmt"
	"log"

	"github.com/ppreeper/odoorpc/filter"
	"github.com/ppreeper/odoorpc/xmlrpc"
)

// Login
// Login to the server and return the uid
func (o *OdooXML) Login() (err error) {
	if o.url == "" {
		if err = o.genURL(); err != nil {
			return fmt.Errorf("genURL failed in login: %w", err)
		}
	}
	// rpc clients
	o.common, err = xmlrpc.NewClient(o.url+"common", nil)
	if err != nil {
		log.Fatal(err)
	}
	o.models, err = xmlrpc.NewClient(o.url+"object", nil)
	if err != nil {
		log.Fatal(err)
	}

	// Logging in
	if err := o.common.Call("authenticate", []any{
		o.database, o.username, o.password,
		map[string]any{},
	}, &o.uid); err != nil {
		return fmt.Errorf("login failed: %w", err)
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
func (o *OdooXML) Create(model string, values map[string]any) (row int, err error) {
	if err := o.models.Call("execute_kw", []any{
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
func (o *OdooXML) Load(model string, header []string, values [][]any) (ids []int, err error) {
	var results any
	err = o.models.Call("execute", []any{
		o.database, o.uid, o.password,
		model, "load", header, values,
	}, &results)
	if err != nil {
		return []int{}, fmt.Errorf("load failed: %w", err)
	}
	switch restype := results.(type) {
	case map[string]any:
		vmap := results.(map[string]any)
		idmap, ok := vmap["ids"].([]any)
		if !ok {
			return []int{-1}, fmt.Errorf("ids not found in response: %w", err)
		}
		for _, id := range idmap {
			ids = append(ids, int(id.(int64)))
		}
	case int64:
		ids = []int{int(restype)}
	default:
		ids = []int{-1}
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
func (o *OdooXML) Count(model string, filters ...any) (count int, err error) {
	fmt.Println("Count", model, filters)
	if err := o.models.Call("execute", []any{
		o.database, o.uid, o.password,
		model, "search_count", filter.FilterList(filters...),
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
func (o *OdooXML) FieldsGet(model string, fields []string, fieldAttributes ...string) (recordFields map[string]any, err error) {
	if err := o.models.Call("execute", []any{
		o.database, o.uid, o.password, model, "fields_get",
		fields,
		fieldAttributes,
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
func (o *OdooXML) GetID(model string, filters ...any) (id int, err error) {
	var ids []int
	if err := o.models.Call("execute_kw", []any{
		o.database, o.uid, o.password,
		model, "search",
		filters,
		map[string]any{"limit": 1},
	}, &ids); err != nil {
		return -1, fmt.Errorf("get_id failed: %w", err)
	}
	if len(ids) == 0 {
		return -1, fmt.Errorf("get_id no record found")
	}
	return int(ids[0]), nil
}

// Search record
// Return ids of records matching the query
// model: model name
// domain: list of search criteria following the modified odoo domain syntax
// Example:
// domain = [[["name", "=", "ZExample1"]]]
func (o *OdooXML) Search(model string, filters ...any) (ids []int, err error) {
	if err := o.models.Call("execute", []any{
		o.database, o.uid, o.password,
		model, "search",
		filter.FilterList(filters...),
	}, &ids); err != nil {
		return []int{}, fmt.Errorf("search failed: %w", err)
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
func (o *OdooXML) Read(model string, ids []int, fields ...string) (records []map[string]any, err error) {
	if err := o.models.Call("execute", []any{
		o.database, o.uid, o.password,
		model, "read", ids, filter.FilterString(fields...),
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
func (o *OdooXML) SearchRead(model string, offset int, limit int, fields []string, filters ...any) (records []map[string]any, err error) {
	options := make(map[string]any)
	if offset > 0 {
		options["offset"] = offset
	}
	if limit > 0 {
		options["limit"] = limit
	}
	if len(fields) > 0 {
		options["fields"] = filter.FilterString(fields...)
	}

	if err := o.models.Call("execute_kw", []any{
		o.database, o.uid, o.password,
		model, "search_read",
		filters,
		options,
	}, &records); err != nil {
		return records, err
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
func (o *OdooXML) Write(model string, recordID int, values map[string]any) (result bool, err error) {
	if err := o.models.Call("execute_kw", []any{
		o.database, o.uid, o.password,
		model, "write",
		[]any{[]int{recordID}, values},
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
func (o *OdooXML) Unlink(model string, recordIDs []int) (result bool, err error) {
	if err := o.models.Call("execute_kw", []any{
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
func (o *OdooXML) Execute(model string, method string, args []any) (result bool, err error) {
	if err := o.models.Call("execute_kw", []any{
		o.database, o.uid, o.password,
		model, method, args,
	}, &result); err != nil {
		return result, err
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
func (o *OdooXML) ExecuteKw(model string, method string, args []any, kwargs []map[string]any) (result bool, err error) {
	if err := o.models.Call("execute_kw", []any{
		o.database, o.uid, o.password,
		model, method, args, kwargs,
	}, &result); err != nil {
		return result, err
	}
	return result, nil
}
