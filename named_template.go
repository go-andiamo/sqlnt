package sqlnt

import (
	"context"
	"database/sql"
	"fmt"
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
	// NullableStringArgs specifies the names of args that are nullable string
	// i.e. where the value is an empty string, null is used instead
	NullableStringArgs(names ...string) NamedTemplate
	// GetArgNames returns a map of the arg names (where the map value is a bool indicating whether
	// the arg is omissible
	//
	// NB. Each arg is immutable - changing it has no effect on the template
	GetArgNames() map[string]bool
	// GetArgsInfo returns a map of each named arg info
	//
	// NB. Each arg info is immutable - changing it has no effect on the template
	GetArgsInfo() map[string]ArgInfo
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
	tokenOptions      []TokenOption
}

// NewNamedTemplate creates a new NamedTemplate
//
// # Returns an error if the supplied template cannot be parsed for arg names
//
// Multiple options can be specified - each must be either a sqlnt.Option or sqlnt.TokenOption
func NewNamedTemplate(statement string, options ...any) (NamedTemplate, error) {
	opt, tokenOptions, err := getOptions(options...)
	if err != nil {
		return nil, err
	}
	result := newNamedTemplate(statement, opt.UsePositionalTags(), opt.ArgTag(), tokenOptions)
	if err = result.buildArgs(); err != nil {
		return nil, err
	}
	return result, nil
}

// MustCreateNamedTemplate creates a new NamedTemplate
//
// is the same as NewNamedTemplate, except panics in case of error
func MustCreateNamedTemplate(statement string, options ...any) NamedTemplate {
	nt, err := NewNamedTemplate(statement, options...)
	if err != nil {
		panic(err)
	}
	return nt
}

func newNamedTemplate(statement string, usePositionalTags bool, argTag string, tokenOptions []TokenOption) *namedTemplate {
	return &namedTemplate{
		originalStatement: statement,
		args:              map[string]*namedArg{},
		usePositionalTags: usePositionalTags,
		argTag:            argTag,
		tokenOptions:      tokenOptions,
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
			av := arg.value(v)
			for _, posn := range arg.positions {
				out[posn] = av
			}
		} else if !arg.omissible {
			return nil, fmt.Errorf("named arg '%s' missing", name)
		} else if arg.defValue != nil {
			v = arg.defaultedValue(name)
			for _, posn := range arg.positions {
				out[posn] = v
			}
		}
	}
	return out, nil
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

// NullableStringArgs specifies the names of args that are nullable string
// i.e. where the value is an empty string, null is used instead
func (n *namedTemplate) NullableStringArgs(names ...string) NamedTemplate {
	for _, name := range names {
		if arg, ok := n.args[name]; ok {
			arg.nullableString = true
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

// GetArgsInfo returns a map of each named arg info
//
// NB. Each arg info is immutable - changing it has no effect on the template
func (n *namedTemplate) GetArgsInfo() map[string]ArgInfo {
	result := make(map[string]ArgInfo, len(n.args))
	for name, arg := range n.args {
		result[name] = arg.toInfo()
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
		r := newNamedTemplate(n.originalStatement, option.UsePositionalTags(), option.ArgTag(), n.tokenOptions)
		_ = r.buildArgs()
		for name, arg := range n.args {
			arg.copyOptionsTo(r.args[name])
		}
		return r
	}
}

// Append appends a statement portion to current statement and returns a new NamedTemplate
//
// Returns an error if the supplied statement portion cannot be parsed for arg names
func (n *namedTemplate) Append(portion string) (NamedTemplate, error) {
	result := newNamedTemplate(n.originalStatement+portion, n.usePositionalTags, n.argTag, n.tokenOptions)
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
