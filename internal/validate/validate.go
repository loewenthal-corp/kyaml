package validate

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/loewenthal-corp/kyaml/internal/format"
)

// ValidationResult holds the outcome of validating a single file.
type ValidationResult struct {
	Path  string
	Valid bool
	Err   error
}

// Validate returns nil if the input is already valid KYAML.
func Validate(input []byte) error {
	formatted, err := format.Format(input)
	if err != nil {
		return fmt.Errorf("failed to format: %w", err)
	}
	if !bytes.Equal(input, formatted) {
		return fmt.Errorf("not valid KYAML: formatting would change the document")
	}
	return nil
}

// ValidateFile reads and validates a file.
func ValidateFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return Validate(data)
}

// ValidateDir walks a directory and validates each YAML file.
func ValidateDir(dir string) ([]ValidationResult, error) {
	var results []ValidationResult

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() != "." && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}
		if d.IsDir() || !format.IsYAMLFile(path) {
			return nil
		}

		vErr := ValidateFile(path)
		results = append(results, ValidationResult{
			Path:  path,
			Valid: vErr == nil,
			Err:   vErr,
		})
		return nil
	})

	return results, err
}
