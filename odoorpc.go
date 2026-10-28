// Package odoorpc defines the client contract shared by the Odoo transports
// in this module: the JSON API v2 (odoojson), JSON-RPC (odoojrpc) and XML-RPC
// (odooxmlrpc) packages.
package odoorpc

import "context"

// Odoo is the transport-independent client contract.
//
// Every method takes a context and reports failures with a non-nil error.
// Methods returning an int use -1 both for "not found" (with a nil error,
// where documented) and as the value returned alongside an error; methods
// returning a bool return false alongside an error. Callers must check err
// before interpreting the value.
//
// Errors from the implementations are wrapped as "<operation> failed: ..."
// so the failing operation can be identified without knowing the transport.
type Odoo interface {
	// Login authenticates against the database. The XML-RPC and JSON-RPC
	// transports perform a real authenticate call and retain the uid; the
	// JSON API v2 transport only validates the configuration, because it
	// authenticates every request with a bearer token.
	Login(ctx context.Context) (err error)

	// Create creates a single record and returns its id. On failure it
	// returns -1 and a non-nil error.
	Create(ctx context.Context, model string, values map[string]any) (row int, err error)

	// Load creates records from a header and a list of rows, and returns the
	// ids of the created records.
	Load(ctx context.Context, model string, header []string, values [][]any) (ids []int, err error)

	// Count returns the number of records matching filters.
	Count(ctx context.Context, model string, filters ...any) (count int, err error)

	// FieldsGet returns the definitions of the model's fields, optionally
	// restricted to the given fields and field attributes.
	FieldsGet(ctx context.Context, model string, fields []string, fieldAttributes ...string) (recordFields map[string]any, err error)

	// GetID returns the id of the first record matching filters, or -1 with a
	// nil error when no record matches.
	GetID(ctx context.Context, model string, filters ...any) (id int, err error)

	// Search returns the ids of the records matching filters.
	Search(ctx context.Context, model string, filters ...any) (ids []int, err error)

	// Read returns the requested fields of the records with the given ids.
	Read(ctx context.Context, model string, ids []int, fields ...string) (records []map[string]any, err error)

	// SearchRead returns the requested fields of the records matching
	// filters, starting at offset and returning at most limit records.
	SearchRead(ctx context.Context, model string, offset int, limit int, fields []string, filters ...any) (records []map[string]any, err error)

	// Write updates the record with the given id and reports whether the
	// server accepted the write.
	Write(ctx context.Context, model string, recordID int, values map[string]any) (result bool, err error)

	// Unlink deletes the records with the given ids and reports whether the
	// server accepted the deletion.
	Unlink(ctx context.Context, model string, recordIDs []int) (result bool, err error)

	// Execute calls a model method with positional arguments. Because the
	// result is a bool, it is only useful for methods that return a boolean.
	//
	// The JSON API v2 transport has no positional argument list and returns
	// an error when args is non-empty; use ExecuteKw with keyword arguments
	// for that transport.
	Execute(ctx context.Context, model string, method string, args []any) (result bool, err error)

	// ExecuteKw calls a model method with positional arguments and keyword
	// arguments. Because the result is a bool, it is only useful for methods
	// that return a boolean.
	//
	// The JSON API v2 transport only supports keyword arguments and returns
	// an error when args is non-empty.
	ExecuteKw(ctx context.Context, model string, method string, args []any, kwargs []map[string]any) (result bool, err error)
}
