package visitor_test

import (
	"spec-type-mapper/internal/data"
	"spec-type-mapper/visitor"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slices"
)

type testOperationVisitor struct {
	Operations map[visitor.OperationKey]*openapi3.Operation
}

func (v *testOperationVisitor) Visit(key visitor.OperationKey, operation *openapi3.Operation) error {
	v.Operations[key] = operation
	return nil
}

type testSchemaVisitor struct {
	PreGot  []*openapi3.SchemaRef
	PostGot []*openapi3.SchemaRef
}

func (v *testSchemaVisitor) PreVisit(schema *openapi3.SchemaRef) error {
	v.PreGot = append(v.PreGot, schema)
	return nil
}
func (v *testSchemaVisitor) PostVisit(schema *openapi3.SchemaRef) error {
	v.PostGot = append(v.PostGot, schema)
	return nil
}

func TestIterateOperations(t *testing.T) {
	loader := openapi3.NewLoader()
	doc, _ := loader.LoadFromData(data.ExampleJSON)
	v := &testOperationVisitor{Operations: map[visitor.OperationKey]*openapi3.Operation{}}

	wantOperationIDs := map[visitor.OperationKey]string{
		{
			Path:   "/pets",
			Method: "GET",
		}: "listPets",
		{
			Path:   "/pets",
			Method: "POST",
		}: "createPets",
		{
			Path:   "/pets/{petId}",
			Method: "GET",
		}: "showPetById",
	}

	err := visitor.IterateOperations(doc, v)
	assert.Nil(t, err)
	assert.Equal(t, len(wantOperationIDs), len(v.Operations))
	for key, got := range v.Operations {
		assert.Contains(t, wantOperationIDs, key)
		assert.Equal(t, wantOperationIDs[key], got.OperationID)
	}
}

func TestWalkSchemas(t *testing.T) {
	loader := openapi3.NewLoader()
	doc, _ := loader.LoadFromData(data.ExampleJSON)
	testcases := doc.Components.Schemas

	wants := map[string][]*openapi3.Schema{
		"Pet": {
			&openapi3.Schema{
				Type: "object",
				Properties: openapi3.Schemas{
					"id":   nil,
					"name": nil,
					"tag":  nil,
				},
			},
			&openapi3.Schema{
				Type:   "integer",
				Format: "int64",
			},
			&openapi3.Schema{
				Type: "string",
			},
			&openapi3.Schema{
				Type: "string",
			},
		},
		"Pets": {
			&openapi3.Schema{
				Type: "array",
			},
			&openapi3.Schema{
				Type: "object",
				Properties: openapi3.Schemas{
					"id":   nil,
					"name": nil,
					"tag":  nil,
				},
			},
			&openapi3.Schema{
				Type:   "integer",
				Format: "int64",
			},
			&openapi3.Schema{
				Type: "string",
			},
			&openapi3.Schema{
				Type: "string",
			},
		},
		"Error": {
			&openapi3.Schema{
				Type: "object",
				Properties: openapi3.Schemas{
					"code":    nil,
					"message": nil,
				},
			},
			&openapi3.Schema{
				Type:   "integer",
				Format: "int32",
			},
			&openapi3.Schema{
				Type: "string",
			},
		},
	}

	for name, in := range testcases {
		t.Run(name, func(t *testing.T) {
			v := &testSchemaVisitor{}
			err := visitor.WalkSchemas(in, v)
			assert.Nil(t, err)
			wants := wants[name]
			assert.Equal(t, len(wants), len(v.PreGot))
			assert.Equal(t, len(wants), len(v.PostGot))
			for _, want := range wants {
				matchSchema := func(got *openapi3.SchemaRef) bool {
					for prop := range want.Properties {
						if _, ok := got.Value.Properties[prop]; !ok {
							return false
						}
					}
					return want.Type == got.Value.Type && want.Format == got.Value.Format
				}

				{
					found := slices.IndexFunc(v.PreGot, matchSchema)
					assert.GreaterOrEqualf(t, found, 0, "want = %s\ngot  = %s", spew.Sdump(want), len(v.PreGot))
				}
				{
					found := slices.IndexFunc(v.PostGot, matchSchema)
					assert.GreaterOrEqualf(t, found, 0, "want = %s\ngot  = %s", spew.Sdump(want), len(v.PostGot))
				}
			}
		})
	}
}
