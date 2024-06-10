package odoorpc

type Odoo interface {
	Login() (err error)
	Create(model string, values map[string]any) (row int, err error)
	Load(model string, header []string, values [][]any) (ids []int, err error)
	Count(model string, filters ...any) (count int, err error)
	FieldsGet(model string, fields []string, fieldAttributes ...string) (recordFields map[string]any, err error)
	GetID(model string, filters ...any) (id int, err error)
	Search(model string, filters ...any) (ids []int, err error)
	Read(model string, ids []int, fields ...string) (records []map[string]any, err error)
	SearchRead(model string, offset int, limit int, fields []string, filters ...any) (records []map[string]any, err error)
	Write(model string, recordID int, values map[string]any) (result bool, err error)
	Unlink(model string, recordIDs []int) (result bool, err error)
	Execute(model string, method string, args []any) (result bool, err error)
	ExecuteKw(model string, method string, args []any, kwargs []map[string]any) (result bool, err error)
}
