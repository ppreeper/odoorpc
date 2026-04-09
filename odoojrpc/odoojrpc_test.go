package odoojrpc

import (
	"errors"
	"fmt"
	"testing"
)

var urlTests = []struct {
	schema        string
	hostname      string
	port          int
	expectedURL   string
	expectedError error
}{
	{"", "", 0, "", errors.New("invalid schema: http or https")},
	{"", "localhost", 0, "", errors.New("invalid schema: http or https")},
	{"", "", 8069, "", errors.New("invalid schema: http or https")},
	{"ftp", "", 0, "", errors.New("invalid schema: http or https")},
	{"ftp", "localhost", 0, "", errors.New("invalid schema: http or https")},
	{"ftp", "", 8069, "", errors.New("invalid schema: http or https")},
	{"ftp", "localhost", 8069, "", errors.New("invalid schema: http or https")},
	{"http", "", 0, "", errors.New("invalid port: 1-65535")},
	{"http", "localhost", 0, "", errors.New("invalid port: 1-65535")},
	{"http", "localhost", -1, "", errors.New("invalid port: 1-65535")},
	{"https", "", 0, "", errors.New("invalid port: 1-65535")},
	{"https", "localhost", 0, "", errors.New("invalid port: 1-65535")},
	{"https", "localhost", -1, "", errors.New("invalid port: 1-65535")},
	{"http", "", 8069, "", errors.New("invalid hostname length: 1-2048")},
	{"https", "", 8069, "", errors.New("invalid hostname length: 1-2048")},
	{"http", "localhost", 8069, "http://localhost:8069/jsonrpc/", nil},
	{"https", "localhost", 8069, "https://localhost:8069/jsonrpc/", nil},
}

func TestURL(t *testing.T) {
	for _, tt := range urlTests {
		tt := tt
		name := fmt.Sprintf("%s://%s:%d/...", tt.schema, tt.hostname, tt.port)
		t.Run(name, func(t *testing.T) {
			o := OdooJSON{
				hostname: tt.hostname,
				port:     tt.port,
				schema:   tt.schema,
			}

			err := o.genURL()

			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("expected error %q, got nil", tt.expectedError)
					return
				}
				if err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error %q, got %q", tt.expectedError, err)
				}
				if o.url != "" {
					t.Errorf("url should be empty on error, got %q", o.url)
				}
				return
			}

			if err != nil {
				t.Errorf("expected no error, got %q", err)
				return
			}
			if o.url != tt.expectedURL {
				t.Errorf("expected url %q, got %q", tt.expectedURL, o.url)
			}
		})
	}
}
