package main

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/alecthomas/kong"

	kyaml "github.com/loewenthal-corp/kyaml"
	"github.com/loewenthal-corp/kyaml/internal/format"
	"github.com/loewenthal-corp/kyaml/internal/validate"
)

var CLI struct {
	Format   FormatCmd   `cmd:"" aliases:"fmt" help:"Format YAML files to KYAML."`
	Validate ValidateCmd `cmd:"" help:"Validate that files are valid KYAML."`
	Version  VersionCmd  `cmd:"" help:"Print version."`
}

type FormatCmd struct {
	Write bool     `short:"w" help:"Write result back to files."`
	Paths []string `arg:"" optional:"" help:"Files or directories to format." type:"existingpath"`
}

func (c *FormatCmd) Run() error {
	if len(c.Paths) == 0 {
		input, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("reading stdin: %w", err)
		}
		out, err := format.Format(input)
		if err != nil {
			return err
		}
		_, err = os.Stdout.Write(out)
		return err
	}

	for _, path := range c.Paths {
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			results, err := format.FormatDir(path, c.Write)
			if err != nil {
				return err
			}
			for _, r := range results {
				if r.Err != nil {
					fmt.Fprintf(os.Stderr, "error: %s: %v\n", r.Path, r.Err)
					continue
				}
				if c.Write {
					if r.Changed {
						fmt.Println(r.Path)
					}
				} else {
					os.Stdout.Write(r.Output)
				}
			}
		} else {
			original, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			out, err := format.FormatFile(path, c.Write)
			if err != nil {
				return err
			}
			if c.Write {
				if !bytes.Equal(original, out) {
					fmt.Println(path)
				}
			} else {
				os.Stdout.Write(out)
			}
		}
	}
	return nil
}

type ValidateCmd struct {
	Paths []string `arg:"" help:"Files or directories to validate." type:"existingpath"`
}

func (c *ValidateCmd) Run() error {
	hasErrors := false

	for _, path := range c.Paths {
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			results, err := validate.ValidateDir(path)
			if err != nil {
				return err
			}
			for _, r := range results {
				if r.Err != nil {
					fmt.Fprintf(os.Stderr, "FAIL %s: %v\n", r.Path, r.Err)
					hasErrors = true
				} else {
					fmt.Fprintf(os.Stdout, "ok   %s\n", r.Path)
				}
			}
		} else {
			if err := validate.ValidateFile(path); err != nil {
				fmt.Fprintf(os.Stderr, "FAIL %s: %v\n", path, err)
				hasErrors = true
			} else {
				fmt.Fprintf(os.Stdout, "ok   %s\n", path)
			}
		}
	}

	if hasErrors {
		return fmt.Errorf("validation failed")
	}
	return nil
}

type VersionCmd struct{}

func (c *VersionCmd) Run() error {
	fmt.Println(kyaml.Version)
	return nil
}

func main() {
	k := kong.Parse(&CLI,
		kong.Name("kyaml"),
		kong.Description("KYAML formatter and validator — converts YAML to Kubernetes YAML (KYAML)."),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
	)
	k.FatalIfErrorf(k.Run())
}
