package odoorpc

import "context"

type Odoo interface {
	Login(ctx context.Context) (err error)
	Create(ctx context.Context, model string, values map[string]any) (row int, err error)
	Load(ctx context.Context, model string, header []string, values [][]any) (ids []int, err error)
	Count(ctx context.Context, model string, filters ...any) (count int, err error)
	FieldsGet(ctx context.Context, model string, fields []string, fieldAttributes ...string) (recordFields map[string]any, err error)
	GetID(ctx context.Context, model string, filters ...any) (id int, err error)
	Search(ctx context.Context, model string, filters ...any) (ids []int, err error)
	Read(ctx context.Context, model string, ids []int, fields ...string) (records []map[string]any, err error)
	SearchRead(ctx context.Context, model string, offset int, limit int, fields []string, filters ...any) (records []map[string]any, err error)
	Write(ctx context.Context, model string, recordID int, values map[string]any) (result bool, err error)
	Unlink(ctx context.Context, model string, recordIDs []int) (result bool, err error)
	Execute(ctx context.Context, model string, method string, args []any) (result bool, err error)
	ExecuteKw(ctx context.Context, model string, method string, args []any, kwargs []map[string]any) (result bool, err error)
}
