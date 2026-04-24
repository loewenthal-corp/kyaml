# kyaml

KYAML formatter and validator — converts YAML to [Kubernetes YAML (KYAML)](https://kubernetes.io/docs/reference/encodings/kyaml/), a safer and less ambiguous subset of YAML introduced in Kubernetes v1.34 ([KEP-5295](https://github.com/kubernetes/enhancements/blob/master/keps/sig-cli/5295-kyaml/README.md)).

## What is KYAML?

KYAML is a strict subset of YAML designed specifically for Kubernetes. Any KYAML document is valid YAML, but KYAML eliminates common pitfalls:

- **Double-quoted strings** — no more implicit type coercion (`NO` → `false`)
- **Flow-style syntax** — uses `{}` for maps and `[]` for lists, making it whitespace-insensitive
- **Comments** — unlike JSON, KYAML supports comments
- **Trailing commas** — allowed for cleaner diffs

## Install

```sh
go install github.com/loewenthal-corp/kyaml/cmd/kyaml@latest
```

Or download a binary from the [releases page](https://github.com/loewenthal-corp/kyaml/releases).

## Usage

### Format

Convert YAML files to KYAML:

```sh
# Format a file (print to stdout)
kyaml format deployment.yaml

# Format and write back
kyaml format -w deployment.yaml

# Format all YAML files in a directory
kyaml format -w ./manifests/

# Format from stdin
cat service.yaml | kyaml format
```

The `fmt` alias also works:

```sh
kyaml fmt -w .
```

### Validate

Check if files are already valid KYAML:

```sh
# Validate a file
kyaml validate deployment.yaml

# Validate all YAML files in a directory
kyaml validate ./manifests/
```

Exit code is `1` if any file is not valid KYAML.

## Example

Input (`service.yaml`):

```yaml
apiVersion: v1
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
```

Output (`kyaml format service.yaml`):

```yaml
---
{
  apiVersion: "v1",
  kind: "Service",
  metadata: {
    name: "my-service",
    labels: {
      app: "web",
    },
  },
  spec: {
    type: "ClusterIP",
    ports: [{
      port: 80,
      protocol: "TCP",
      targetPort: 8080,
    }],
    selector: {
      app: "web",
    },
  },
}
```

## Development

```sh
task do        # Run all quality checks
task test      # Run tests
task build     # Build binary
task install   # Install to ~/go/bin
```
