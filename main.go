/*
Copyright 2023 Stefan Schimanski.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"os"
	"regexp"

	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	kyaml "sigs.k8s.io/yaml"
)

func main() {
	outputFile := pflag.StringP("output", "o", "", "Output file. If not specified, output to stdout.")
	help := pflag.BoolP("help", "h", false, "Print help.")
	pflag.Parse()
	if *help {
		pflag.Usage()
		os.Exit(0)
	}

	if len(pflag.Args()) != 1 {
		panic("Usage: crd-gates <file>")
	}
	bs, err := os.ReadFile(pflag.Args()[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read file: %s\n", err)
		os.Exit(1)
	}

	var root yaml.Node
	if err := yaml.Unmarshal(bs, &root); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse YAML: %s\n", err)
		os.Exit(1)
	}

	for _, doc := range root.Content {
		processDoc(doc)
	}

	out := os.Stdout
	if *outputFile != "" {
		out, err = os.Create(*outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create output file: %s\n", err)
			os.Exit(1)
		}
		defer out.Close()
	}
	for _, doc := range root.Content {
		bs, err = yaml.Marshal(doc)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to marshal YAML: %s\n", err)
			os.Exit(1)
		}

		fmt.Fprintln(out, "---")
		fmt.Fprintln(out, string(bs))
	}
}

var markerRE = regexp.MustCompile(`\[\[GATE:(.*?)]]\s*(.*)`)

func processDoc(doc *yaml.Node) {
	versions, err := findByJSONPath(doc, "spec.versions")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to find spec.versions: %s\n", err)
		os.Exit(1)
	}

	for _, version := range versions.Content {
		_, nameNode := findNode(version, "name")
		if nameNode == nil {
			continue
		}
		if nameNode.Kind != yaml.ScalarNode || nameNode.Tag != "!!str" {
			continue
		}
		name := nameNode.Value

		schema, err := findByJSONPath(version, "schema.openAPIV3Schema")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to find schema for version %q: %s\n", name, err)
			continue
		}

		bs, err := yaml.Marshal(schema)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to marshal schema for version %q: %s\n", name, err)
			os.Exit(1)
		}
		var props apiextensionsv1.JSONSchemaProps
		if err := kyaml.Unmarshal(bs, &props); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to unmarshal schema for version %q: %s\n", name, err)
			os.Exit(1)
		}

		iterateSchema(nil, schema, &props, func(k, n *yaml.Node, schema *apiextensionsv1.JSONSchemaProps, path string) {
			matches := markerRE.FindStringSubmatch(schema.Description)
			if matches != nil {
				gateName := matches[1]
				text := matches[2]

				fmt.Printf("spec.versions[%s].openAPIV3Schema.%s: %s\n", name, path, gateName)
				_, desc := findNode(n, "description")
				desc.Value = text
				k.HeadComment = fmt.Sprintf("{{- if .%s }}", gateName)
				k.FootComment = "{{- end }}"
			}
		}, nil)
	}
}
