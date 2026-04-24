package format

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple key-value",
			input:    "name: hello\n",
			expected: "---\n{\n  name: \"hello\",\n}\n",
		},
		{
			name:     "nested maps use braces",
			input:    "parent:\n  child: value\n",
			expected: "---\n{\n  parent: {\n    child: \"value\",\n  },\n}\n",
		},
		{
			name:     "lists use brackets",
			input:    "items:\n  - one\n  - two\n",
			expected: "---\n{\n  items: [\n    \"one\",\n    \"two\",\n  ],\n}\n",
		},
		{
			name:     "norway problem - NO is quoted",
			input:    "country: NO\n",
			expected: "---\n{\n  country: \"NO\",\n}\n",
		},
		{
			name:     "boolean values are not quoted",
			input:    "enabled: true\n",
			expected: "---\n{\n  enabled: true,\n}\n",
		},
		{
			name:     "integer values are not quoted",
			input:    "count: 42\n",
			expected: "---\n{\n  count: 42,\n}\n",
		},
		{
			name:     "float values are not quoted",
			input:    "pi: 3.14\n",
			expected: "---\n{\n  pi: 3.14,\n}\n",
		},
		{
			name:     "multi-document YAML",
			input:    "name: a\n---\nname: b\n",
			expected: "---\n{\n  name: \"a\",\n}\n---\n{\n  name: \"b\",\n}\n",
		},
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := Format([]byte(tt.input))
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(got))
		})
	}
}

func TestFormat_Idempotent(t *testing.T) {
	t.Parallel()

	inputs := []string{
		"name: hello\n",
		"parent:\n  child: value\n",
		"items:\n  - one\n  - two\n",
		"country: NO\n",
		"enabled: true\ncount: 42\n",
		"name: a\n---\nname: b\n",
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			t.Parallel()
			first, err := Format([]byte(input))
			require.NoError(t, err)

			second, err := Format(first)
			require.NoError(t, err)

			assert.Equal(t, string(first), string(second), "formatting should be idempotent")
		})
	}
}

func TestFormatFile_NoWrite(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")
	original := []byte("name: hello\n")
	require.NoError(t, os.WriteFile(path, original, 0o644))

	out, err := FormatFile(path, false)
	require.NoError(t, err)
	assert.Contains(t, string(out), "---")
	assert.Contains(t, string(out), "{")

	// File should not have been modified.
	onDisk, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, original, onDisk)
}

func TestFormatFile_Write(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")
	require.NoError(t, os.WriteFile(path, []byte("name: hello\n"), 0o644))

	out, err := FormatFile(path, true)
	require.NoError(t, err)

	onDisk, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, out, onDisk, "file should match formatter output after write")
}

func TestFormatFile_WriteIdempotent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")
	require.NoError(t, os.WriteFile(path, []byte("name: hello\n"), 0o644))

	// Format once with write.
	first, err := FormatFile(path, true)
	require.NoError(t, err)

	// Format again with write on already-formatted content.
	second, err := FormatFile(path, true)
	require.NoError(t, err)
	assert.Equal(t, first, second, "re-formatting should not change output")

	onDisk, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, first, onDisk)
}

func TestFormatDir_OnlyYAMLFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	yamlContent := []byte("name: hello\n")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.yaml"), yamlContent, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.yml"), yamlContent, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "c.txt"), []byte("not yaml"), 0o644))

	results, err := FormatDir(dir, false)
	require.NoError(t, err)

	var paths []string
	for _, r := range results {
		require.NoError(t, r.Err)
		paths = append(paths, filepath.Base(r.Path))
	}
	assert.ElementsMatch(t, []string{"a.yaml", "b.yml"}, paths)
}

func TestFormatDir_SkipsHiddenDirs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	hidden := filepath.Join(dir, ".hidden")
	require.NoError(t, os.Mkdir(hidden, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(hidden, "secret.yaml"), []byte("key: val\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "visible.yaml"), []byte("key: val\n"), 0o644))

	results, err := FormatDir(dir, false)
	require.NoError(t, err)

	assert.Len(t, results, 1)
	assert.Equal(t, "visible.yaml", filepath.Base(results[0].Path))
}

func TestFormatDir_ChangedFlag(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	unformatted := []byte("name: hello\n")
	formatted, err := Format(unformatted)
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "unformatted.yaml"), unformatted, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "formatted.yaml"), formatted, 0o644))

	results, err := FormatDir(dir, false)
	require.NoError(t, err)
	assert.Len(t, results, 2)

	changed := map[string]bool{}
	for _, r := range results {
		require.NoError(t, r.Err)
		changed[filepath.Base(r.Path)] = r.Changed
	}
	assert.True(t, changed["unformatted.yaml"], "unformatted file should be marked as changed")
	assert.False(t, changed["formatted.yaml"], "already-formatted file should not be marked as changed")
}

func TestIsYAMLFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path string
		want bool
	}{
		{"file.yaml", true},
		{"file.yml", true},
		{"file.YAML", true},
		{"file.YML", true},
		{"file.json", false},
		{"file.txt", false},
		{"file", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, IsYAMLFile(tt.path))
		})
	}
}
