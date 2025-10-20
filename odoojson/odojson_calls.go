package odoojson

import (
	"fmt"
	"reflect"

	"github.com/ppreeper/odoosearchdomain"
)

// Login
// Login to the server and return the uid
func (o *OdooJSON) Login() (err error) {
	if o.schema != "http" && o.schema != "https" {
		return fmt.Errorf("invalid schema: http or https: %w", err)
	}
	if o.port == 0 || o.port > 65535 {
		return fmt.Errorf("invalid port: 1-65535: %w", err)
	}
	if len(o.hostname) == 0 || len(o.hostname) > 2048 {
		return fmt.Errorf("invalid hostname length: 1-2048: %w", err)
	}
	o.url = fmt.Sprintf("%s://%s:%d/json/2/", o.schema, o.hostname, o.port)
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
func (o *OdooJSON) Create(model string, values map[string]any) (row int, err error) {
	rowID, err := o.Call(model, "create", map[string]any{
		"vals_list": []map[string]any{values},
	})
	if err != nil {
		return -1, fmt.Errorf("create failed: %w", err)
	}

	if reflect.TypeOf(rowID).Kind() == reflect.Slice {
		s := reflect.ValueOf(rowID)
		if s.Len() >= 1 {
			row = int(s.Index(0).Interface().(float64))
		}
	} else {
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
func (o *OdooJSON) Load(model string, header []string, values [][]any) (ids []int, err error) {
	data, err := o.Call(model, "load", map[string]any{
		"fields": header,
		"data":   values,
	})
	if err != nil {
		return ids, fmt.Errorf("load failed: %w", err)
	}

	if reflect.TypeOf(data).Kind() == reflect.Map {
		responseMap := data.(map[string]any)
		if reflect.TypeOf(responseMap["ids"]).Kind() == reflect.Slice {
			s := reflect.ValueOf(responseMap["ids"])
			for i := 0; i < s.Len(); i++ {
				ids = append(ids, int(s.Index(i).Interface().(float64)))
			}
		}
	} else {
		return []int{}, fmt.Errorf("load failed: %w", err)
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
func (o *OdooJSON) Count(model string, filters ...any) (count int, err error) {
	search_count, err := o.Call(model, "search_count", map[string]any{
		"domain": odoosearchdomain.DomainList(filters...),
		"limit":  0,
	})
	if err != nil {
		return -1, fmt.Errorf("count failed: %w", err)
	}
	if reflect.TypeOf(search_count).Kind() == reflect.Float64 {
		count = int(reflect.ValueOf(search_count).Interface().(float64))
	} else {
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
func (o *OdooJSON) FieldsGet(model string, fields []string, fieldAttributes ...string) (recordFields map[string]any, err error) {
	fieldSet, err := o.Call(model, "fields_get", map[string]any{
		"allfields":  fields,
		"attributes": fieldAttributes,
	})
	if err != nil {
		return recordFields, fmt.Errorf("get_id failed: %w", err)
	}
	if reflect.TypeOf(fieldSet).Kind() == reflect.Map {
		recordFields = fieldSet.(map[string]any)
	}
	return
}

// GetID record
// Return id of the first record matching the query, or -1 if none is found
// model: model name
// domain: list of search criteria following the modified odoo domain syntax
// Example:
// domain = [[["name", "=", "ZExample1"]]]
func (o *OdooJSON) GetID(model string, filters ...any) (id int, err error) {
	var ids []int
	data, err := o.Call(model, "search", map[string]any{
		"domain": odoosearchdomain.DomainList(filters...),
		"order":  "id asc",
	})
	if err != nil {
		return -1, fmt.Errorf("get_id failed: %w", err)
	}

	if reflect.TypeOf(data).Kind() == reflect.Slice {
		s := reflect.ValueOf(data)
		for i := 0; i < s.Len(); i++ {
			ids = append(ids, int(s.Index(i).Interface().(float64)))
		}
	}
	if len(ids) == 0 {
		return -1, nil
	}

	return int(ids[0]), nil
}

// Search record
// Return ids of records matching the query
// model: model name
// domain: list of search criteria following the modified odoo domain syntax
// Example:
// domain = [[["name", "=", "ZExample1"]]]
func (o *OdooJSON) Search(model string, filters ...any) (ids []int, err error) {
	data, err := o.Call(model, "search", map[string]any{
		"domain": odoosearchdomain.DomainList(filters...),
		"order":  "id asc",
	})

	if reflect.TypeOf(data).Kind() == reflect.Slice {
		s := reflect.ValueOf(data)
		for i := 0; i < s.Len(); i++ {
			ids = append(ids, int(s.Index(i).Interface().(float64)))
		}
	} else {
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
func (o *OdooJSON) Read(model string, ids []int, fields ...string) (records []map[string]any, err error) {
	data, err := o.Call(model, "read", map[string]any{
		"ids":    ids,
		"fields": fields,
		"load":   "_classic_read",
	})

	if reflect.TypeOf(data).Kind() == reflect.Slice {
		s := reflect.ValueOf(data)
		for i := 0; i < s.Len(); i++ {
			records = append(records, s.Index(i).Interface().(map[string]any))
		}
	} else {
		return []map[string]any{}, fmt.Errorf("search failed: %w", err)
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
func (o *OdooJSON) SearchRead(model string, offset int, limit int, fields []string, filters ...any) (records []map[string]any, err error) {
	data, err := o.Call(model, "search_read", map[string]any{
		"domain": odoosearchdomain.DomainList(filters...),
		"offset": offset,
		"limit":  limit,
		"fields": fields,
		"order":  "id asc",
		// "read_kwargs": map[string]any{
		// 	"load": "_classic_read",
		// },
	})

	if reflect.TypeOf(data).Kind() == reflect.Slice {
		s := reflect.ValueOf(data)
		for i := 0; i < s.Len(); i++ {
			records = append(records, s.Index(i).Interface().(map[string]any))
		}
	} else {
		return []map[string]any{}, fmt.Errorf("search failed: %w", err)
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
func (o *OdooJSON) Write(model string, recordID int, values map[string]any) (result bool, err error) {
	res, err := o.Call(model, "write", map[string]any{
		"ids":  []int{recordID},
		"vals": values,
	})
	if err != nil {
		return false, fmt.Errorf("write failed: %w", err)
	}
	if reflect.TypeOf(res).Kind() == reflect.Bool {
		result = res.(bool)
	}

	return result, nil
}

// Unlink record
// Delete the records with the given ids
// model: model name
// ids: list of record ids
// Example:
// ids = [1, 2, 3]
func (o *OdooJSON) Unlink(model string, recordIDs []int) (result bool, err error) {
	res, err := o.Call(model, "unlink", map[string]any{
		"ids": recordIDs,
	})
	if err != nil {
		return false, fmt.Errorf("unlink failed: %w", err)
	}
	if reflect.TypeOf(res).Kind() == reflect.Bool {
		result = res.(bool)
	}

	return result, err
}

// Execute
// stub for Execute
func (o *OdooJSON) Execute(model string, method string, args []any) (result bool, err error) {
	if len(args) == 0 {
		return false, fmt.Errorf("action failed: no arguments provided")
	}
	if reflect.TypeOf(args[0]).Kind() != reflect.Map {
		return false, fmt.Errorf("action failed: first argument must be a map")
	}
	_, err = o.Call(model, method, args[0].(map[string]any))
	if err != nil {
		return false, fmt.Errorf("action failed: %w", err)
	}
	return true, err
}

// ExecuteKw
// stub for ExecuteKw
func (o *OdooJSON) ExecuteKw(model string, method string, args []any, kwargs []map[string]any) (result bool, err error) {
	res, err := o.Call(model, method, kwargs[0])
	if err != nil {
		return false, fmt.Errorf("action failed: %w", err)
	}
	if reflect.TypeOf(res).Kind() == reflect.Bool {
		result = res.(bool)
	}
	return result, err
}

// Action
// stub for Action
func (o *OdooJSON) Action(model string, action string, params map[string]any) (result bool, err error) {
	res, err := o.Call(model, action, params)
	if err != nil {
		return false, fmt.Errorf("action failed: %w", err)
	}
	if reflect.TypeOf(res).Kind() == reflect.Bool {
		result = res.(bool)
	}

	return result, err
}
