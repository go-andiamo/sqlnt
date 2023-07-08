package sqlnt

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// NamedTemplate represents a named template
//
// Use NewNamedTemplate or MustCreateNamedTemplate to create a new one
type NamedTemplate interface {
	// Statement returns the sql statement to use (with named args transposed)
	Statement() string
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
	// Calling this without any names makes are args omissible
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
	argPositions      map[string][]int
	argsCount         int
	omissibleArgs     map[string]bool
	defaultValues     map[string]DefaultValueFunc
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
		argPositions:      map[string][]int{},
		omissibleArgs:     map[string]bool{},
		defaultValues:     map[string]DefaultValueFunc{},
		usePositionalTags: usePositionalTags,
		argTag:            argTag,
	}
}

func (n *namedTemplate) buildArgs() error {
	var builder strings.Builder
	n.argsCount = 0
	lastPos := 0
	runes := []rune(n.originalStatement)
	purge := func(pos int) {
		if lastPos != -1 && pos > lastPos {
			builder.WriteString(string(runes[lastPos:pos]))
		}
	}
	getNamed := func(pos int) (string, int, error) {
		i := pos + 1
		skip := 0
		for ; i < len(runes); i++ {
			if !isNameRune(runes[i]) {
				break
			}
			skip++
		}
		if skip == 0 {
			return "", 0, fmt.Errorf("named marker ':' without name (at position %d)", pos)
		}
		return string(runes[pos+1 : i]), skip, nil
	}
	for pos := 0; pos < len(runes); pos++ {
		if runes[pos] == ':' {
			purge(pos)
			if (pos+1) < len(runes) && runes[pos+1] == ':' {
				// double escaped name marker...
				pos++
				lastPos = pos
			} else {
				name, skip, err := getNamed(pos)
				if err != nil {
					return err
				}
				pos += skip
				lastPos = pos + 1
				builder.WriteString(n.addNamedArg(name))
			}
		}
	}
	purge(len(runes))
	n.statement = builder.String()
	return nil
}

func isNameRune(r rune) bool {
	return r == '_' || r == '-' || (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func (n *namedTemplate) addNamedArg(name string) string {
	if n.usePositionalTags {
		if posns, ok := n.argPositions[name]; ok {
			return n.argTag + strconv.Itoa(posns[0]+1)
		} else {
			n.argPositions[name] = []int{n.argsCount}
			n.argsCount++
			return n.argTag + strconv.Itoa(n.argsCount)
		}
	} else {
		n.argPositions[name] = append(n.argPositions[name], n.argsCount)
		n.argsCount++
		return n.argTag
	}
}

// Statement returns the sql statement to use (with named args transposed)
func (n *namedTemplate) Statement() string {
	return n.statement
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
	in, err := inArgs(args...)
	if err != nil {
		return nil, err
	}
	for name, posns := range n.argPositions {
		if v, ok := in[name]; ok {
			for _, posn := range posns {
				out[posn] = v
			}
		} else if !n.omissibleArgs[name] {
			return nil, fmt.Errorf("named arg '%s' missing", name)
		} else if dvf, ok := n.defaultValues[name]; ok {
			v = dvf(name)
			for _, posn := range posns {
				out[posn] = v
			}
		}
	}
	return out, nil
}

func inArgs(args ...any) (map[string]any, error) {
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
// Calling this without any names makes are args omissible
func (n *namedTemplate) OmissibleArgs(names ...string) NamedTemplate {
	if len(names) == 0 {
		for name := range n.argPositions {
			n.omissibleArgs[name] = true
		}
	} else {
		for _, name := range names {
			n.omissibleArgs[name] = true
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
	n.omissibleArgs[name] = true
	if dvf, ok := v.(func(name string) any); ok {
		n.defaultValues[name] = dvf
	} else {
		n.defaultValues[name] = func(name string) any {
			return v
		}
	}
	return n
}

// GetArgNames returns a map of the arg names (where the map value is a bool indicating whether
// the arg is omissible
func (n *namedTemplate) GetArgNames() map[string]bool {
	result := make(map[string]bool, len(n.argPositions))
	for name := range n.argPositions {
		result[name] = n.omissibleArgs[name]
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
		n.copyAdditionsTo(r)
		return r
	}
}

func (n *namedTemplate) copy() *namedTemplate {
	r := newNamedTemplate(n.originalStatement, n.usePositionalTags, n.argTag)
	r.statement = n.statement
	r.argsCount = n.argsCount
	r.usePositionalTags = n.usePositionalTags
	r.argTag = n.argTag
	for name, v := range n.argPositions {
		r.argPositions[name] = v
	}
	n.copyAdditionsTo(r)
	return r
}

func (n *namedTemplate) copyAdditionsTo(r *namedTemplate) {
	for name, v := range n.omissibleArgs {
		r.omissibleArgs[name] = v
	}
	for name, v := range n.defaultValues {
		r.defaultValues[name] = v
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
