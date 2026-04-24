package validate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/loewenthal-corp/kyaml/internal/format"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	t.Parallel()

	t.Run("valid KYAML returns nil", func(t *testing.T) {
		t.Parallel()
		kyaml, err := format.Format([]byte("name: hello\n"))
		require.NoError(t, err)

		assert.NoError(t, Validate(kyaml))
	})

	t.Run("block-style YAML returns error", func(t *testing.T) {
		t.Parallel()
		err := Validate([]byte("name: hello\n"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not valid KYAML")
	})

	t.Run("invalid YAML returns error", func(t *testing.T) {
		t.Parallel()
		err := Validate([]byte(":\n  :\n    - :\n  bad: ["))
		assert.Error(t, err)
	})
}

func TestValidateFile(t *testing.T) {
	t.Parallel()

	t.Run("valid KYAML file", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, "valid.yaml")

		kyaml, err := format.Format([]byte("name: hello\n"))
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(path, kyaml, 0o644))

		assert.NoError(t, ValidateFile(path))
	})

	t.Run("regular YAML file returns error", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, "invalid.yaml")
		require.NoError(t, os.WriteFile(path, []byte("name: hello\n"), 0o644))

		assert.Error(t, ValidateFile(path))
	})
}

func TestValidateDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Write a valid KYAML file.
	kyaml, err := format.Format([]byte("name: hello\n"))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "valid.yaml"), kyaml, 0o644))

	// Write an invalid (block-style) YAML file.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "invalid.yml"), []byte("name: hello\n"), 0o644))

	// Write a non-YAML file that should be ignored.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hi"), 0o644))

	results, err := ValidateDir(dir)
	require.NoError(t, err)
	assert.Len(t, results, 2)

	byName := map[string]ValidationResult{}
	for _, r := range results {
		byName[filepath.Base(r.Path)] = r
	}

	assert.True(t, byName["valid.yaml"].Valid)
	assert.NoError(t, byName["valid.yaml"].Err)

	assert.False(t, byName["invalid.yml"].Valid)
	assert.Error(t, byName["invalid.yml"].Err)
}
