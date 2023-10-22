package main

import (
	_ "embed"
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
)

//go:embed example.json
var exampleJSON []byte

func main() {
	/*
		var spec openapi3.Spec

		if err := spec.UnmarshalJSON(exampleJSON); err != nil {
			log.Panicf(`fail to unmarshal JSON: %+v`, err)
		}
		c := spew.NewDefaultConfig()
		c.DisableCapacities = true
		c.DisablePointerAddresses = true
		c.Dump(spec)
	*/
	loader := openapi3.NewLoader()
	doc, _ := loader.LoadFromData(exampleJSON)
	//

	_ = doc.Validate(loader.Context)
	TraverseOperations(doc, func(path, method string, operation *openapi3.Operation) error {
		fmt.Printf("---------- %s %s: %s ----------\n", method, path, operation.OperationID)
		for _, param := range operation.Parameters {
			for _, content := range param.Value.Content {
				fmt.Printf("param: %s: %T($ref=%s)\n", param.Value.Name, content.Schema.Value, param.Value.Schema.Ref)
			}
			fmt.Printf("param: %s: %T($ref=%s)\n", param.Value.Name, param.Value.Schema.Value, param.Value.Schema.Ref)
		}
		if operation.RequestBody != nil {
			for _, content := range operation.RequestBody.Value.Content {
				fmt.Printf("req: %T($ref=%s)\n", content.Schema.Value, content.Schema.Ref)
			}
		}
		for code, res := range operation.Responses {
			for _, content := range res.Value.Content {
				fmt.Printf("res: %s: %T($ref=%s)\n", code, content.Schema.Value, content.Schema.Ref)
			}
		}
		return nil
	})
}

func TraverseOperations(doc *openapi3.T, f func(path string, method string, operation *openapi3.Operation) error) error {
	for path, item := range doc.Paths {
		for method, operation := range item.Operations() {
			if err := f(path, method, operation); err != nil {
				return err
			}
		}
	}
	return nil
}

func WalkSchemas(doc *openapi3.T, f func(schema *openapi3.SchemaRef) error) error {
	referenced := map[string]bool{}
	return TraverseOperations(doc, func(path, method string, operation *openapi3.Operation) error {
		for _, param := range operation.Parameters {
			for _, content := range param.Value.Content {
				s := content.Schema
				if s.Ref != "" {
					if referenced[s.Ref] {
						continue
					}
					referenced[s.Ref] = true
				}
				if err := walkSchemasImpl(s, f); err != nil {
					return err
				}
			}
			s := param.Value.Schema
			if s.Ref != "" {
				if referenced[s.Ref] {
					continue
				}
				referenced[s.Ref] = true
			}
			if err := walkSchemasImpl(s, f); err != nil {
				return err
			}
		}
		if operation.RequestBody == nil {
			empty := openapi3.NewSchemaRef("", openapi3.NewObjectSchema())
			if err := walkSchemasImpl(empty, f); err != nil {
				return err
			}
		} else {
			for _, content := range operation.RequestBody.Value.Content {
				s := content.Schema
				if s.Ref != "" {
					if referenced[s.Ref] {
						continue
					}
					referenced[s.Ref] = true
				}
				if err := walkSchemasImpl(s, f); err != nil {
					return err
				}
			}
		}
		for _, res := range operation.Responses {
			for _, content := range res.Value.Content {
				s := content.Schema
				if s.Ref != "" {
					if referenced[s.Ref] {
						continue
					}
					referenced[s.Ref] = true
				}
				if err := walkSchemasImpl(s, f); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func walkSchemasImpl(schema *openapi3.SchemaRef, f func(schema *openapi3.SchemaRef) error) error {
	if err := f(schema); err != nil {
		return err
	}
	switch schema.Value.Type {
	case "array":
		if err := walkSchemasImpl(schema.Value.Items, f); err != nil {
			return err
		}
	case "object":
		for _, propertySchema := range schema.Value.Properties {
			if err := walkSchemasImpl(propertySchema, f); err != nil {
				return err
			}
		}
	}

	return nil
}
