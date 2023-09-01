# crd-gates

Add feature gates to CRDs

```shell
$ crd-gates crd.yaml
```

It outputs the processed CRD manifest to stdout.

It does the following transformation of schemas:

```yaml
properties:
  foo:
    description: [[GATE:FeatureGateName]] THIS IS AN ALPHA FIELD. This is a foo field.
```

to

```yaml
properties:
  # {{- if .FeatureGateName }}
  foo:
    description: THIS IS AN ALPHA FIELD. This is a foo field.
  # {{- end }}
```
