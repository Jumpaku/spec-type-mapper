package visitor

import "github.com/getkin/kin-openapi/openapi3"

type OperationKey struct {
	Path   string
	Method string
}

type OperationVisitor interface {
	Visit(key OperationKey, operation *openapi3.Operation) error
}

type SchemaVisitor interface {
	PreVisit(schema *openapi3.SchemaRef) error
	PostVisit(schema *openapi3.SchemaRef) error
}

type OperationVisitorFunc func(key OperationKey, operation *openapi3.Operation) error

func (visit OperationVisitorFunc) Visit(key OperationKey, operation *openapi3.Operation) error {
	return visit(key, operation)
}

func IterateOperations(doc *openapi3.T, visitor OperationVisitor) error {
	for path, item := range doc.Paths {
		for method, operation := range item.Operations() {
			if err := visitor.Visit(OperationKey{Path: path, Method: method}, operation); err != nil {
				return err
			}
		}
	}
	return nil
}

func WalkSchemas(schema *openapi3.SchemaRef, visitor SchemaVisitor) error {
	if schema == nil {
		return nil
	}

	if err := visitor.PreVisit(schema); err != nil {
		return err
	}
	v := schema.Value

	for _, v := range v.OneOf {
		if err := WalkSchemas(v, visitor); err != nil {
			return err
		}
	}
	for _, v := range v.AnyOf {
		if err := WalkSchemas(v, visitor); err != nil {
			return err
		}
	}
	for _, v := range v.AllOf {
		if err := WalkSchemas(v, visitor); err != nil {
			return err
		}
	}
	if err := WalkSchemas(v.Not, visitor); err != nil {
		return err
	}
	if err := WalkSchemas(v.Items, visitor); err != nil {
		return err
	}
	for _, v := range v.Properties {
		if err := WalkSchemas(v, visitor); err != nil {
			return err
		}
	}

	if v := v.AdditionalProperties.Schema; v != nil {
		if err := WalkSchemas(v, visitor); err != nil {
			return err
		}
	}

	if err := visitor.PostVisit(schema); err != nil {
		return err
	}
	return nil
}
