package odoojson

import (
	"net/url"
)

// OdooJSON2 connection
// Return a new instance of the OdooJSON2 class
type OdooJSON struct {
	hostname string `default:"localhost"`
	port     int    `default:"8069"`
	schema   string `default:"http"`
	database string `default:""`
	apikey   string `default:""`
	url      string
}

func (o *OdooJSON) WithHostname(hostname string) *OdooJSON {
	o.hostname = hostname
	return o
}

func (o *OdooJSON) WithPort(port int) *OdooJSON {
	o.port = port
	return o
}

func (o *OdooJSON) WithSchema(schema string) *OdooJSON {
	o.schema = schema
	return o
}

func (o *OdooJSON) WithDatabase(database string) *OdooJSON {
	o.database = database
	return o
}

func (o *OdooJSON) WithAPIKey(apiKey string) *OdooJSON {
	o.apikey = apiKey
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
func (o *OdooJSON) genURL(model, method string) string {
	urlPath, _ := url.JoinPath(o.url, model, method)
	return urlPath
}
