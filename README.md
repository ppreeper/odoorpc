# odoorpc

Go client for Odoo RPC. One interface, three transports, so tooling can run
against any Odoo version without rewriting its data access.

```go
import "github.com/ppreeper/odoorpc"
```

Requires Go 1.24 or newer.

## Transports

| Package | Protocol | Endpoint | Authentication | Odoo versions |
| --- | --- | --- | --- | --- |
| `odooxmlrpc` | XML-RPC | `/xmlrpc/2/` | database, username, password | 14–20 |
| `odoojrpc` | JSON-RPC | `/jsonrpc/` | database, username, password | 14–20 |
| `odoojson` | JSON API v2 | `/json/2/<model>/<method>` | API key (Bearer) | 19+ |

All three implement `odoorpc.Odoo`, asserted at compile time, so the same
calling code works with each:

```go
var client odoorpc.Odoo = odooxmlrpc.NewOdoo()
```

On Odoo 19 and newer, XML-RPC and JSON-RPC are served by the `rpc` addon. They
are not removed in 20.0; enable the addon on servers that do not ship it by
default. JSON API v2 only exists from 19.0 onward.

## Install

```bash
go get github.com/ppreeper/odoorpc
```

## Quick start

### XML-RPC

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ppreeper/odoorpc/odooxmlrpc"
)

func main() {
	ctx := context.Background()

	client := odooxmlrpc.NewOdoo().
		WithSchema("https").
		WithHostname("odoo.example.com").
		WithPort(443).
		WithDatabase("odoo").
		WithUsername("admin").
		WithPassword("secret")

	if err := client.Login(ctx); err != nil {
		log.Fatalf("login: %v", err)
	}

	partners, err := client.SearchRead(ctx, "res.partner", 0, 10,
		[]string{"name", "email"}, "is_company", "=", true)
	if err != nil {
		log.Fatalf("search_read: %v", err)
	}
	for _, partner := range partners {
		fmt.Println(partner["name"], partner["email"])
	}
}
```

### JSON-RPC

Same configuration and calls; only the package and constructor change.

```go
client := odoojrpc.NewOdoo().
	WithSchema("https").
	WithHostname("odoo.example.com").
	WithPort(443).
	WithDatabase("odoo").
	WithUsername("admin").
	WithPassword("secret")

if err := client.Login(ctx); err != nil {
	log.Fatalf("login: %v", err)
}
```

### JSON API v2

The v2 endpoint authenticates every request with an API key, so there is no
session to establish. `Login` validates the configuration and returns an error
for a missing key, invalid schema, port, or hostname; it does not contact the
server.

```go
client := odoojson.NewOdoo().
	WithSchema("https").
	WithHostname("odoo.example.com").
	WithPort(443).
	WithDatabase("odoo").
	WithAPIKey(os.Getenv("ODOO_API_KEY"))

if err := client.Login(ctx); err != nil {
	log.Fatalf("config: %v", err)
}
```

Create a key in Odoo under **Preferences → Account Security → New API Key**
(or the equivalent user form in the backend). The key is shown once. Keys can
be scoped to specific endpoints: a key issued for another service (for example
the MCP endpoint) is rejected by `/json/2/` with `401 Invalid apikey`, so issue
a key intended for RPC use.

The client sends the key as `Authorization: Bearer <key>` and sets
`X-Odoo-Database` from `WithDatabase`, which selects the database on servers
that host more than one.

## Common operations

The interface methods are transport independent:

```go
// Search returns matching ids.
ids, err := client.Search(ctx, "res.partner", "is_company", "=", true)

// GetID returns the first match, or -1 and a nil error when nothing matches.
id, err := client.GetID(ctx, "res.partner", "name", "=", "Acme")

// Count returns the number of matches.
count, err := client.Count(ctx, "res.partner", "is_company", "=", true)

// FieldsGet returns field definitions, optionally limited to specific fields
// and attributes.
fields, err := client.FieldsGet(ctx, "res.partner",
	[]string{"name", "email"}, "string", "type", "required")

// SearchRead combines search and read.
records, err := client.SearchRead(ctx, "res.partner", 0, 10,
	[]string{"name", "email"}, "is_company", "=", true)

// Read returns the requested fields for known ids.
records, err = client.Read(ctx, "res.partner", []int{1, 2}, "name", "email")

// Create returns the id of the new record.
partnerID, err := client.Create(ctx, "res.partner", map[string]any{
	"name":  "Acme",
	"email": "info@acme.example",
})

// Write updates one record and reports whether the server accepted it.
ok, err := client.Write(ctx, "res.partner", partnerID, map[string]any{
	"email": "sales@acme.example",
})

// Load creates records from a header and rows.
newIDs, err := client.Load(ctx, "res.partner",
	[]string{"name", "email"}, [][]any{
		{"Globex", "info@globex.example"},
		{"Initech", "info@initech.example"},
	})

// Unlink deletes records.
ok, err = client.Unlink(ctx, "res.partner", newIDs)
```

`Search`, `Count`, `GetID`, `SearchRead` and the other filter-taking methods
accept a domain as variadic terms:

```go
// is_company = true AND country_id = 10 (adjacent terms are ANDed)
client.Search(ctx, "res.partner", "is_company", "=", true, "country_id", "=", 10)
```

A single `[]any` argument is treated as a complete domain, which is how OR and
explicit connectors are expressed:

```go
client.Search(ctx, "res.partner", []any{
	"|",
	[]any{"name", "ilike", "acme"},
	[]any{"email", "ilike", "acme"},
})
```

A helper can depend on the interface rather than a concrete transport:

```go
func companyNames(ctx context.Context, client odoorpc.Odoo) ([]string, error) {
	records, err := client.SearchRead(ctx, "res.partner", 0, 100,
		[]string{"name"}, "is_company", "=", true)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(records))
	for _, record := range records {
		if name, ok := record["name"].(string); ok {
			names = append(names, name)
		}
	}
	return names, nil
}
```

## Errors

Every method returns an error. Check it before using the value:

- Integer methods return `-1` alongside an error.
- `GetID` also returns `-1` with a nil error when no record matches.
- Boolean methods return `false` alongside an error.
- Errors are wrapped as `"<operation> failed: ..."`, for example
  `search_read failed: ...`, so the failing call is identifiable.
- HTTP transports report non-2xx responses with the status code and, when the
  server provides it, the Odoo error argument.

Responses are capped at 32 MiB to protect against a misbehaving server, and the
HTTP clients use a 30 second default timeout, configurable with
`WithTimeout`.

## Limitations

`Execute` and `ExecuteKw` return `bool`, so the interface only exposes model
methods whose result is a boolean (for example `has_group`). This predates the
JSON API v2 and applies to all transports.

JSON API v2 has no positional argument list: every payload key becomes a
keyword argument of the model method. The `odoojson` transport therefore rejects
positional arguments with an explicit error; pass arguments through `ExecuteKw`
instead.

For results other than a boolean, use the transport's generic call:

```go
// JSON-RPC
result, err := odoojrpcClient.Call(ctx, "object", "execute_kw",
	"odoo", uid, "secret", "res.partner", "name_search",
	[]any{"Acme"}, map[string]any{"limit": 5})

// JSON API v2
result, err := odoojsonClient.Call(ctx, "res.partner", "name_search",
	map[string]any{"name": "Acme", "limit": 5})
```

`odooxmlrpc` exposes only the typed interface and `Execute`/`ExecuteKw`; it has
no exported generic call. The JSON API v2 transport also provides `Action`,
which is not part of `odoorpc.Odoo`.

### Version differences

- Odoo 20 treats `limit: 0` in `search_read` as "return no rows", while 14–18
  treat it as "no limit". Pass a positive limit when results must be identical
  on every version.
- Odoo 20's `search_count` rejects an explicit zero limit, so the JSON API v2
  transport omits the key entirely when counting.

## Validation

```bash
go test ./...
```

The transports have been exercised end to end against Odoo 14.0 through 20.0,
over plain HTTP and through a TLS reverse proxy, covering login, count, get-id,
search, read, search-read, fields-get, create, write, read-back, unlink,
read-deleted, load, `execute_kw`, `execute`, and error handling. XML-RPC and
JSON-RPC pass all 16 steps on every version; JSON API v2 passes all 15 steps on
19 and 20 (`Execute` cannot be expressed over v2; `ExecuteKw` is covered). The
matrix and the payload fixes it produced are documented in
[PR #3](https://github.com/ppreeper/odoorpc/pull/3).

## License

Apache License 2.0. See [LICENSE](LICENSE).
