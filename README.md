# SQLNT
[![GoDoc](https://godoc.org/github.com/go-andiamo/sqlnt?status.svg)](https://pkg.go.dev/github.com/go-andiamo/sqlnt)
[![Latest Version](https://img.shields.io/github/v/tag/go-andiamo/sqlnt.svg?sort=semver&style=flat&label=version&color=blue)](https://github.com/go-andiamo/sqlnt/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/go-andiamo/sqlnt)](https://goreportcard.com/report/github.com/go-andiamo/sqlnt)

## Overview

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
[try on go-playground](https://go.dev/play/p/RWwIqbV_mON)

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
[try on go-playground](https://go.dev/play/p/uLg4hy9Gvha)

## Installation
To install Sqlnt, use go get:

    go get github.com/go-andiamo/sqlnt

To update Sqlnt to the latest version, run:

    go get -u github.com/go-andiamo/sqlnt

## Enhanced Features

### Omissible args
By default, named templates check that all named args have been supplied...
```go
template := sqlnt.MustCreateNamedTemplate(`INSERT INTO table (col_a,col_b) VALUES (:a, :b)`, nil)
_, err := template.Args(map[string]any{"a": "a value"})
if err != nil {
    panic(err) // will panic because named arg "b" is missing
}
```
However, named args can be set as omissible...
```go
template := sqlnt.MustCreateNamedTemplate(`INSERT INTO table (col_a,col_b) VALUES (:a, :b)`, nil)
template.OmissibleArgs("b")
args, err := template.Args(map[string]any{"a": "a value"})
if err != nil {
    panic(err) // will not panic here because named arg "b" is missing but omissible
} else {
    fmt.Printf("%#v", args) // prints: []interface {}{"a value", interface {}(nil)}
}
```
Named args can also be set as omissible in the original template by suffixing the name with `?`...
```go
template := sqlnt.MustCreateNamedTemplate(`INSERT INTO table (col_a,col_b) VALUES (:a, :b?)`, nil)
args, err := template.Args(map[string]any{"a": "a value"})
if err != nil {
    panic(err) // will not panic here because named arg "b" is missing but omissible
} else {
    fmt.Printf("%#v", args) // prints: []interface {}{"a value", interface {}(nil)}
}
```
### Default values
Named templates also provides for default - where if a named arg is not supplied a default value is used...
```go
template := sqlnt.MustCreateNamedTemplate(`INSERT INTO table (name,status) VALUES (:name, :status)`, nil)
template.DefaultValue("status", "unknown")
args, err := template.Args(map[string]any{"name": "some name"})
if err != nil {
    panic(err) // will not panic here because named arg "status" is missing but defaulted
} else {
    fmt.Printf("%#v", args) // prints: []interface {}{"some name", "unknown"}
}
```
Default values can also be supplied as a function...
```go
template := sqlnt.MustCreateNamedTemplate(`INSERT INTO table (name,status,created_at) VALUES (:name, :status, :createdAt)`, nil)
template.DefaultValue("status", "unknown")
template.DefaultValue("createdAt", func(name string) any {
    return time.Now()
})
args, err := template.Args(map[string]any{"name": "some name"})
if err != nil {
    panic(err) // will not panic here because named args "status" and "createdAt" are missing but defaulted
} else {
    fmt.Printf("%#v", args) // prints: []interface {}{"some name", "unknown", time.Date{...}}
}
```

### Tokens
Sometimes you may have a common lexicon of table names, columns and/or arg names defined as consts.
These can be used in templates using token replace notation (`{{token}}`) in the template string and transposed by providing a `sqlnt.TokenOption` implementation...
```go
package main

import (
    "database/sql"
    "fmt"
    "github.com/go-andiamo/sqlnt"
    "time"
)

func main() {
    insertQueue := sqlnt.MustCreateNamedTemplate(InsertStatement, &Lexicon{TableNameQueues}).
        DefaultValue(ParamNameCreatedAt, nowFunc).DefaultValue(ParamNameStatus, "unknown")
    insertTopic := sqlnt.MustCreateNamedTemplate(InsertStatement, &Lexicon{TableNameTopics}).
        DefaultValue(ParamNameCreatedAt, nowFunc).DefaultValue(ParamNameStatus, "unknown")

    statement, args := insertQueue.MustStatementAndArgs(sql.Named(ParamNameName, "foo"))
    fmt.Printf("statement: %s\n    args: %#v\n", statement, args)

    statement, args = insertTopic.MustStatementAndArgs(sql.Named(ParamNameName, "foo"))
    fmt.Printf("statement: %s\n    args: %#v\n", statement, args)
}

const (
    InsertStatement    = "INSERT INTO {{table}} ({{baseCols}}) VALUES ({{insertArgs}})"
    TableNameQueues    = "queues"
    TableNameTopics    = "topics"
    BaseCols           = "name,status,created_at"
    ParamNameName      = "name"
    ParamNameStatus    = "status"
    ParamNameCreatedAt = "createdAt"
)

var nowFunc = func(name string) any {
    return time.Now()
}

var commonLexiconMap = map[string]string{
    "baseCols":   BaseCols,
    "insertArgs": ":" + ParamNameName + ",:" + ParamNameStatus + ",:" + ParamNameCreatedAt,
}

type Lexicon struct {
    TableName string
}

// Replace implements sqlnt.TokenOption.Replace
func (l *Lexicon) Replace(token string) (string, bool) {
    if token == "table" {
        return l.TableName, true
    }
    r, ok := commonLexiconMap[token]
    return r, ok
}
```
