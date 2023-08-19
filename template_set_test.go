package sqlnt

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

type MySet struct {
	Select NamedTemplate `SELECT *
FROM {{tableName}}
WHERE col_a = :a`
	Insert NamedTemplate `sql:"INSERT INTO {{tableName}} (col_a,col_b,col_c) VALUES (:a,:b,:c)"`
	Delete NamedTemplate `sql:"DELETE FROM {{tableName}} WHERE col_a = :a"`
	delete NamedTemplate // unexported fields not used
}

type MySet2 struct {
	MySet
}

type MySet3 struct {
	MySet2
}

func TestNewTemplateSet(t *testing.T) {
	ts, err := NewTemplateSet[MySet](testTokenOption)
	assert.NoError(t, err)
	assert.NotNil(t, ts.Select)
	assert.Equal(t, "SELECT *\nFROM foo\nWHERE col_a = ?", ts.Select.Statement())
	assert.NotNil(t, ts.Insert)
	assert.Equal(t, "INSERT INTO foo (col_a,col_b,col_c) VALUES (?,?,?)", ts.Insert.Statement())
	assert.NotNil(t, ts.Delete)
	assert.Equal(t, "DELETE FROM foo WHERE col_a = ?", ts.Delete.Statement())
	assert.Nil(t, ts.delete)
}

func TestNewTemplateSet_Anonymous(t *testing.T) {
	ts, err := NewTemplateSet[struct {
		Select NamedTemplate `SELECT *
FROM {{tableName}}
WHERE col_a = :a`
	}](testTokenOption)
	assert.NoError(t, err)
	assert.NotNil(t, ts.Select)
	assert.Equal(t, "SELECT *\nFROM foo\nWHERE col_a = ?", ts.Select.Statement())
}

func TestNewTemplateSet_Nested(t *testing.T) {
	ts, err := NewTemplateSet[MySet2](testTokenOption)
	assert.NoError(t, err)
	assert.NotNil(t, ts.Select)
}

func TestNewTemplateSet_NestedDouble(t *testing.T) {
	ts, err := NewTemplateSet[MySet3](testTokenOption)
	assert.NoError(t, err)
	assert.NotNil(t, ts.Select)
}

func TestNewTemplateSet_Error_NotStruct(t *testing.T) {
	_, err := NewTemplateSet[string](testTokenOption)
	assert.Error(t, err)
	assert.Equal(t, "not a struct", err.Error())
}

type BadSet1 struct {
	Select NamedTemplate // doesn't have 'sql' tag
}

func TestNewTemplateSet_Error_NoSqlTag(t *testing.T) {
	_, err := NewTemplateSet[BadSet1](testTokenOption)
	assert.Error(t, err)
	assert.Equal(t, "field 'Select' does not have 'sql' tag", err.Error())
}

type BadSet2 struct {
	Select NamedTemplate `sql:"{{unknown_token}}"`
}

func TestNewTemplateSet_Error_BadTemplate(t *testing.T) {
	_, err := NewTemplateSet[BadSet2](testTokenOption)
	assert.Error(t, err)
	assert.Equal(t, "unknown token: unknown_token", err.Error())
}

type BadSet3 struct {
	BadSet2
}

func TestNewTemplateSet_Error_NestedBadTemplate(t *testing.T) {
	_, err := NewTemplateSet[BadSet3](testTokenOption)
	assert.Error(t, err)
	assert.Equal(t, "unknown token: unknown_token", err.Error())
}

func TestMustCreateTemplateSet(t *testing.T) {
	ts := MustCreateTemplateSet[MySet](testTokenOption)
	assert.NotNil(t, ts.Select)
}

func TestMustCreateTemplateSet_Panics(t *testing.T) {
	assert.Panics(t, func() {
		_ = MustCreateTemplateSet[BadSet1]()
	})
}
