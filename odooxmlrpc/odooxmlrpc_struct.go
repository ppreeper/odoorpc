package odooxmlrpc

import (
	"fmt"

	"github.com/ppreeper/odoorpc/xmlrpc"
)

// OdooXML connection
// Return a new instance of the OdooXML class
type OdooXML struct {
	hostname string `default:"localhost"`
	port     int    `default:"8069"`
	schema   string `default:"http"`
	database string `default:"odoo"`
	username string `default:"odoo"`
	password string `default:"odoo"`
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

func NewOdoo() *OdooXML {
	o := &OdooXML{}
	return o
}

func NewOdooWithConfig(config OdooXML) *OdooXML {
	o := &config
	return o
}

// genURL returns url string
func (o *OdooXML) genURL() (err error) {
	if o.schema != "http" && o.schema != "https" {
		return fmt.Errorf("invalid schema: http or https: %w", err)
	}
	if o.port == 0 || o.port > 65535 {
		return fmt.Errorf("invalid port: 1-65535: %w", err)
	}
	if len(o.hostname) == 0 || len(o.hostname) > 2048 {
		return fmt.Errorf("invalid hostname length: 1-2048: %w", err)
	}
	o.url = fmt.Sprintf("%s://%s:%d/xmlrpc/2/", o.schema, o.hostname, o.port)
	return nil
}
