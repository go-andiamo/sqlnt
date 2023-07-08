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
	testCases := []struct {
		statement           string
		expectError         bool
		expectStatement     string
		expectArgsCount     int
		expectArgNamesCount int
		option              Option
		omissibleArgs       []string
		inArgs              []any
		expectArgsError     bool
		expectOutArgs       []any
	}{
		{
			statement:           `INSERT INTO table (col_a, col_b, col_c) VALUES(:a, :bb, :ccc)`,
			expectStatement:     `INSERT INTO table (col_a, col_b, col_c) VALUES(?, ?, ?)`,
			expectArgsCount:     3,
			expectArgNamesCount: 3,
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
			option:              MySqlOption,
			expectArgsCount:     3,
			expectArgNamesCount: 3,
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
			option:              MySqlOption,
			expectArgsCount:     3,
			expectArgNamesCount: 3,
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
			option:              MySqlOption,
			expectArgsCount:     3,
			expectArgNamesCount: 3,
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
			option:              MySqlOption,
			expectArgsCount:     3,
			expectArgNamesCount: 3,
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
			option:              MySqlOption,
			expectArgsCount:     3,
			expectArgNamesCount: 3,
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
			option:              MySqlOption,
			expectArgsCount:     3,
			expectArgNamesCount: 3,
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
			option:              MySqlOption,
			expectArgsCount:     3,
			expectArgNamesCount: 3,
			inArgs:              []any{"not a map"},
			expectArgsError:     true,
		},
		{
			statement:           `INSERT INTO table (col_a, col_b, col_c) VALUES(:a, :bb, :ccc)`,
			expectStatement:     `INSERT INTO table (col_a, col_b, col_c) VALUES(?, ?, ?)`,
			option:              MySqlOption,
			expectArgsCount:     3,
			expectArgNamesCount: 3,
			inArgs:              []any{&unmarshalable{}},
			expectArgsError:     true,
		},
		{
			statement:           `INSERT INTO table (col_a, col_b, col_c) VALUES(:a, :bb, :ccc)`,
			expectStatement:     `INSERT INTO table (col_a, col_b, col_c) VALUES($1, $2, $3)`,
			option:              PostgresOption,
			expectArgsCount:     3,
			expectArgNamesCount: 3,
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
			option:              PostgresOption,
			expectArgsCount:     2,
			expectArgNamesCount: 2,
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
			option:              PostgresOption,
			expectArgsCount:     2,
			expectArgNamesCount: 2,
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
			option:              PostgresOption,
			expectArgsCount:     2,
			expectArgNamesCount: 2,
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
			option:              PostgresOption,
			expectArgsCount:     2,
			expectArgNamesCount: 2,
			omissibleArgs:       []string{},
			inArgs:              []any{map[string]any{}},
			expectArgsError:     false,
			expectOutArgs:       []any{nil, nil},
		},
		{
			statement:           `UPDATE table SET col_a = :a`,
			expectStatement:     `UPDATE table SET col_a = ?`,
			expectArgsCount:     1,
			expectArgNamesCount: 1,
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
			option:              PostgresOption,
			expectArgsCount:     1,
			expectArgNamesCount: 1,
			inArgs: []any{
				map[string]any{
					"a": "a value",
				},
			},
			expectOutArgs: []any{"a value"},
		},
		{
			statement:   `INSERT INTO table (col_a, col_b, col_c) VALUES(:, :b, :c)`,
			option:      PostgresOption,
			expectError: true,
		},
		{
			statement:           `INSERT INTO table (col_a, col_b, col_c) VALUES(:a, '::bb', '::ccc')`,
			expectStatement:     `INSERT INTO table (col_a, col_b, col_c) VALUES(?, ':bb', ':ccc')`,
			expectArgsCount:     1,
			expectArgNamesCount: 1,
		},
		{
			statement:           `INSERT INTO table (col_a, col_b, col_c) VALUES(:a, :::bb, '::::ccc')`,
			expectStatement:     `INSERT INTO table (col_a, col_b, col_c) VALUES(?, :?, '::ccc')`,
			expectArgsCount:     2,
			expectArgNamesCount: 2,
		},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("[%d]%s", i+1, tc.statement), func(t *testing.T) {
			nt, err := NewNamedTemplate(tc.statement, tc.option)
			if tc.expectError {
				assert.Nil(t, nt)
				assert.Error(t, err)
				assert.Panics(t, func() {
					_ = MustCreateNamedTemplate(tc.statement, tc.option)
				})
			} else {
				assert.NoError(t, err)
				assert.NotPanics(t, func() {
					_ = MustCreateNamedTemplate(tc.statement, tc.option)
				})
				assert.Equal(t, tc.statement, nt.OriginalStatement())
				assert.Equal(t, tc.expectStatement, nt.Statement())
				assert.Equal(t, tc.expectArgsCount, nt.ArgsCount())
				assert.Equal(t, tc.expectArgNamesCount, len(nt.GetArgNames()))
				if tc.inArgs != nil {
					t.Run("in out args", func(t *testing.T) {
						if tc.omissibleArgs != nil {
							nt.OmissibleArgs(tc.omissibleArgs...)
						}
						outArgs, err := nt.Args(tc.inArgs...)
						if tc.expectArgsError {
							assert.Error(t, err)
							assert.Panics(t, func() {
								_ = nt.MustArgs(tc.inArgs...)
							})
						} else {
							assert.NoError(t, err)
							assert.Equal(t, tc.expectOutArgs, outArgs)
							assert.NotPanics(t, func() {
								_ = nt.MustArgs(tc.inArgs...)
							})
						}
					})
				}
			}
		})
	}
}

type unmarshalable struct{}

func (u *unmarshalable) MarshalJSON() ([]byte, error) {
	return nil, errors.New("fooey")
}

func TestNamedTemplate_DefaultValue(t *testing.T) {
	now := time.Now()
	nt := MustCreateNamedTemplate(`INSERT INTO table (col_a, created_at) VALUES (:a, :crat)`, nil).
		DefaultValue("crat", now)
	assert.Equal(t, `INSERT INTO table (col_a, created_at) VALUES (?, ?)`, nt.Statement())
	args, err := nt.Args(map[string]any{
		"a": "a value",
	})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(args))
	assert.Equal(t, "a value", args[0])
	assert.Equal(t, now, args[1])

	time.Sleep(time.Second)
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
	nt := MustCreateNamedTemplate(`INSERT INTO table (col_a, col_b, col_c) VALUES (:a, :b, :a)`, nil).
		DefaultValue("a", "a default").
		OmissibleArgs("b")
	assert.Equal(t, `INSERT INTO table (col_a, col_b, col_c) VALUES (?, ?, ?)`, nt.Statement())
	assert.Equal(t, []any{"a default", nil, "a default"}, nt.MustArgs(map[string]any{}))

	nt2 := nt.Clone(nil)
	assert.Equal(t, `INSERT INTO table (col_a, col_b, col_c) VALUES (?, ?, ?)`, nt2.Statement())
	assert.Equal(t, []any{"a default", nil, "a default"}, nt2.MustArgs(map[string]any{}))

	nt2 = nt.Clone(PostgresOption)
	assert.Equal(t, `INSERT INTO table (col_a, col_b, col_c) VALUES ($1, $2, $1)`, nt2.Statement())
	assert.Equal(t, []any{"a default", nil}, nt2.MustArgs(map[string]any{}))
}

func TestFoo(t *testing.T) {
	db, mock, _ := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))

	mock.ExpectQuery("SELECT name FROM users WHERE id in (?, ?) limit 1").WithArgs(1, 2).WillReturnError(fmt.Errorf("some error"))
	row := db.QueryRow("SELECT name FROM users WHERE id in (?, ?) limit 1", 1, 2)
	println(row.Err().Error())
	if err := mock.ExpectationsWereMet(); err != nil {
		fmt.Printf("there were unfulfilled expectations: %s", err)
	}
}

func TestNamedTemplate_Exec(t *testing.T) {
	nt, err := NewNamedTemplate(`INSERT INTO table (col_a, col_b, col_c) VALUES (:a, :b, :a)`, nil)
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
	nt, err := NewNamedTemplate(`INSERT INTO table (col_a, col_b, col_c) VALUES (:a, :b, :a)`, nil)
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
	nt, err := NewNamedTemplate(`SELECT * FROM table WHERE col_a = :a OR col_a = :b AND col_c = :a`, nil)
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
	nt, err := NewNamedTemplate(`SELECT * FROM table WHERE col_a = :a OR col_a = :b AND col_c = :a`, nil)
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
