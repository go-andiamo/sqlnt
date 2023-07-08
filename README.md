# SQLNT
[![GoDoc](https://godoc.org/github.com/go-andiamo/sqlnt?status.svg)](https://pkg.go.dev/github.com/go-andiamo/sqlnt)
[![Latest Version](https://img.shields.io/github/v/tag/go-andiamo/sqlnt.svg?sort=semver&style=flat&label=version&color=blue)](https://github.com/go-andiamo/sqlnt/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/go-andiamo/sqlnt)](https://goreportcard.com/report/github.com/go-andiamo/sqlnt)


Go package for named SQL templates... tired of battling `driver does not support the use of Named Parameters`
or wish you could reliably use named parameters instead of incomprehensible `(?, ?, ?, ?, ?, ?)`

Try...

```go
package main

import (
    "database/sql"
    "github.com/go-andiamo/sqlnt"
)

var template = sqlnt.MustCreateNamedTemplate(`INSERT INTO table 
(col_a, col_b, col_c)
VALUES (:a, :b, :c)`, nil)

func insertExample(db *sql.DB, aVal string, bVal string, cVal string) error {
    _, err := db.Exec(template.Statement(), template.MustArgs(
        map[string]any{
            "a": aVal,
            "b": bVal,
            "c": cVal,
    })...)
    return err
}
```
Or...
```go
package main

import (
    "database/sql"
    "github.com/go-andiamo/sqlnt"
)

var template = sqlnt.MustCreateNamedTemplate(`INSERT INTO table 
(col_a, col_b, col_c)
VALUES (:a, :b, :c)`, nil)

func insertExample(db *sql.DB, aVal string, bVal string, cVal string) error {
    _, err := template.Exec(db,
        map[string]any{
            "a": aVal,
            "b": bVal,
        }, 
        sql.NamedArg{Name: "c", Value: cVal})
    return err
}
```