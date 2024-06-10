package odoojrpc

import "fmt"

// OdooJSON connection
// Return a new instance of the :class 'OdooJSON' class.
type OdooJSON struct {
	hostname string `default:"localhost"`
	port     int    `default:"8069"`
	schema   string `default:"http"`
	database string `default:"odoo"`
	username string `default:"odoo"`
	password string `default:"odoo"`
	url      string
	uid      int
}

func (o *OdooJSON) WithHostname(hostname string) *OdooJSON {
	o.hostname = hostname
	return o
}

func (o *OdooJSON) WithPort(port int) *OdooJSON {
	o.port = port
	return o
}

func (o *OdooJSON) WithDatabase(database string) *OdooJSON {
	o.database = database
	return o
}

func (o *OdooJSON) WithUsername(username string) *OdooJSON {
	o.username = username
	return o
}

func (o *OdooJSON) WithPassword(password string) *OdooJSON {
	o.password = password
	return o
}

func (o *OdooJSON) WithSchema(schema string) *OdooJSON {
	o.schema = schema
	return o
}

func NewOdoo() *OdooJSON {
	o := &OdooJSON{}
	return o
}

func NewOdooWithConfig(config OdooJSON) *OdooJSON {
	o := &config
	return o
}

// genURL returns url string
func (o *OdooJSON) genURL() (err error) {
	if o.schema != "http" && o.schema != "https" {
		return fmt.Errorf("invalid schema: http or https: %w", err)
	}
	if o.port == 0 || o.port > 65535 {
		return fmt.Errorf("invalid port: 1-65535: %w", err)
	}
	if len(o.hostname) == 0 || len(o.hostname) > 2048 {
		return fmt.Errorf("invalid hostname length: 1-2048: %w", err)
	}

	o.url = fmt.Sprintf("%s://%s:%d/jsonrpc/", o.schema, o.hostname, o.port)
	return nil
}
