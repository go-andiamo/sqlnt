package sqlnt

// DefaultUsePositionalTags is the default setting for whether to use positional arg tags
var DefaultUsePositionalTags = false

// DefaultArgTag is the default setting for arg tag placeholders
var DefaultArgTag = "?"

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
	DefaultsOption Option = _DefaultsOption
)

var (
	_MySqlOption = &option{
		usePositionalTags: false,
		argTag:            "?",
	}
	_PostgresOption = &option{
		usePositionalTags: true,
		argTag:            "$",
	}
	_DefaultsOption = &defaultOption{}
)

type option struct {
	usePositionalTags bool
	argTag            string
}

func (d *option) UsePositionalTags() bool {
	return d.usePositionalTags
}

func (d *option) ArgTag() string {
	return d.argTag
}

type defaultOption struct {
}

func (d *defaultOption) UsePositionalTags() bool {
	return DefaultUsePositionalTags
}

func (d *defaultOption) ArgTag() string {
	return DefaultArgTag
}
