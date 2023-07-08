package sqlnt

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDefaults(t *testing.T) {
	assert.False(t, DefaultUsePositionalTags)
	assert.Equal(t, "?", DefaultArgTag)

	assert.False(t, DefaultsOption.UsePositionalTags())
	assert.Equal(t, "?", DefaultsOption.ArgTag())
}

func TestMySqlOption(t *testing.T) {
	assert.False(t, MySqlOption.UsePositionalTags())
	assert.Equal(t, "?", MySqlOption.ArgTag())
}

func TestPostgresOption(t *testing.T) {
	assert.True(t, PostgresOption.UsePositionalTags())
	assert.Equal(t, "$", PostgresOption.ArgTag())
}
