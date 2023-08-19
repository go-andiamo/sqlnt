package sqlnt

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

const sqlTag = "sql"

// NewTemplateSet builds a set of templates for the given struct type T
//
// Fields of type sqlnt.NamedTemplate are created and set from the field tag 'sql'
//
// Example:
//
//	type MyTemplateSet struct {
//	  Select sqlnt.NamedTemplate `sql:"SELECT * FROM foo WHERE col_a = :a"`
//	  Insert sqlnt.NamedTemplate `sql:"INSERT INTO foo (col_a, col_b, col_c) VALUES(:a, :b, :c)"`
//	  Delete sqlnt.NamedTemplate `sql:"DELETE FROM foo WHERE col_a = :a"`
//	}
//	set, err := sqlnt.NewTemplateSet[MyTemplateSet]()
//
// Note: If the overall field tag does not contain a 'sql' tag nor any other tags (i.e. there are no double-quotes in it)
// then the entire field tag value is used as the template - enabling the use of carriage returns to format the statement
//
// Example:
//
//	type MyTemplateSet struct {
//	  Select sqlnt.NamedTemplate `SELECT *
//	  FROM foo
//	  WHERE col_a = :a`
//	}
func NewTemplateSet[T any](options ...any) (*T, error) {
	var chk T
	if reflect.TypeOf(chk).Kind() != reflect.Struct {
		return nil, errors.New("not a struct")
	}
	r := new(T)
	if err := setTemplateFields(reflect.ValueOf(r).Elem(), options...); err != nil {
		return nil, err
	}
	return r, nil
}

// MustCreateTemplateSet is the same as NewTemplateSet except that it panics on error
func MustCreateTemplateSet[T any](options ...any) *T {
	r, err := NewTemplateSet[T](options...)
	if err != nil {
		panic(err)
	}
	return r
}

var ntt = reflect.TypeOf((*NamedTemplate)(nil)).Elem()

func setTemplateFields(rv reflect.Value, options ...any) error {
	rvt := rv.Type()
	for i := 0; i < rv.NumField(); i++ {
		fld := rv.Field(i)
		if ft := rvt.Field(i); ft.IsExported() {
			if fld.Type() == ntt {
				tag, ok := ft.Tag.Lookup(sqlTag)
				if !ok {
					if ft.Tag != "" && !strings.ContainsRune(string(ft.Tag), '"') {
						tag = string(ft.Tag)
					} else {
						return fmt.Errorf("field '%s' does not have '%s' tag", ft.Name, sqlTag)
					}
				}
				if tmp, err := NewNamedTemplate(tag, options...); err == nil {
					fld.Set(reflect.ValueOf(tmp))
				} else {
					return err
				}
			} else if fld.Kind() == reflect.Struct {
				sub := reflect.New(fld.Type()).Elem()
				if err := setTemplateFields(sub, options...); err == nil {
					fld.Set(sub)
				} else {
					return err
				}
			}
		}
	}
	return nil
}
