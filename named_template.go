package sqlnt

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

// NamedTemplate represents a named template
//
// Use NewNamedTemplate or MustCreateNamedTemplate to create a new one
type NamedTemplate interface {
	// Statement returns the sql statement to use (with named args transposed)
	Statement() string
	// StatementAndArgs returns the sql statement to use (with named args transposed) and
	// the input named args converted to positional args
	//
	// Essentially the same as calling Statement and then Args
	StatementAndArgs(args ...any) (string, []any, error)
	// MustStatementAndArgs is the same as StatementAndArgs, except no error is returned (and panics on error)
	MustStatementAndArgs(args ...any) (string, []any)
	// OriginalStatement returns the original named template statement
	OriginalStatement() string
	// Args converts the input named args to positional args (for use in db.Exec, db.Query etc.)
	//
	// Each arg in the supplied args can be:
	//
	// * map[string]any
	//
	// * sql.NamedArg
	//
	// * or any map where all keys are set as string
	//
	// * or anything that can be marshalled and then unmarshalled to map[string]any (such as structs!)
	//
	// If any of the named args specified in the query are missing, returns an error
	//
	// NB. named args are not considered missing when they have denoted as omissible (see NamedTemplate.OmissibleArgs) or
	// have been set with a default value (see NamedTemplate.DefaultValue)
	Args(args ...any) ([]any, error)
	// MustArgs is the same as Args, except no error is returned (and panics on error)
	MustArgs(args ...any) []any
	// ArgsCount returns the number of args that are passed into the statement
	ArgsCount() int
	// OmissibleArgs specifies the names of args that can be omitted
	//
	// Calling this without any names makes all args omissible
	//
	// Note: Named args can also be set as omissible in the template - example:
	//    tmp := sqlnt.MustCreateNamedTemplate(`INSERT INTO table (col_a,col_b) VALUES (:a, :b?)`)
	// makes the named arg "b" omissible (denoted by the '?' after name)
	OmissibleArgs(names ...string) NamedTemplate
	// DefaultValue specifies a value to be used for a given arg name when the arg
	// is not supplied in the map for Args or MustArgs
	//
	// Setting a default value for an arg name also makes that arg omissible
	//
	// If the value passed is a
	//   func(name string) any
	// then that func is called to obtain the default value
	DefaultValue(name string, v any) NamedTemplate
	// GetArgNames returns a map of the arg names (where the map value is a bool indicating whether
	// the arg is omissible
	GetArgNames() map[string]bool
	// Clone clones the named template to another with a different option
	Clone(option Option) NamedTemplate
	// Append appends a statement portion to current statement and returns a new NamedTemplate
	//
	// Returns an error if the supplied statement portion cannot be parsed for arg names
	Append(portion string) (NamedTemplate, error)
	// MustAppend is the same as Append, except no error is returned (and panics on error)
	MustAppend(portion string) NamedTemplate
	// Exec performs sql.DB.Exec on the supplied db with the supplied named args
	Exec(db *sql.DB, args ...any) (sql.Result, error)
	// ExecContext performs sql.DB.ExecContext on the supplied db with the supplied named args
	ExecContext(ctx context.Context, db *sql.DB, args ...any) (sql.Result, error)
	// Query performs sql.DB.Query on the supplied db with the supplied named args
	Query(db *sql.DB, args ...any) (*sql.Rows, error)
	// QueryContext performs sql.DB.QueryContext on the supplied db with the supplied named args
	QueryContext(ctx context.Context, db *sql.DB, args ...any) (*sql.Rows, error)
}

type namedTemplate struct {
	originalStatement string
	statement         string
	args              map[string]*namedArg
	argsCount         int
	usePositionalTags bool
	argTag            string
}

// NewNamedTemplate creates a new NamedTemplate
//
// Returns an error if the supplied template cannot be parsed for arg names
func NewNamedTemplate(statement string, option Option) (NamedTemplate, error) {
	if option == nil {
		option = DefaultsOption
	}
	result := newNamedTemplate(statement, option.UsePositionalTags(), option.ArgTag())
	if err := result.buildArgs(); err != nil {
		return nil, err
	}
	return result, nil
}

// MustCreateNamedTemplate creates a new NamedTemplate
//
// is the same as NewNamedTemplate, except panics in case of error
func MustCreateNamedTemplate(statement string, option Option) NamedTemplate {
	nt, err := NewNamedTemplate(statement, option)
	if err != nil {
		panic(err)
	}
	return nt
}

func newNamedTemplate(statement string, usePositionalTags bool, argTag string) *namedTemplate {
	return &namedTemplate{
		originalStatement: statement,
		args:              map[string]*namedArg{},
		usePositionalTags: usePositionalTags,
		argTag:            argTag,
	}
}

// Statement returns the sql statement to use (with named args transposed)
func (n *namedTemplate) Statement() string {
	return n.statement
}

// StatementAndArgs returns the sql statement to use (with named args transposed) and
// the input named args converted to positional args
//
// Essentially the same as calling Statement and then Args
func (n *namedTemplate) StatementAndArgs(args ...any) (string, []any, error) {
	rargs, err := n.Args(args...)
	return n.statement, rargs, err
}

// MustStatementAndArgs is the same as StatementAndArgs, except no error is returned (and panics on error)
func (n *namedTemplate) MustStatementAndArgs(args ...any) (string, []any) {
	rargs, err := n.Args(args...)
	if err != nil {
		panic(err)
	}
	return n.statement, rargs
}

// OriginalStatement returns the original named template statement
func (n *namedTemplate) OriginalStatement() string {
	return n.originalStatement
}

// Args converts the input named args to positional args (for use in db.Exec, db.Query etc.)
//
// Each arg in the supplied args can be:
//
// * map[string]any
//
// * sql.NamedArg
//
// * or any map where all keys are set as string
//
// * or anything that can be marshalled and then unmarshalled to map[string]any (such as structs!)
//
// # If any of the named args specified in the query are missing, returns an error
//
// NB. named args are not considered missing when they have denoted as omissible (see NamedTemplate.OmissibleArgs) or
// have been set with a default value (see NamedTemplate.DefaultValue)
func (n *namedTemplate) Args(args ...any) ([]any, error) {
	out := make([]any, n.argsCount)
	mapped, err := mappedArgs(args...)
	if err != nil {
		return nil, err
	}
	for name, arg := range n.args {
		if v, ok := mapped[name]; ok {
			for _, posn := range arg.positions {
				out[posn] = v
			}
		} else if !arg.omissible {
			return nil, fmt.Errorf("named arg '%s' missing", name)
		} else if arg.defValue != nil {
			v = arg.defValue(name)
			for _, posn := range arg.positions {
				out[posn] = v
			}
		}
	}
	return out, nil
}

func mappedArgs(args ...any) (map[string]any, error) {
	result := map[string]any{}
	for _, arg := range args {
		if arg != nil {
			switch targ := arg.(type) {
			case map[string]any:
				for k, v := range targ {
					result[k] = v
				}
			case *sql.NamedArg:
				result[targ.Name] = targ.Value
			case sql.NamedArg:
				result[targ.Name] = targ.Value
			default:
				if vo := reflect.ValueOf(arg); vo.Kind() == reflect.Map {
					// it's a map, but not a map[string]any...
					iter := vo.MapRange()
					for iter.Next() {
						if k, ok := iter.Key().Interface().(string); ok {
							result[k] = iter.Value().Interface()
						} else {
							return nil, errors.New("invalid map - keys must be string")
						}
					}
				} else {
					// not a type aware of - try marshaling and then unmarshalling to a map...
					if data, err := json.Marshal(arg); err == nil {
						var jm map[string]any
						if err := json.Unmarshal(data, &jm); err == nil {
							for k, v := range jm {
								result[k] = v
							}
						} else {
							return nil, err
						}
					} else {
						return nil, err
					}
				}
			}
		}
	}
	return result, nil
}

// MustArgs is the same as Args, except no error is returned (and panics on error)
func (n *namedTemplate) MustArgs(args ...any) []any {
	out, err := n.Args(args...)
	if err != nil {
		panic(err)
	}
	return out
}

// ArgsCount returns the number of args that are passed into the statement
func (n *namedTemplate) ArgsCount() int {
	return n.argsCount
}

// OmissibleArgs specifies the names of args that can be omitted
//
// # Calling this without any names makes all args omissible
//
// Note: Named args can also be set as omissible in the template - example:
//
//	tmp := sqlnt.MustCreateNamedTemplate(`INSERT INTO table (col_a,col_b) VALUES (:a, :b?)`)
//
// makes the named arg "b" omissible (denoted by the '?' after name)
func (n *namedTemplate) OmissibleArgs(names ...string) NamedTemplate {
	if len(names) == 0 {
		for _, arg := range n.args {
			arg.omissible = true
		}
	} else {
		for _, name := range names {
			if arg, ok := n.args[name]; ok {
				arg.omissible = true
			}
		}
	}
	return n
}

// DefaultValueFunc is the function signature for funcs that can be passed to
// NamedTemplate.DefaultValue
type DefaultValueFunc func(name string) any

// DefaultValue specifies a value to be used for a given arg name when the arg
// is not supplied in the map for Args or MustArgs
//
// # Setting a default value for an arg name also makes that arg omissible
//
// If the value passed is a
//
//	func(name string) any
//
// then that func is called to obtain the default value
func (n *namedTemplate) DefaultValue(name string, v interface{}) NamedTemplate {
	if arg, ok := n.args[name]; ok {
		arg.omissible = true
		if dvf, ok := v.(func(name string) any); ok {
			arg.defValue = dvf
		} else {
			arg.defValue = func(name string) any {
				return v
			}
		}
	}
	return n
}

// GetArgNames returns a map of the arg names (where the map value is a bool indicating whether
// the arg is omissible
func (n *namedTemplate) GetArgNames() map[string]bool {
	result := make(map[string]bool, len(n.args))
	for name, arg := range n.args {
		result[name] = arg.omissible
	}
	return result
}

// Clone clones the named template to another with a different option
func (n *namedTemplate) Clone(option Option) NamedTemplate {
	if option == nil {
		option = DefaultsOption
	}
	if option.UsePositionalTags() == n.usePositionalTags && option.ArgTag() == n.argTag {
		// no material change, just copy everything...
		return n.copy()
	} else {
		r := newNamedTemplate(n.originalStatement, option.UsePositionalTags(), option.ArgTag())
		_ = r.buildArgs()
		for name, arg := range n.args {
			if rarg, ok := r.args[name]; ok {
				rarg.omissible = arg.omissible
				rarg.defValue = arg.defValue
			}
		}
		return r
	}
}

// Append appends a statement portion to current statement and returns a new NamedTemplate
//
// Returns an error if the supplied statement portion cannot be parsed for arg names
func (n *namedTemplate) Append(portion string) (NamedTemplate, error) {
	result := newNamedTemplate(n.originalStatement+portion, n.usePositionalTags, n.argTag)
	if err := result.buildArgs(); err != nil {
		return nil, err
	}
	for name, arg := range n.args {
		if rarg, ok := result.args[name]; ok {
			rarg.omissible = arg.omissible
			rarg.defValue = arg.defValue
		}
	}
	return result, nil
}

// MustAppend is the same as Append, except no error is returned (and panics on error)
func (n *namedTemplate) MustAppend(portion string) NamedTemplate {
	if result, err := n.Append(portion); err == nil {
		return result
	} else {
		panic(err)
	}
}

func (n *namedTemplate) copy() *namedTemplate {
	r := newNamedTemplate(n.originalStatement, n.usePositionalTags, n.argTag)
	r.statement = n.statement
	r.argsCount = n.argsCount
	r.usePositionalTags = n.usePositionalTags
	r.argTag = n.argTag
	for name, arg := range n.args {
		r.args[name] = arg.clone()
	}
	return r
}

// Exec performs sql.DB.Exec on the supplied db with the supplied named args
func (n *namedTemplate) Exec(db *sql.DB, args ...any) (sql.Result, error) {
	if qargs, err := n.Args(args...); err == nil {
		return db.Exec(n.statement, qargs...)
	} else {
		return nil, err
	}
}

// ExecContext performs sql.DB.ExecContext on the supplied db with the supplied named args
func (n *namedTemplate) ExecContext(ctx context.Context, db *sql.DB, args ...any) (sql.Result, error) {
	if qargs, err := n.Args(args...); err == nil {
		return db.ExecContext(ctx, n.statement, qargs...)
	} else {
		return nil, err
	}
}

// Query performs sql.DB.Query on the supplied db with the supplied named args
func (n *namedTemplate) Query(db *sql.DB, args ...any) (*sql.Rows, error) {
	if qargs, err := n.Args(args...); err == nil {
		return db.Query(n.statement, qargs...)
	} else {
		return nil, err
	}
}

// QueryContext performs sql.DB.QueryContext on the supplied db with the supplied named args
func (n *namedTemplate) QueryContext(ctx context.Context, db *sql.DB, args ...any) (*sql.Rows, error) {
	if qargs, err := n.Args(args...); err == nil {
		return db.QueryContext(ctx, n.statement, qargs...)
	} else {
		return nil, err
	}
}
