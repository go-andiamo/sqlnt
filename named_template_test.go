package sqlnt

import (
	"fmt"
	"github.com/stretchr/testify/assert"
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
		inArgs              map[string]any
		expectArgsError     bool
		expectOutArgs       []any
	}{
		{
			statement:           `INSERT INTO table (col_a, col_b, col_c) VALUES(:a, :bb, :ccc)`,
			expectStatement:     `INSERT INTO table (col_a, col_b, col_c) VALUES(?, ?, ?)`,
			expectArgsCount:     3,
			expectArgNamesCount: 3,
			inArgs: map[string]any{
				"a":   "a value",
				"bb":  "bb value",
				"ccc": "ccc value",
			},
			expectOutArgs: []any{"a value", "bb value", "ccc value"},
		},
		{
			statement:           `INSERT INTO table (col_a, col_b, col_c) VALUES(:a, :bb, :ccc)`,
			expectStatement:     `INSERT INTO table (col_a, col_b, col_c) VALUES(?, ?, ?)`,
			option:              MySqlOption,
			expectArgsCount:     3,
			expectArgNamesCount: 3,
			inArgs: map[string]any{
				"a":   "a value",
				"bb":  "bb value",
				"ccc": "ccc value",
			},
			expectOutArgs: []any{"a value", "bb value", "ccc value"},
		},
		{
			statement:           `INSERT INTO table (col_a, col_b, col_c) VALUES(:a, :bb, :ccc)`,
			expectStatement:     `INSERT INTO table (col_a, col_b, col_c) VALUES($1, $2, $3)`,
			option:              PostgresOption,
			expectArgsCount:     3,
			expectArgNamesCount: 3,
			inArgs: map[string]any{
				"a":   "a value",
				"bb":  "bb value",
				"ccc": "ccc value",
			},
			expectOutArgs: []any{"a value", "bb value", "ccc value"},
		},
		{
			statement:           `INSERT INTO table (col_a, col_b, col_c) VALUES(:a, :b, :a)`,
			expectStatement:     `INSERT INTO table (col_a, col_b, col_c) VALUES(?, ?, ?)`,
			expectArgsCount:     3,
			expectArgNamesCount: 2,
			inArgs: map[string]any{
				"a": "a value",
				"b": "b value",
			},
			expectOutArgs: []any{"a value", "b value", "a value"},
		},
		{
			statement:           `INSERT INTO table (col_a, col_b, col_c) VALUES(:a, :b, :a)`,
			expectStatement:     `INSERT INTO table (col_a, col_b, col_c) VALUES(?, ?, ?)`,
			expectArgsCount:     3,
			expectArgNamesCount: 2,
			inArgs: map[string]any{
				"a": "a value",
			},
			expectArgsError: true,
		},
		{
			statement:           `INSERT INTO table (col_a, col_b, col_c) VALUES(:a, :b, :a)`,
			expectStatement:     `INSERT INTO table (col_a, col_b, col_c) VALUES($1, $2, $1)`,
			option:              PostgresOption,
			expectArgsCount:     2,
			expectArgNamesCount: 2,
			inArgs: map[string]any{
				"a": "a value",
				"b": "b value",
			},
			expectOutArgs: []any{"a value", "b value"},
		},
		{
			statement:           `INSERT INTO table (col_a, col_b, col_c) VALUES(:a, :b, :a)`,
			expectStatement:     `INSERT INTO table (col_a, col_b, col_c) VALUES($1, $2, $1)`,
			option:              PostgresOption,
			expectArgsCount:     2,
			expectArgNamesCount: 2,
			inArgs: map[string]any{
				"a": "a value",
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
			inArgs: map[string]any{
				"a": "a value",
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
			inArgs:              map[string]any{},
			expectArgsError:     false,
			expectOutArgs:       []any{nil, nil},
		},
		{
			statement:           `UPDATE table SET col_a = :a`,
			expectStatement:     `UPDATE table SET col_a = ?`,
			expectArgsCount:     1,
			expectArgNamesCount: 1,
			inArgs: map[string]any{
				"a": "a value",
			},
			expectOutArgs: []any{"a value"},
		},
		{
			statement:           `UPDATE table SET col_a = :a`,
			expectStatement:     `UPDATE table SET col_a = $1`,
			option:              PostgresOption,
			expectArgsCount:     1,
			expectArgNamesCount: 1,
			inArgs: map[string]any{
				"a": "a value",
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
						outArgs, err := nt.Args(tc.inArgs)
						if tc.expectArgsError {
							assert.Error(t, err)
							assert.Panics(t, func() {
								_ = nt.MustArgs(tc.inArgs)
							})
						} else {
							assert.NoError(t, err)
							assert.Equal(t, tc.expectOutArgs, outArgs)
							assert.NotPanics(t, func() {
								_ = nt.MustArgs(tc.inArgs)
							})
						}
					})
				}
			}
		})
	}
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
