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

// TokenOption is an interface that can be provided to NewNamedTemplate or MustCreateNamedTemplate
// to replace tokens in the statement (tokens are denoted by `{{token}}`)
//
// If tokens are found but none of the provided TokenOption implementations provides a replacement
// then NewNamedTemplate will error
type TokenOption interface {
	// Replace receives the token and returns the replacement and a bool indicating whether to use the replacement
	Replace(token string) (string, bool)
}

var (
	MySqlOption    Option = _MySqlOption    // option to produce final args like ?, ?, ? (e.g. for https://github.com/go-sql-driver/mysql)
	PostgresOption Option = _PostgresOption // option to produce final args like $1, $2, $3 (e.g. for https://github.com/lib/pq or https://github.com/jackc/pgx)
	DefaultsOption Option = _DefaultsOption // option to produce final args determined by DefaultUsePositionalTags and DefaultArgTag
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
