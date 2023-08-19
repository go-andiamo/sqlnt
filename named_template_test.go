package sqlnt

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestNamedTemplate(t *testing.T) {
	emptyString := ""
	testCases := []struct {
		statement           string
		expectError         bool
		expectErrorMessage  string
		expectStatement     string
		expectOriginal      string
		expectArgsCount     int
		expectArgNamesCount int
		expectArgNames      []string
		expectOmissibleArgs []string
		options             []any
		omissibleArgs       []string
		nullableStringArgs  []string
		inArgs              []any
		expectArgsError     bool
		expectOutArgs       []any
	}{
		{
			statement:           `INSERT INTO table (col_a, col_b, col_c) VALUES(:a, :bb, :ccc)`,
			expectStatement:     `INSERT INTO table (col_a, col_b, col_c) VALUES(?, ?, ?)`,
			expectArgsCount:     3,
			expectArgNamesCount: 3,
			expectArgNames:      []string{"a", "bb", "ccc"},
			inArgs: []any{
				map[string]any{
					"a":   "a value",
					"bb":  "bb value",
					"ccc": "ccc value",
				},
			},
			expectOutArgs: []any{"a value", "bb value", "ccc value"},
		},
		{
			statement:           `INSERT INTO table (col_a, col_b, col_c) VALUES(:a, :bb, :ccc)`,
			expectStatement:     `INSERT INTO table (col_a, col_b, col_c) VALUES(?, ?, ?)`,
			options:             []any{MySqlOption},
			expectArgsCount:     3,
			expectArgNamesCount: 3,
			expectArgNames:      []string{"a", "bb", "ccc"},
			inArgs: []any{
				map[string]any{
					"a":   "a value",
					"bb":  "bb value",
					"ccc": "ccc value",
				},
			},
			expectOutArgs: []any{"a value", "bb value", "ccc value"},
		},
		{
			statement:           `INSERT INTO table (col_a, col_b, col_c) VALUES(:a, :bb, :ccc)`,
			expectStatement:     `INSERT INTO table (col_a, col_b, col_c) VALUES(?, ?, ?)`,
			options:             []any{MySqlOption},
			expectArgsCount:     3,
			expectArgNamesCount: 3,
			expectArgNames:      []string{"a", "bb", "ccc"},
			inArgs: []any{
				map[string]any{
					"a": "a value",
				},
				sql.NamedArg{
					Name:  "bb",
					Value: "bb value",
				},
				&sql.NamedArg{
					Name:  "ccc",
					Value: "ccc value",
				},
			},
			expectOutArgs: []any{"a value", "bb value", "ccc value"},
		},
		{
			statement:           `INSERT INTO table (col_a, col_b, col_c) VALUES(:a, :bb, :ccc)`,
			expectStatement:     `INSERT INTO table (col_a, col_b, col_c) VALUES(?, ?, ?)`,
			options:             []any{MySqlOption},
			expectArgsCount:     3,
			expectArgNamesCount: 3,
			expectArgNames:      []string{"a", "bb", "ccc"},
			inArgs: []any{
				struct {
					A string `json:"a"`
					B string `json:"bb"`
					C string `json:"ccc"`
				}{
					"a value",
					"bb value",
					"ccc value",
				},
			},
			expectOutArgs: []any{"a value", "bb value", "ccc value"},
		},
		{
			statement:           `INSERT INTO table (col_a, col_b, col_c) VALUES(:a, :bb, :ccc)`,
			expectStatement:     `INSERT INTO table (col_a, col_b, col_c) VALUES(?, ?, ?)`,
			options:             []any{MySqlOption},
			expectArgsCount:     3,
			expectArgNamesCount: 3,
			expectArgNames:      []string{"a", "bb", "ccc"},
			inArgs: []any{
				map[string]int{
					"a":   3,
					"bb":  2,
					"ccc": 1,
				},
			},
			expectOutArgs: []any{3, 2, 1},
		},
		{
			statement:           `INSERT INTO table (col_a, col_b, col_c) VALUES(:a, :bb, :ccc)`,
			expectStatement:     `INSERT INTO table (col_a, col_b, col_c) VALUES(?, ?, ?)`,
			options:             []any{MySqlOption},
			expectArgsCount:     3,
			expectArgNamesCount: 3,
			expectArgNames:      []string{"a", "bb", "ccc"},
			inArgs: []any{
				map[any]int{
					"a":   3,
					"bb":  2,
					"ccc": 1,
				},
			},
			expectOutArgs: []any{3, 2, 1},
		},
		{
			statement:           `INSERT INTO table (col_a, col_b, col_c) VALUES(:a, :bb, :ccc)`,
			expectStatement:     `INSERT INTO table (col_a, col_b, col_c) VALUES(?, ?, ?)`,
			options:             []any{MySqlOption},
			expectArgsCount:     3,
			expectArgNamesCount: 3,
			expectArgNames:      []string{"a", "bb", "ccc"},
			inArgs: []any{
				map[any]int{
					"a": 3,
					2:   2, // key is not a string!
				},
			},
			expectArgsError: true,
		},
		{
			statement:           `INSERT INTO table (col_a, col_b, col_c) VALUES(:a, :bb, :ccc)`,
			expectStatement:     `INSERT INTO table (col_a, col_b, col_c) VALUES(?, ?, ?)`,
			options:             []any{MySqlOption},
			expectArgsCount:     3,
			expectArgNamesCount: 3,
			expectArgNames:      []string{"a", "bb", "ccc"},
			inArgs:              []any{"not a map"},
			expectArgsError:     true,
		},
		{
			statement:           `INSERT INTO table (col_a, col_b, col_c) VALUES(:a, :bb, :ccc)`,
			expectStatement:     `INSERT INTO table (col_a, col_b, col_c) VALUES(?, ?, ?)`,
			options:             []any{MySqlOption},
			expectArgsCount:     3,
			expectArgNamesCount: 3,
			expectArgNames:      []string{"a", "bb", "ccc"},
			inArgs:              []any{&unmarshalable{}},
			expectArgsError:     true,
		},
		{
			statement:           `INSERT INTO table (col_a, col_b, col_c) VALUES(:a, :bb, :ccc)`,
			expectStatement:     `INSERT INTO table (col_a, col_b, col_c) VALUES($1, $2, $3)`,
			options:             []any{PostgresOption},
			expectArgsCount:     3,
			expectArgNamesCount: 3,
			expectArgNames:      []string{"a", "bb", "ccc"},
			inArgs: []any{
				map[string]any{
					"a":   "a value",
					"bb":  "bb value",
					"ccc": "ccc value",
				},
			},
			expectOutArgs: []any{"a value", "bb value", "ccc value"},
		},
		{
			statement:           `INSERT INTO table (col_a, col_b, col_c) VALUES(:a, :b, :a)`,
			expectStatement:     `INSERT INTO table (col_a, col_b, col_c) VALUES(?, ?, ?)`,
			expectArgsCount:     3,
			expectArgNamesCount: 2,
			expectArgNames:      []string{"a", "b"},
			inArgs: []any{
				map[string]any{
					"a": "a value",
					"b": "b value",
				},
			},
			expectOutArgs: []any{"a value", "b value", "a value"},
		},
		{
			statement:           `INSERT INTO table (col_a, col_b, col_c) VALUES(:a, :b, :a)`,
			expectStatement:     `INSERT INTO table (col_a, col_b, col_c) VALUES(?, ?, ?)`,
			expectArgsCount:     3,
			expectArgNamesCount: 2,
			expectArgNames:      []string{"a", "b"},
			inArgs: []any{
				map[string]any{
					"a": "a value",
				},
			},
			expectArgsError: true,
		},
		{
			statement:           `INSERT INTO table (col_a, col_b, col_c) VALUES(:a, :b, :a)`,
			expectStatement:     `INSERT INTO table (col_a, col_b, col_c) VALUES($1, $2, $1)`,
			options:             []any{PostgresOption},
			expectArgsCount:     2,
			expectArgNamesCount: 2,
			expectArgNames:      []string{"a", "b"},
			inArgs: []any{
				map[string]any{
					"a": "a value",
					"b": "b value",
				},
			},
			expectOutArgs: []any{"a value", "b value"},
		},
		{
			statement:           `INSERT INTO table (col_a, col_b, col_c) VALUES(:a, :b, :a)`,
			expectStatement:     `INSERT INTO table (col_a, col_b, col_c) VALUES($1, $2, $1)`,
			options:             []any{PostgresOption},
			expectArgsCount:     2,
			expectArgNamesCount: 2,
			expectArgNames:      []string{"a", "b"},
			inArgs: []any{
				map[string]any{
					"a": "a value",
				},
			},
			expectArgsError: true,
		},
		{
			statement:           `INSERT INTO table (col_a, col_b, col_c) VALUES(:a, :b, :a)`,
			expectStatement:     `INSERT INTO table (col_a, col_b, col_c) VALUES($1, $2, $1)`,
			options:             []any{PostgresOption},
			expectArgsCount:     2,
			expectArgNamesCount: 2,
			expectArgNames:      []string{"a", "b"},
			expectOmissibleArgs: []string{"b"},
			omissibleArgs:       []string{"b"},
			inArgs: []any{
				map[string]any{
					"a": "a value",
				},
			},
			expectArgsError: false,
			expectOutArgs:   []any{"a value", nil},
		},
		{
			statement:           `INSERT INTO table (col_a, col_b, col_c) VALUES(:a, :b, :a)`,
			expectStatement:     `INSERT INTO table (col_a, col_b, col_c) VALUES($1, $2, $1)`,
			options:             []any{PostgresOption},
			expectArgsCount:     2,
			expectArgNamesCount: 2,
			expectArgNames:      []string{"a", "b"},
			expectOmissibleArgs: []string{"a", "b"},
			omissibleArgs:       []string{},
			inArgs:              []any{map[string]any{}},
			expectArgsError:     false,
			expectOutArgs:       []any{nil, nil},
		},
		{
			statement:           `INSERT INTO table (col_a, col_b, col_c) VALUES(:a, :b, :a?)`,
			expectStatement:     `INSERT INTO table (col_a, col_b, col_c) VALUES($1, $2, $1)`,
			options:             []any{PostgresOption},
			expectArgsCount:     2,
			expectArgNamesCount: 2,
			expectArgNames:      []string{"a", "b"},
			expectOmissibleArgs: []string{"a"},
			inArgs:              []any{map[string]any{"b": "b value"}},
			expectArgsError:     false,
			expectOutArgs:       []any{nil, "b value"},
		},
		{
			statement:           `INSERT INTO table (col_a, col_b, col_c) VALUES(:a?, :b, :a)`,
			expectStatement:     `INSERT INTO table (col_a, col_b, col_c) VALUES($1, $2, $1)`,
			options:             []any{PostgresOption},
			expectArgsCount:     2,
			expectArgNamesCount: 2,
			expectArgNames:      []string{"a", "b"},
			expectOmissibleArgs: []string{"a"},
			inArgs:              []any{map[string]any{"b": "b value"}},
			expectArgsError:     false,
			expectOutArgs:       []any{nil, "b value"},
		},
		{
			statement:           `SELECT * FROM table WHERE col_a = :a?`,
			expectStatement:     `SELECT * FROM table WHERE col_a = ?`,
			options:             []any{},
			expectArgsCount:     1,
			expectArgNamesCount: 1,
			expectArgNames:      []string{"a"},
			expectOmissibleArgs: []string{"a"},
			inArgs:              []any{map[string]any{}},
			expectArgsError:     false,
			expectOutArgs:       []any{nil},
		},
		{
			statement:           `UPDATE table SET col_a = :a`,
			expectStatement:     `UPDATE table SET col_a = ?`,
			expectArgsCount:     1,
			expectArgNamesCount: 1,
			expectArgNames:      []string{"a"},
			inArgs: []any{
				map[string]any{
					"a": "a value",
				},
			},
			expectOutArgs: []any{"a value"},
		},
		{
			statement:           `UPDATE table SET col_a = :a`,
			expectStatement:     `UPDATE table SET col_a = $1`,
			options:             []any{PostgresOption},
			expectArgsCount:     1,
			expectArgNamesCount: 1,
			expectArgNames:      []string{"a"},
			inArgs: []any{
				map[string]any{
					"a": "a value",
				},
			},
			expectOutArgs: []any{"a value"},
		},
		{
			statement:           `UPDATE table SET col_a = :a?`,
			expectStatement:     `UPDATE table SET col_a = ?`,
			expectArgsCount:     1,
			expectArgNamesCount: 1,
			expectArgNames:      []string{"a"},
			inArgs:              []any{},
			expectOutArgs:       []any{nil},
		},
		{
			statement:           `UPDATE table SET col_a = :a?`,
			expectStatement:     `UPDATE table SET col_a = $1`,
			options:             []any{PostgresOption},
			expectArgsCount:     1,
			expectArgNamesCount: 1,
			expectArgNames:      []string{"a"},
			inArgs:              []any{},
			expectOutArgs:       []any{nil},
		},
		{
			statement:          `INSERT INTO table (col_a, col_b, col_c) VALUES(:, :b, :c)`,
			options:            []any{PostgresOption},
			expectError:        true,
			expectErrorMessage: "named marker ':' without name (at position 47)",
		},
		{
			statement:           `INSERT INTO table (col_a, col_b, col_c) VALUES(:a, '::bb', '::ccc')`,
			expectStatement:     `INSERT INTO table (col_a, col_b, col_c) VALUES(?, ':bb', ':ccc')`,
			expectArgsCount:     1,
			expectArgNamesCount: 1,
			expectArgNames:      []string{"a"},
		},
		{
			statement:           `INSERT INTO table (col_a, col_b, col_c) VALUES(:a, :::bb, '::::ccc')`,
			expectStatement:     `INSERT INTO table (col_a, col_b, col_c) VALUES(?, :?, '::ccc')`,
			expectArgsCount:     2,
			expectArgNamesCount: 2,
			expectArgNames:      []string{"a", "bb"},
		},
		{
			statement:           `UPDATE table SET col_a = ::`,
			expectStatement:     `UPDATE table SET col_a = :`,
			expectArgsCount:     0,
			expectArgNamesCount: 0,
		},
		{
			statement:           `UPDATE table SET col_a = ::::`,
			expectStatement:     `UPDATE table SET col_a = ::`,
			expectArgsCount:     0,
			expectArgNamesCount: 0,
		},
		{
			statement:          `UPDATE table SET col_a = :a`,
			options:            []any{true}, // not a valid option
			expectError:        true,
			expectErrorMessage: "invalid option",
		},
		{
			statement:           `UPDATE table SET col_a = :a`,
			expectStatement:     `UPDATE table SET col_a = ?`,
			options:             []any{nil},
			expectArgsCount:     1,
			expectArgNamesCount: 1,
			expectArgNames:      []string{"a"},
			expectError:         false,
		},
		{
			statement:           `INSERT INTO {{tableName}} ({{cols}}) VALUES(:{{argA}},:{{argB}},:{{argC}})`,
			expectStatement:     `INSERT INTO foo (col_a,col_b,col_c) VALUES($1,$2,$3)`,
			expectOriginal:      `INSERT INTO foo (col_a,col_b,col_c) VALUES(:a,:b,:c)`,
			options:             []any{PostgresOption, testTokenOption},
			expectArgsCount:     3,
			expectArgNamesCount: 3,
			expectArgNames:      []string{"a", "b", "c"},
			inArgs: []any{
				map[string]any{
					"a": "a value",
					"b": "b value",
					"c": "c value",
				},
			},
			expectOutArgs: []any{"a value", "b value", "c value"},
		},
		{
			statement:           `INSERT INTO {{nested}} ({{cols}}) VALUES(:{{argA}},:{{argB}},:{{argC}})`,
			expectStatement:     `INSERT INTO foo (col_a,col_b,col_c) VALUES($1,$2,$3)`,
			expectOriginal:      `INSERT INTO foo (col_a,col_b,col_c) VALUES(:a,:b,:c)`,
			options:             []any{PostgresOption, testTokenOption},
			expectArgsCount:     3,
			expectArgNamesCount: 3,
			expectArgNames:      []string{"a", "b", "c"},
			inArgs: []any{
				map[string]any{
					"a": "a value",
					"b": "b value",
					"c": "c value",
				},
			},
			expectOutArgs: []any{"a value", "b value", "c value"},
		},
		{
			statement:          `INSERT INTO {{unknownToken}} ({{cols}}) VALUES({{argA}},{{argB}},{{argC}})`,
			options:            []any{PostgresOption, testTokenOption},
			expectError:        true,
			expectErrorMessage: "unknown token: unknownToken",
		},
		{
			statement:          `INSERT INTO {{unknown token}} ({{another unknown}}) VALUES({{argA}},{{argB}},{{argC}})`,
			options:            []any{PostgresOption, testTokenOption},
			expectError:        true,
			expectErrorMessage: "unknown tokens: unknown token, another unknown",
		},
		{
			statement:           `INSERT INTO table (col_a, col_b, col_c) VALUES(:a, :b, :a)`,
			expectStatement:     `INSERT INTO table (col_a, col_b, col_c) VALUES($1, $2, $1)`,
			options:             []any{PostgresOption},
			expectArgsCount:     2,
			expectArgNamesCount: 2,
			expectArgNames:      []string{"a", "b"},
			nullableStringArgs:  []string{"a"},
			inArgs: []any{map[string]any{
				"a": "",
				"b": "",
			}},
			expectArgsError: false,
			expectOutArgs:   []any{nil, ""},
		},
		{
			statement:           `INSERT INTO table (col_a, col_b, col_c) VALUES(:a, :b, :a)`,
			expectStatement:     `INSERT INTO table (col_a, col_b, col_c) VALUES($1, $2, $1)`,
			options:             []any{PostgresOption},
			expectArgsCount:     2,
			expectArgNamesCount: 2,
			expectArgNames:      []string{"a", "b"},
			nullableStringArgs:  []string{"a"},
			inArgs: []any{map[string]any{
				"a": &emptyString,
				"b": "",
			}},
			expectArgsError: false,
			expectOutArgs:   []any{nil, ""},
		},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("[%d]%s", i+1, tc.statement), func(t *testing.T) {
			nt, err := NewNamedTemplate(tc.statement, tc.options...)
			if tc.expectError {
				assert.Nil(t, nt)
				assert.Error(t, err)
				assert.Equal(t, tc.expectErrorMessage, err.Error())
				assert.Panics(t, func() {
					_ = MustCreateNamedTemplate(tc.statement, tc.options...)
				})
			} else {
				assert.NoError(t, err)
				assert.NotPanics(t, func() {
					_ = MustCreateNamedTemplate(tc.statement, tc.options...)
				})
				if tc.omissibleArgs != nil {
					nt.OmissibleArgs(tc.omissibleArgs...)
				}
				nt.NullableStringArgs(tc.nullableStringArgs...)
				if tc.expectOriginal == "" {
					assert.Equal(t, tc.statement, nt.OriginalStatement())
				} else {
					assert.Equal(t, tc.expectOriginal, nt.OriginalStatement())
				}
				assert.Equal(t, tc.expectStatement, nt.Statement())
				assert.Equal(t, tc.expectArgsCount, nt.ArgsCount())
				args := nt.GetArgNames()
				argsInfo := nt.GetArgsInfo()
				assert.Equal(t, tc.expectArgNamesCount, len(args))
				assert.Equal(t, tc.expectArgNamesCount, len(argsInfo))
				for _, name := range tc.expectArgNames {
					_, exists := args[name]
					assert.True(t, exists)
					_, exists = argsInfo[name]
					assert.True(t, exists)
				}
				for _, name := range tc.expectOmissibleArgs {
					omissible, ok := args[name]
					assert.True(t, ok)
					assert.True(t, omissible)
					assert.True(t, argsInfo[name].Omissible)
				}
				for _, name := range tc.nullableStringArgs {
					assert.True(t, argsInfo[name].NullableString)
				}
				if tc.inArgs != nil {
					t.Run("in out args", func(t *testing.T) {
						outArgs, err := nt.Args(tc.inArgs...)
						if tc.expectArgsError {
							assert.Error(t, err)
							assert.Panics(t, func() {
								_ = nt.MustArgs(tc.inArgs...)
							})
							assert.Panics(t, func() {
								_, _ = nt.MustStatementAndArgs(tc.inArgs...)
							})
						} else {
							assert.NoError(t, err)
							assert.Equal(t, tc.expectOutArgs, outArgs)
							assert.NotPanics(t, func() {
								_ = nt.MustArgs(tc.inArgs...)
							})
							stmt, outArgs, err := nt.StatementAndArgs(tc.inArgs...)
							assert.NoError(t, err)
							assert.Equal(t, tc.expectOutArgs, outArgs)
							assert.Equal(t, tc.expectStatement, stmt)
							assert.NotPanics(t, func() {
								_, _ = nt.MustStatementAndArgs(tc.inArgs...)
							})
						}
					})
				}
			}
		})
	}
}

var testTokenOption = TokenOptionMap{
	"tableName": "foo",
	"cols":      "col_a,col_b,col_c",
	"argA":      "a",
	"argB":      "b",
	"argC":      "c",
	"nested":    "{{tableName}}",
}

type unmarshalable struct{}

func (u *unmarshalable) MarshalJSON() ([]byte, error) {
	return nil, errors.New("fooey")
}

func TestNamedTemplate_DefaultValue(t *testing.T) {
	now := time.Now()
	nt := MustCreateNamedTemplate(`INSERT INTO table (col_a, created_at) VALUES (:a, :crat)`).
		DefaultValue("crat", now)
	assert.Equal(t, `INSERT INTO table (col_a, created_at) VALUES (?, ?)`, nt.Statement())
	args, err := nt.Args(map[string]any{
		"a": "a value",
	})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(args))
	assert.Equal(t, "a value", args[0])
	assert.Equal(t, now, args[1])

	time.Sleep(50 * time.Millisecond) // wait for time to change!
	nt.DefaultValue("crat", func(name string) any {
		return time.Now()
	})
	args, err = nt.Args(map[string]any{
		"a": "a value",
	})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(args))
	assert.Equal(t, "a value", args[0])
	assert.True(t, (args[1].(time.Time)).After(now))
}

func TestNamedTemplate_Clone(t *testing.T) {
	nt := MustCreateNamedTemplate(`INSERT INTO table (col_a, col_b, col_c) VALUES (:a, :b, :a)`).
		DefaultValue("a", "a default").
		OmissibleArgs("b").
		NullableStringArgs("a")
	assert.Equal(t, `INSERT INTO table (col_a, col_b, col_c) VALUES (?, ?, ?)`, nt.Statement())
	assert.Equal(t, []any{"a default", nil, "a default"}, nt.MustArgs(map[string]any{}))

	nt2 := nt.Clone(nil)
	assert.Equal(t, `INSERT INTO table (col_a, col_b, col_c) VALUES (?, ?, ?)`, nt2.Statement())
	assert.Equal(t, []any{"a default", nil, "a default"}, nt2.MustArgs(map[string]any{}))
	info := nt2.GetArgsInfo()
	assert.Equal(t, "?", info["a"].Tag)
	assert.Equal(t, "?", info["b"].Tag)
	assert.True(t, info["a"].Omissible)
	assert.True(t, info["b"].Omissible)
	assert.NotNil(t, info["a"].DefaultValue)
	assert.Nil(t, info["b"].DefaultValue)
	assert.True(t, info["a"].NullableString)
	assert.False(t, info["b"].NullableString)

	nt2 = nt.Clone(PostgresOption)
	assert.Equal(t, `INSERT INTO table (col_a, col_b, col_c) VALUES ($1, $2, $1)`, nt2.Statement())
	assert.Equal(t, []any{"a default", nil}, nt2.MustArgs(map[string]any{}))
	info = nt2.GetArgsInfo()
	assert.Equal(t, "$1", info["a"].Tag)
	assert.Equal(t, "$2", info["b"].Tag)
	assert.True(t, info["a"].Omissible)
	assert.True(t, info["b"].Omissible)
	assert.NotNil(t, info["a"].DefaultValue)
	assert.Nil(t, info["b"].DefaultValue)
	assert.True(t, info["a"].NullableString)
	assert.False(t, info["b"].NullableString)
}

func TestNamedTemplate_Append(t *testing.T) {
	nt := MustCreateNamedTemplate(`SELECT * FROM table WHERE col_a = :a`).
		OmissibleArgs("a")
	assert.Equal(t, `SELECT * FROM table WHERE col_a = ?`, nt.Statement())

	nt2, err := nt.Append(` AND col_b = :b`)
	assert.NoError(t, err)
	assert.Equal(t, `SELECT * FROM table WHERE col_a = ? AND col_b = ?`, nt2.Statement())
	argNames := nt2.GetArgNames()
	omissible, ok := argNames["a"]
	assert.True(t, ok)
	assert.True(t, omissible)
	omissible, ok = argNames["b"]
	assert.True(t, ok)
	assert.False(t, omissible)

	assert.NotPanics(t, func() {
		_ = nt.MustAppend(` AND col_b = :b`)
	})

	_, err = nt.Append(` AND col_b = : not valid name`)
	assert.Error(t, err)
	assert.Panics(t, func() {
		_ = nt.MustAppend(` AND col_b = : not valid name`)
	})
}

func TestNamedTemplate_Exec(t *testing.T) {
	nt, err := NewNamedTemplate(`INSERT INTO table (col_a, col_b, col_c) VALUES (:a, :b, :a)`)
	require.NoError(t, err)
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	mock.ExpectExec(nt.Statement()).
		WithArgs("aa", "bb", "aa").
		WillReturnResult(sqlmock.NewResult(1, 1))

	_, err = nt.Exec(db, map[string]any{
		"a": "aa",
		"b": "bb",
	})
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())

	_, err = nt.Exec(db, map[string]any{
		"a": "aa",
	})
	assert.Error(t, err)
}

func TestNamedTemplate_ExecContext(t *testing.T) {
	nt, err := NewNamedTemplate(`INSERT INTO table (col_a, col_b, col_c) VALUES (:a, :b, :a)`)
	require.NoError(t, err)
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	mock.ExpectExec(nt.Statement()).
		WithArgs("aa", "bb", "aa").
		WillReturnResult(sqlmock.NewResult(1, 1))

	_, err = nt.ExecContext(context.Background(), db, map[string]any{
		"a": "aa",
		"b": "bb",
	})
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())

	_, err = nt.ExecContext(context.Background(), db, map[string]any{
		"a": "aa",
	})
	assert.Error(t, err)
}

func TestNamedTemplate_Query(t *testing.T) {
	nt, err := NewNamedTemplate(`SELECT * FROM table WHERE col_a = :a OR col_a = :b AND col_c = :a`)
	require.NoError(t, err)
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	mock.ExpectQuery(nt.Statement()).
		WithArgs("aa", "bb", "aa").
		WillReturnRows()

	_, err = nt.Query(db, map[string]any{
		"a": "aa",
		"b": "bb",
	})
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())

	_, err = nt.Query(db, map[string]any{
		"a": "aa",
	})
	assert.Error(t, err)
}

func TestNamedTemplate_QueryContext(t *testing.T) {
	nt, err := NewNamedTemplate(`SELECT * FROM table WHERE col_a = :a OR col_a = :b AND col_c = :a`)
	require.NoError(t, err)
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	require.NoError(t, err)
	mock.ExpectQuery(nt.Statement()).
		WithArgs("aa", "bb", "aa").
		WillReturnRows()

	_, err = nt.QueryContext(context.Background(), db, map[string]any{
		"a": "aa",
		"b": "bb",
	})
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())

	_, err = nt.QueryContext(context.Background(), db, map[string]any{
		"a": "aa",
	})
	assert.Error(t, err)
}
