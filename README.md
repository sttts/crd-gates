# crd-gates

Add feature gates to CRDs

```shell
$ crd-gates crd.yaml
```

It output the processed CRD manifest to stdout.

It does the following transformations of schemas:

```yaml
properties:
  foo:
    description: [GATE:FeatureGateName] THIS IS AN ALPHA FIELD. This is a foo field.
```

to

```go-template
properties:
  # {{- if .FeatureGateName }}
  foo:
    description: THIS IS AN ALPHA FIELD. This is a foo field.
  # {{- end }}
```
