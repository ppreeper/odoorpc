package odoojrpc

import (
	"errors"
	"fmt"
	"testing"
)

var (
	ErrSchema  = errors.New("invalid schema: http or https")
	ErrPort    = errors.New("invalid port: 1-65535")
	ErrHostLen = errors.New("invalid hostname length: 1-2048")
)

var urlPatterns = []struct {
	schema        string
	hostname      string
	port          int
	expected      string
	expectedError error
}{
	{"", "", 0, "", ErrSchema},
	{"", "localhost", 0, "", fmt.Errorf("init error: %w", ErrSchema)},
	{"", "", 8069, "", fmt.Errorf("init error: %w", ErrSchema)},
	{"http", "", 0, "", fmt.Errorf("init error: %w", ErrPort)},
	{"http", "localhost", 0, "", fmt.Errorf("init error: %w", ErrPort)},
	{"http", "", 8069, "", fmt.Errorf("init error: %w", ErrHostLen)},
	{"https", "", 0, "", fmt.Errorf("init error: %w", ErrPort)},
	{"https", "localhost", 0, "", fmt.Errorf("init error: %w", ErrPort)},
	{"https", "", 8069, "", fmt.Errorf("init error: %w", ErrHostLen)},
	{"ftp", "", 0, "", fmt.Errorf("init error: %w", ErrSchema)},
	{"ftp", "localhost", 0, "", fmt.Errorf("init error: %w", ErrSchema)},
	{"ftp", "", 8069, "", fmt.Errorf("init error: %w", ErrSchema)},
	{"ftp", "localhost", 8069, "", fmt.Errorf("init error: %w", ErrSchema)},
	{"http", "localhost", 8069, "http://localhost:8069/jsonrpc", nil},
	{"https", "localhost", 8069, "https://localhost:8069/jsonrpc", nil},
}

func TestURL(t *testing.T) {
	for i, pattern := range urlPatterns {
		o := OdooJSON{
			hostname: pattern.hostname,
			port:     pattern.port,
			schema:   pattern.schema,
		}

		o.genURL()

		if len(o.url) != len(pattern.expected) {
			t.Errorf("\n[%d]: slice size not equal, expected: %d, got %d", i, len(pattern.expected), len(o.url))
			t.Errorf("\n[%d]: expected %s, got %s", i, pattern.expected, o.url)
		}
	}
}
