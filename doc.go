// Package sqlnt - a Go package for sql named arg templates
//
/*
Provides a simple, driver agnostic, way to use SQL statement templates with named args

Example:
  var tmp = sqlnt.MustCreateNamedTemplate(`INSERT INTO table (col_a, col_b) VALUES(:a, :b)`, nil)

  args := map[string]any{
    "a": "a value",
    "b": "b value",
  }
  _, _ = db.Exec(tmp.Statement(), tmp.MustArgs(args)...)
*/
package sqlnt
