package sqlnt

import (
	"fmt"
	"strconv"
	"strings"
)

var DefaultUsePositionalTags = false
var DefaultParamTag = "?"

// NamedTemplate represents a named template
//
// Use NewNamedTemplate or MustCreateNamedTemplate to create a new one
type NamedTemplate interface {
	// Statement returns the sql statement to use (with named args transposed)
	Statement() string
	// OriginalStatement returns the original named template statement
	OriginalStatement() string
	// Args converts the input map args to positional args (for use in sql.Exec etc.)
	Args(in map[string]any) ([]any, error)
	// MustArgs is the same as Args, except no error is returned (and panics on error)
	MustArgs(in map[string]any) []any
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
}

type namedTemplate struct {
	originalStatement string
	statement         string
	argPositions      map[string][]int
	argsCount         int
	omissibleArgs     map[string]bool
	defaultValues     map[string]DefaultValueFunc
	usePositionalTags bool
	paramTag          string
}

// NewNamedTemplate creates a new NamedTemplate
//
// Returns an error if the supplied template cannot be parsed for arg names
func NewNamedTemplate(statement string, option Option) (NamedTemplate, error) {
	if option == nil {
		option = defaultedOption
	}
	result := &namedTemplate{
		originalStatement: statement,
		argPositions:      map[string][]int{},
		omissibleArgs:     map[string]bool{},
		defaultValues:     map[string]DefaultValueFunc{},
		usePositionalTags: option.UsePositionalTags(),
		paramTag:          option.ArgTag(),
	}
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
				builder.WriteString(n.addParamName(name))
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

func (n *namedTemplate) addParamName(name string) string {
	if n.usePositionalTags {
		if posns, ok := n.argPositions[name]; ok {
			return n.paramTag + strconv.Itoa(posns[0]+1)
		} else {
			n.argPositions[name] = []int{n.argsCount}
			n.argsCount++
			return n.paramTag + strconv.Itoa(n.argsCount)
		}
	} else {
		n.argPositions[name] = append(n.argPositions[name], n.argsCount)
		n.argsCount++
		return n.paramTag
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

// Args converts the input map args to positional args (for use in sql.Exec etc.)
func (n *namedTemplate) Args(in map[string]any) ([]any, error) {
	out := make([]any, n.argsCount)
	for name, posns := range n.argPositions {
		if v, ok := in[name]; ok {
			for _, posn := range posns {
				out[posn] = v
			}
		} else if !n.omissibleArgs[name] {
			return nil, fmt.Errorf("named param '%s' missing", name)
		} else if dvf, ok := n.defaultValues[name]; ok {
			v = dvf(name)
			for _, posn := range posns {
				out[posn] = v
			}
		}
	}
	return out, nil
}

// MustArgs is the same as Args, except no error is returned (and panics on error)
func (n *namedTemplate) MustArgs(in map[string]any) []any {
	out, err := n.Args(in)
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

// Option is the interface that can be passed to NewNamedTemplate or MustCreateNamedTemplate
// and determines whether positional tags (i.e. numbered tags) can be used and the arg placeholder to be used
type Option interface {
	// UsePositionalTags specifies whether positional arg tags (e.g. $1, $2 etc.) can be used in the
	// final sql statement
	UsePositionalTags() bool
	// ArgTag specifies the string used as the arg placeholder in the final sql statement
	//
	// e.g. return "?" for MySql or "$" for Postgres
	ArgTag() string
}

var (
	MySqlOption    Option = _MySqlOption
	PostgresOption Option = _PostgresOption
)

var (
	_MySqlOption = &option{
		usePositionalTags: false,
		paramTag:          "?",
	}
	_PostgresOption = &option{
		usePositionalTags: true,
		paramTag:          "$",
	}
)

type option struct {
	usePositionalTags bool
	paramTag          string
}

func (d *option) UsePositionalTags() bool {
	return d.usePositionalTags
}

func (d *option) ArgTag() string {
	return d.paramTag
}

var defaultedOption Option = &defaultOption{}

type defaultOption struct {
}

func (d *defaultOption) UsePositionalTags() bool {
	return DefaultUsePositionalTags
}

func (d *defaultOption) ArgTag() string {
	return DefaultParamTag
}
