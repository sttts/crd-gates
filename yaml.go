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
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

type VisitorFunc func(key, n *yaml.Node, schema *apiextensionsv1.JSONSchemaProps, path string)

func findNode(yamlNode *yaml.Node, name string) (*yaml.Node, *yaml.Node) {
	if yamlNode.Kind != yaml.MappingNode {
		return nil, nil
	}
	for i := 0; i < len(yamlNode.Content); i += 2 {
		keyNode := yamlNode.Content[i]
		valueNode := yamlNode.Content[i+1]
		if keyNode.Value == name {
			return keyNode, valueNode
		}
	}
	return nil, nil
}

func iterateSchema(k, n *yaml.Node, schemaProps *apiextensionsv1.JSONSchemaProps, visitor VisitorFunc, path []string) {
	visitor(k, n, schemaProps, strings.Join(path, "."))

	if len(schemaProps.Properties) > 0 {
		_, propertiesNode := findNode(n, "properties")
		if propertiesNode != nil {
			for key, prop := range schemaProps.Properties {
				keyNode, subNode := findNode(propertiesNode, key)
				if subNode != nil {
					newPath := append(path, "properties", key)
					iterateSchema(keyNode, subNode, &prop, visitor, newPath)
				}
			}
		}
	}

	if schemaProps.Items != nil && schemaProps.Items.Schema != nil {
		_, itemsNode := findNode(n, "items")
		if itemsNode != nil {
			if itemsNode.Kind == yaml.SequenceNode {
				for i, itemNode := range itemsNode.Content {
					newPath := append(path, fmt.Sprintf("items[%d]", i))
					iterateSchema(itemNode, itemNode, schemaProps.Items.Schema, visitor, newPath)
				}
			}
		}
	}
}

// FindByJSONPath traverses a yaml.Node tree to find a node corresponding to the given JSONPath.
func findByJSONPath(root *yaml.Node, jsonPath string) (*yaml.Node, error) {
	// Tokenize the JSONPath string
	tokens := strings.Split(jsonPath, ".")
	currentNode := root

	if root.Kind == yaml.DocumentNode {
		return nil, fmt.Errorf("root node is a document node")
	}

	// Traverse the tree based on tokens
	for _, token := range tokens {
		if token == "" {
			continue
		}

		found := false
		for i := 0; i < len(currentNode.Content); i += 2 {
			keyNode := currentNode.Content[i]
			valueNode := currentNode.Content[i+1]

			// Check if the current key node matches the token
			if keyNode.Value == token {
				currentNode = valueNode
				found = true
				break
			}

			// Handle array indices
			if keyNode.Kind == yaml.SequenceNode {
				if idx, err := strconv.Atoi(token); err == nil && idx < len(keyNode.Content) {
					currentNode = keyNode.Content[idx]
					found = true
					break
				}
			}
		}

		if !found {
			return nil, fmt.Errorf("path element not found: %s", token)
		}
	}

	return currentNode, nil
}
