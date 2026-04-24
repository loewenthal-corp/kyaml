package main

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const blockYAML = `apiVersion: v1
kind: Service
metadata:
  name: my-service
  labels:
    app: web
spec:
  type: ClusterIP
  ports:
    - port: 80
      protocol: TCP
      targetPort: 8080
  selector:
    app: web
`

func formatToKYAML(t *testing.T) string {
	t.Helper()
	f := writeTempFile(t, "input.yaml", blockYAML)
	cmd := &FormatCmd{Paths: []string{f}}
	out, _ := captureOutputs(t, func() error { return cmd.Run() })
	return out
}

func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	return p
}

func writeTempDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		p := filepath.Join(dir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
		require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	}
	return dir
}

func captureOutputs(t *testing.T, fn func() error) (stdout, stderr string) {
	t.Helper()

	oldOut := os.Stdout
	oldErr := os.Stderr

	rOut, wOut, err := os.Pipe()
	require.NoError(t, err)
	rErr, wErr, err := os.Pipe()
	require.NoError(t, err)

	os.Stdout = wOut
	os.Stderr = wErr

	// Read pipes concurrently to avoid deadlock on large output.
	outCh := make(chan []byte, 1)
	errCh := make(chan []byte, 1)
	go func() {
		b, _ := io.ReadAll(rOut)
		outCh <- b
	}()
	go func() {
		b, _ := io.ReadAll(rErr)
		errCh <- b
	}()

	fnErr := fn()
	_ = fnErr

	wOut.Close()
	wErr.Close()
	os.Stdout = oldOut
	os.Stderr = oldErr

	return string(<-outCh), string(<-errCh)
}

// --- Version command ---

func TestVersionCmd(t *testing.T) {
	cmd := &VersionCmd{}
	out, _ := captureOutputs(t, func() error { return cmd.Run() })
	assert.Contains(t, out, "0.")
}

// --- Format command: file ---

func TestFormatCmd_File(t *testing.T) {
	f := writeTempFile(t, "svc.yaml", blockYAML)
	cmd := &FormatCmd{Paths: []string{f}}

	out, _ := captureOutputs(t, func() error { return cmd.Run() })

	assert.Contains(t, out, "---")
	assert.Contains(t, out, `kind: "Service"`)
	assert.Contains(t, out, "{")
}

func TestFormatCmd_FileWrite(t *testing.T) {
	f := writeTempFile(t, "svc.yaml", blockYAML)
	cmd := &FormatCmd{Write: true, Paths: []string{f}}

	out, _ := captureOutputs(t, func() error { return cmd.Run() })

	assert.Contains(t, out, "svc.yaml")

	data, err := os.ReadFile(f)
	require.NoError(t, err)
	assert.Contains(t, string(data), "---")
	assert.Contains(t, string(data), "{")
}

func TestFormatCmd_FileWriteNoChange(t *testing.T) {
	kyamlContent := formatToKYAML(t)
	f := writeTempFile(t, "svc.yaml", kyamlContent)
	cmd := &FormatCmd{Write: true, Paths: []string{f}}

	out, _ := captureOutputs(t, func() error { return cmd.Run() })

	assert.Empty(t, out)
}

// --- Format command: directory ---

func TestFormatCmd_Dir(t *testing.T) {
	dir := writeTempDir(t, map[string]string{
		"a.yaml": blockYAML,
		"b.yml":  "name: test\n",
	})
	cmd := &FormatCmd{Paths: []string{dir}}

	out, _ := captureOutputs(t, func() error { return cmd.Run() })

	assert.Contains(t, out, "---")
	assert.Contains(t, out, `"Service"`)
	assert.Contains(t, out, `"test"`)
}

func TestFormatCmd_DirWrite(t *testing.T) {
	dir := writeTempDir(t, map[string]string{
		"a.yaml": blockYAML,
		"b.yml":  "name: test\n",
	})
	cmd := &FormatCmd{Write: true, Paths: []string{dir}}

	out, _ := captureOutputs(t, func() error { return cmd.Run() })

	assert.Contains(t, out, "a.yaml")
	assert.Contains(t, out, "b.yml")
}

// --- Format command: stdin ---

func TestFormatCmd_Stdin(t *testing.T) {
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdin = r

	go func() {
		w.Write([]byte(blockYAML))
		w.Close()
	}()

	cmd := &FormatCmd{}
	out, _ := captureOutputs(t, func() error { return cmd.Run() })
	os.Stdin = oldStdin

	assert.Contains(t, out, "---")
	assert.Contains(t, out, `kind: "Service"`)
}

// --- Validate command ---

func TestValidateCmd_ValidFile(t *testing.T) {
	kyamlContent := formatToKYAML(t)
	f := writeTempFile(t, "valid.yaml", kyamlContent)
	cmd := &ValidateCmd{Paths: []string{f}}

	out, _ := captureOutputs(t, func() error { return cmd.Run() })

	assert.Contains(t, out, "ok")
	assert.Contains(t, out, "valid.yaml")
}

func TestValidateCmd_InvalidFile(t *testing.T) {
	f := writeTempFile(t, "invalid.yaml", blockYAML)
	cmd := &ValidateCmd{Paths: []string{f}}

	var runErr error
	_, stderr := captureOutputs(t, func() error {
		runErr = cmd.Run()
		return runErr
	})

	assert.Error(t, runErr)
	assert.Contains(t, runErr.Error(), "validation failed")
	assert.Contains(t, stderr, "FAIL")
	assert.Contains(t, stderr, "invalid.yaml")
}

func TestValidateCmd_Dir_Mixed(t *testing.T) {
	kyamlContent := formatToKYAML(t)
	dir := writeTempDir(t, map[string]string{
		"good.yaml": kyamlContent,
		"bad.yaml":  blockYAML,
	})
	cmd := &ValidateCmd{Paths: []string{dir}}

	var runErr error
	stdout, stderr := captureOutputs(t, func() error {
		runErr = cmd.Run()
		return runErr
	})

	assert.Error(t, runErr)
	assert.Contains(t, stdout, "ok")
	assert.Contains(t, stdout, "good.yaml")
	assert.Contains(t, stderr, "FAIL")
	assert.Contains(t, stderr, "bad.yaml")
}

func TestValidateCmd_Dir_AllValid(t *testing.T) {
	kyamlContent := formatToKYAML(t)
	dir := writeTempDir(t, map[string]string{
		"a.yaml": kyamlContent,
		"b.yml":  kyamlContent,
	})
	cmd := &ValidateCmd{Paths: []string{dir}}

	out, _ := captureOutputs(t, func() error { return cmd.Run() })

	assert.Contains(t, out, "ok")
	assert.Contains(t, out, "a.yaml")
	assert.Contains(t, out, "b.yml")
}
