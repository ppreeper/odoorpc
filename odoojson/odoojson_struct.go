package odoojson

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/ppreeper/odoorpc"
)

// Compile-time assertion that *OdooJSON implements the odoorpc.Odoo interface.
var _ odoorpc.Odoo = (*OdooJSON)(nil)

// OdooJSON2 connection
// Return a new instance of the OdooJSON2 class
type OdooJSON struct {
	hostname string
	port     int
	schema   string
	database string
	apikey   string
	timeout  time.Duration
	url      string
	client   *http.Client
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

func (o *OdooJSON) WithTimeout(timeout time.Duration) *OdooJSON {
	o.timeout = timeout
	return o
}

func NewOdoo() *OdooJSON {
	return &OdooJSON{
		hostname: "localhost",
		port:     8069,
		schema:   "http",
		timeout:  30 * time.Second,
	}
}

func NewOdooWithConfig(config OdooJSON) *OdooJSON {
	c := config
	// Avoid sharing the same *http.Client pointer between the provided config
	// and the returned Odoo instance. Copy the client struct so callers can
	// modify the returned client's fields (e.g. Timeout) without affecting the
	// original config.
	if config.client != nil {
		clientCopy := *config.client
		c.client = &clientCopy
	}
	return &c
}

// genURL validates config and builds the base URL.
func (o *OdooJSON) genURL() error {
	if o.schema != "http" && o.schema != "https" {
		return fmt.Errorf("invalid schema: http or https")
	}
	if o.port < 1 || o.port > 65535 {
		return fmt.Errorf("invalid port: 1-65535")
	}
	if len(o.hostname) == 0 || len(o.hostname) > 2048 {
		return fmt.Errorf("invalid hostname length: 1-2048")
	}
	if len(o.apikey) == 0 {
		return fmt.Errorf("invalid apikey: must not be empty")
	}
	o.url = fmt.Sprintf("%s://%s:%d/json/2/", o.schema, o.hostname, o.port)
	if o.client == nil {
		o.client = &http.Client{Timeout: o.timeout}
	}
	return nil
}

// endpointURL composes the full URL for a model/method call.
func (o *OdooJSON) endpointURL(model, method string) (string, error) {
	urlPath, err := url.JoinPath(o.url, model, method)
	if err != nil {
		return "", fmt.Errorf("endpointURL: %w", err)
	}
	return urlPath, nil
}
