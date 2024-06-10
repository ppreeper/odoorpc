package odoosearchdomain

import (
	"reflect"
	"testing"
)

var searchDomainPatterns = []struct {
	domain string
	args   []any
	err    error
}{
	{"", []any{}, nil},
	{"('')", []any{}, errSyntax},
	{"('','')", []any{}, errSyntax},
	{"('a','=')", []any{}, errSyntax},
	{"('name')", []any{}, errSyntax},
	{"('name','=')", []any{}, errSyntax},
	{"('name','=','My Name')", []any{[]any{"name", "=", "My Name"}}, nil},
	{"('name','like','My Name')", []any{[]any{"name", "like", "My Name"}}, nil},
	{"('name','=','My Name'),('name','=','My Name')", []any{}, errSyntax},
	{"[('name','=','My Name')]", []any{[]any{"name", "=", "My Name"}}, nil},
	{"[('name','=','My Name'),('name','=','My Name')]", []any{[]any{"name", "=", "My Name"}, []any{"name", "=", "My Name"}}, nil},
	{"[('name','=','My Name'),'!',('name','=','My Name')]", []any{[]any{"name", "=", "My Name"}, []any{"!", []any{"name", "=", "My Name"}}}, nil},
	{"[('name','=','My Name'),'&',('name','=','My Name'),('name','=','My Name')]", []any{[]any{"name", "=", "My Name"}, []any{"&", []any{"name", "=", "My Name"}, []any{"name", "=", "My Name"}}}, nil},
	{"[('name','=','My Name'),'|',('name','=','My Name'),('name','=','My Name')]", []any{[]any{"name", "=", "My Name"}, []any{"|", []any{"name", "=", "My Name"}, []any{"name", "=", "My Name"}}}, nil},
	{"[('name','=','My Name'),('name','=','My Name'),'!',('name','=','My Name')]", []any{[]any{"name", "=", "My Name"}, []any{"name", "=", "My Name"}, []any{"!", []any{"name", "=", "My Name"}}}, nil},
	{"[('name','=','My Name'),('name','=','My Name'),'&',('name','=','My Name')]", []any{}, errSyntax},
	{"[('name','=','My Name'),('name','=','My Name'),'&',('name','=','My Name'),('name','=','My Name')]", []any{[]any{"name", "=", "My Name"}, []any{"name", "=", "My Name"}, []any{"&", []any{"name", "=", "My Name"}, []any{"name", "=", "My Name"}}}, nil},
	{"[('name','=','My Name'),('name','=','My Name'),'|',('name','=','My Name')]", []any{}, errSyntax},
	{"[('name','=','My Name'),('name','=','My Name'),'|',('name','=','My Name'),('name','=','My Name')]", []any{[]any{"name", "=", "My Name"}, []any{"name", "=", "My Name"}, []any{"|", []any{"name", "=", "My Name"}, []any{"name", "=", "My Name"}}}, nil},
	{"[('name','=','My Name'),('name','=','My Name'),'|',('name','=','My Name'),('name','=','My Name'),'!',('name','=','My Name')]", []any{[]any{"name", "=", "My Name"}, []any{"name", "=", "My Name"}, []any{"|", []any{"name", "=", "My Name"}, []any{"name", "=", "My Name"}}, []any{"!", []any{"name", "=", "My Name"}}}, nil},
	{"[('name','=','My Name'),('name','=','My Name'),'|',('name','=','My Name'),('name','=','My Name'),'!',('name','=','My Name'),('name','=','My Name')]", []any{[]any{"name", "=", "My Name"}, []any{"name", "=", "My Name"}, []any{"|", []any{"name", "=", "My Name"}, []any{"name", "=", "My Name"}}, []any{"!", []any{"name", "=", "My Name"}}, []any{"name", "=", "My Name"}}, nil},
}

func TestSearchDomain(t *testing.T) {
	for i, pattern := range searchDomainPatterns {
		// fmt.Println("test: domain:", pattern.domain, "pattern.err:", pattern.err)
		args, err := SearchDomain(pattern.domain)
		if !reflect.DeepEqual(pattern.args, args) {
			t.Errorf("\n[%d]: expected reflect args: %v, got %v", i, pattern.args, args)
		}
		if err != pattern.err {
			t.Errorf("\n[%d]: expected error: %v, got %v", i, pattern.err, err)
		}
	}
}
