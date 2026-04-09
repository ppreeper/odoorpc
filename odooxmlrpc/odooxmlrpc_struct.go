package odooxmlrpc

import (
	"fmt"
	"time"

	"github.com/ppreeper/odoorpc"
	"github.com/ppreeper/odoorpc/xmlrpc"
)

// Compile-time assertion that *OdooXML implements the odoorpc.Odoo interface.
var _ odoorpc.Odoo = (*OdooXML)(nil)

// OdooXML connection
// Return a new instance of the OdooXML class
type OdooXML struct {
	hostname string
	port     int
	schema   string
	database string
	username string
	password string
	timeout  time.Duration
	url      string
	uid      int
	common   *xmlrpc.Client
	models   *xmlrpc.Client
}

func (o *OdooXML) WithHostname(hostname string) *OdooXML {
	o.hostname = hostname
	return o
}

func (o *OdooXML) WithPort(port int) *OdooXML {
	o.port = port
	return o
}

func (o *OdooXML) WithDatabase(database string) *OdooXML {
	o.database = database
	return o
}

func (o *OdooXML) WithUsername(username string) *OdooXML {
	o.username = username
	return o
}

func (o *OdooXML) WithPassword(password string) *OdooXML {
	o.password = password
	return o
}

func (o *OdooXML) WithSchema(schema string) *OdooXML {
	o.schema = schema
	return o
}

func (o *OdooXML) WithTimeout(timeout time.Duration) *OdooXML {
	o.timeout = timeout
	return o
}

func NewOdoo() *OdooXML {
	return &OdooXML{
		hostname: "localhost",
		port:     8069,
		schema:   "http",
		database: "odoo",
		username: "odoo",
		password: "odoo",
		timeout:  30 * time.Second,
	}
}

func NewOdooWithConfig(config OdooXML) *OdooXML {
	c := config
	return &c
}

// genURL returns url string
func (o *OdooXML) genURL() error {
	if o.schema != "http" && o.schema != "https" {
		return fmt.Errorf("invalid schema: http or https")
	}
	if o.port < 1 || o.port > 65535 {
		return fmt.Errorf("invalid port: 1-65535")
	}
	if len(o.hostname) == 0 || len(o.hostname) > 2048 {
		return fmt.Errorf("invalid hostname length: 1-2048")
	}
	o.url = fmt.Sprintf("%s://%s:%d/xmlrpc/2/", o.schema, o.hostname, o.port)
	return nil
}
