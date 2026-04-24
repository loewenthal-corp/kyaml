package format

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"sigs.k8s.io/yaml/kyaml"
)

// Result holds the outcome of formatting a single file.
type Result struct {
	Path    string
	Output  []byte
	Changed bool
	Err     error
}

// Format takes raw YAML bytes and converts them to KYAML format.
func Format(input []byte) ([]byte, error) {
	var buf bytes.Buffer
	enc := kyaml.Encoder{}
	if err := enc.FromYAML(bytes.NewReader(input), &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// FormatFile reads a file, formats it, and optionally writes back.
func FormatFile(path string, write bool) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	out, err := Format(data)
	if err != nil {
		return nil, err
	}

	if write && !bytes.Equal(data, out) {
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, out, info.Mode()); err != nil {
			return nil, err
		}
	}

	return out, nil
}

// FormatDir walks a directory finding YAML files and formats each one.
func FormatDir(dir string, write bool) ([]Result, error) {
	var results []Result

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() != "." && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}
		if d.IsDir() || !IsYAMLFile(path) {
			return nil
		}

		original, readErr := os.ReadFile(path)
		if readErr != nil {
			results = append(results, Result{Path: path, Err: readErr})
			return nil
		}

		out, fmtErr := FormatFile(path, write)
		results = append(results, Result{
			Path:    path,
			Output:  out,
			Changed: !bytes.Equal(original, out),
			Err:     fmtErr,
		})
		return nil
	})

	return results, err
}

// IsYAMLFile checks if a file has a .yaml or .yml extension.
func IsYAMLFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yaml" || ext == ".yml"
}
