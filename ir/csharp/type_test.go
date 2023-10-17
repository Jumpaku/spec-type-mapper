package csharp_test

import (
	"spec-type-mapper/internal/data"
	"spec-type-mapper/ir/csharp"
	"spec-type-mapper/visitor"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/maps"
)

func schemaOf(value *openapi3.Schema) *openapi3.SchemaRef {
	return &openapi3.SchemaRef{Value: value}
}
func schemaBoolean() *openapi3.SchemaRef {
	return schemaOf(&openapi3.Schema{Type: "boolean"})
}
func schemaNumber(format string) *openapi3.SchemaRef {
	return schemaOf(&openapi3.Schema{Type: "number", Format: format})
}
func schemaInteger(format string) *openapi3.SchemaRef {
	return schemaOf(&openapi3.Schema{Type: "integer", Format: format})
}
func schemaString(format string) *openapi3.SchemaRef {
	return schemaOf(&openapi3.Schema{Type: "string", Format: format})
}
func schemaObject(props map[string]*openapi3.SchemaRef) *openapi3.SchemaRef {
	return schemaOf(&openapi3.Schema{Type: "object", Properties: props})
}
func schemaArray(items *openapi3.SchemaRef) *openapi3.SchemaRef {
	return schemaOf(&openapi3.Schema{Type: "array", Items: items})
}
func schemaEnum(values []string) *openapi3.SchemaRef {
	enumVals := []any{}
	for _, v := range values {
		enumVals = append(enumVals, v)
	}
	return schemaOf(&openapi3.Schema{Type: "string", Enum: enumVals})
}

func matchType(t *testing.T, want, got csharp.Type) {
	t.Helper()

	assert.Equal(t, want.Kind, got.Kind)
	assert.Equal(t, want.Nullable, got.Nullable)
	assert.ElementsMatch(t, want.EnumValues, got.EnumValues)
	if want.ClassMembers != nil {
		for member := range want.ClassMembers {
			if _, ok := got.ClassMembers[member]; !ok {
				t.Errorf("want member %s not found in got members", member)
			}
		}
	}
	if want.ListElementType != nil {
		if got.ListElementType == nil {
			t.Error("got list element type is nil")
		}
	}
}

func TestResolveType_Simple(t *testing.T) {
	testcases := []struct {
		name string
		root *openapi3.SchemaRef
		want csharp.Type
	}{
		// string
		{
			name: "string",
			root: schemaString(""),
			want: csharp.Type{Kind: csharp.KindString},
		}, {
			name: "string(byte)",
			root: schemaString("byte"),
			want: csharp.Type{Kind: csharp.KindString},
		}, {
			name: "string(binary)",
			root: schemaString("binary"),
			want: csharp.Type{Kind: csharp.KindString},
		}, {
			name: "string(date)",
			root: schemaString("date"),
			want: csharp.Type{Kind: csharp.KindString},
		}, {
			name: "string(date-time)",
			root: schemaString("date-time"),
			want: csharp.Type{Kind: csharp.KindString},
		}, {
			name: "string(password)",
			root: schemaString("password"),
			want: csharp.Type{Kind: csharp.KindString},
		},
		// float
		{
			name: "number(float)",
			root: schemaNumber("float"),
			want: csharp.Type{Kind: csharp.KindFloat},
		},
		// double
		{
			name: "string(double)",
			root: schemaNumber("double"),
			want: csharp.Type{Kind: csharp.KindDouble},
		},
		// float
		{
			name: "integer(int32)",
			root: schemaInteger("int32"),
			want: csharp.Type{Kind: csharp.KindInt},
		},
		// double
		{
			name: "integer(int64)",
			root: schemaInteger("int64"),
			want: csharp.Type{Kind: csharp.KindLong},
		},
		// bool
		{
			name: "boolean",
			root: schemaBoolean(),
			want: csharp.Type{Kind: csharp.KindBool},
		},
		// enum
		{
			name: "enum",
			root: schemaEnum([]string{"aaa", "bbb", "ccc"}),
			want: csharp.Type{Kind: csharp.KindEnum, EnumValues: []string{"aaa", "bbb", "ccc"}},
		},
		// array
		{
			name: "array",
			root: schemaArray(schemaBoolean()),
			want: csharp.Type{Kind: csharp.KindList},
		},
		// object
		{
			name: "object",
			root: schemaObject(map[string]*openapi3.SchemaRef{}),
			want: csharp.Type{Kind: csharp.KindClass},
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			got := csharp.ResolveType(testcase.root)
			matchType(t, testcase.want, got[testcase.root])
		})
	}
}
func TestResolveType_Composite(t *testing.T) {
	testcases := []struct {
		name    string
		root    *openapi3.SchemaRef
		want    csharp.Type
		wantLen int
	}{
		{
			name:    "array of strings",
			root:    schemaArray(schemaString("")),
			want:    csharp.Type{Kind: csharp.KindList, ListElementType: schemaString("")},
			wantLen: 2,
		}, {
			name:    "array of longs",
			root:    schemaArray(schemaInteger("int64")),
			want:    csharp.Type{Kind: csharp.KindList, ListElementType: schemaInteger("int64")},
			wantLen: 2,
		}, {
			name:    "array of arrays of longs",
			root:    schemaArray(schemaArray(schemaInteger("int64"))),
			want:    csharp.Type{Kind: csharp.KindList, ListElementType: schemaArray(schemaInteger("int64"))},
			wantLen: 3,
		},
		{
			name: "object",
			root: schemaObject(map[string]*openapi3.SchemaRef{
				"propA": schemaString(""),
				"propB": schemaInteger("int32"),
				"propC": schemaInteger("int64"),
				"propD": schemaNumber("float"),
				"propE": schemaNumber("double"),
				"propF": schemaBoolean(),
			}),
			want: csharp.Type{Kind: csharp.KindClass, ClassMembers: map[string]*openapi3.SchemaRef{
				"propA": schemaString(""),
				"propB": schemaInteger("int32"),
				"propC": schemaInteger("int64"),
				"propD": schemaNumber("float"),
				"propE": schemaNumber("double"),
				"propF": schemaBoolean(),
			}},
			wantLen: 7,
		},
		{
			name: "array of objects",
			root: schemaArray(schemaObject(map[string]*openapi3.SchemaRef{
				"propA": schemaString(""),
				"propB": schemaInteger("int32"),
				"propC": schemaInteger("int64"),
				"propD": schemaNumber("float"),
				"propE": schemaNumber("double"),
				"propF": schemaBoolean(),
			})),
			want: csharp.Type{Kind: csharp.KindList, ListElementType: schemaObject(map[string]*openapi3.SchemaRef{
				"propA": schemaString(""),
				"propB": schemaInteger("int32"),
				"propC": schemaInteger("int64"),
				"propD": schemaNumber("float"),
				"propE": schemaNumber("double"),
				"propF": schemaBoolean(),
			})},
			wantLen: 8,
		},
		{
			name: "object having array member",
			root: schemaObject(map[string]*openapi3.SchemaRef{
				"propA": schemaString(""),
				"propB": schemaInteger("int32"),
				"propC": schemaInteger("int64"),
				"propD": schemaNumber("float"),
				"propE": schemaNumber("double"),
				"propF": schemaBoolean(),
				"propG": schemaArray(schemaObject(map[string]*openapi3.SchemaRef{})),
			}),
			want: csharp.Type{Kind: csharp.KindClass, ClassMembers: map[string]*openapi3.SchemaRef{
				"propA": schemaString(""),
				"propB": schemaInteger("int32"),
				"propC": schemaInteger("int64"),
				"propD": schemaNumber("float"),
				"propE": schemaNumber("double"),
				"propF": schemaBoolean(),
				"propG": schemaArray(schemaObject(map[string]*openapi3.SchemaRef{})),
			}},
			wantLen: 9,
		},
		{
			name: "object having array and object members",
			root: schemaObject(map[string]*openapi3.SchemaRef{
				"propA": schemaString(""),
				"propB": schemaInteger("int32"),
				"propC": schemaInteger("int64"),
				"propD": schemaNumber("float"),
				"propE": schemaNumber("double"),
				"propF": schemaBoolean(),
				"propG": schemaArray(schemaObject(map[string]*openapi3.SchemaRef{})),
				"propH": schemaObject(map[string]*openapi3.SchemaRef{
					"propA": schemaString(""),
					"propB": schemaInteger("int32"),
					"propC": schemaInteger("int64"),
					"propD": schemaNumber("float"),
					"propE": schemaNumber("double"),
					"propF": schemaBoolean(),
				}),
			}),
			want: csharp.Type{Kind: csharp.KindClass, ClassMembers: map[string]*openapi3.SchemaRef{
				"propA": schemaString(""),
				"propB": schemaInteger("int32"),
				"propC": schemaInteger("int64"),
				"propD": schemaNumber("float"),
				"propE": schemaNumber("double"),
				"propF": schemaBoolean(),
				"propG": schemaArray(schemaObject(map[string]*openapi3.SchemaRef{})),
				"propH": schemaObject(map[string]*openapi3.SchemaRef{
					"propA": schemaString(""),
					"propB": schemaInteger("int32"),
					"propC": schemaInteger("int64"),
					"propD": schemaNumber("float"),
					"propE": schemaNumber("double"),
					"propF": schemaBoolean(),
				}),
			}},
			wantLen: 16,
		},
		{
			name: "object with oneOf",
			root: schemaOf(&openapi3.Schema{
				Type: "object",
				OneOf: []*openapi3.SchemaRef{
					schemaObject(map[string]*openapi3.SchemaRef{
						"propA": schemaString(""),
						"propB": schemaInteger("int32"),
						"propC": schemaInteger("int64"),
					}),
					schemaObject(map[string]*openapi3.SchemaRef{
						"propD": schemaNumber("float"),
						"propE": schemaNumber("double"),
						"propF": schemaBoolean(),
					}),
					schemaObject(map[string]*openapi3.SchemaRef{
						"propG": schemaArray(schemaObject(map[string]*openapi3.SchemaRef{})),
						"propH": schemaObject(map[string]*openapi3.SchemaRef{
							"propA": schemaString(""),
							"propB": schemaInteger("int32"),
							"propC": schemaInteger("int64"),
							"propD": schemaNumber("float"),
							"propE": schemaNumber("double"),
							"propF": schemaBoolean(),
						}),
					}),
				},
			}),
			want: csharp.Type{Kind: csharp.KindClass, ClassMembers: map[string]*openapi3.SchemaRef{
				"propA": schemaString(""),
				"propB": schemaInteger("int32"),
				"propC": schemaInteger("int64"),
				"propD": schemaNumber("float"),
				"propE": schemaNumber("double"),
				"propF": schemaBoolean(),
				"propG": schemaArray(schemaObject(map[string]*openapi3.SchemaRef{})),
				"propH": schemaObject(map[string]*openapi3.SchemaRef{
					"propA": schemaString(""),
					"propB": schemaInteger("int32"),
					"propC": schemaInteger("int64"),
					"propD": schemaNumber("float"),
					"propE": schemaNumber("double"),
					"propF": schemaBoolean(),
				}),
			}},
			wantLen: 19,
		},
		{
			name: "object with anyOf",
			root: schemaOf(&openapi3.Schema{
				Type: "object",
				AnyOf: []*openapi3.SchemaRef{
					schemaObject(map[string]*openapi3.SchemaRef{
						"propA": schemaString(""),
						"propB": schemaInteger("int32"),
						"propC": schemaInteger("int64"),
					}),
					schemaObject(map[string]*openapi3.SchemaRef{
						"propD": schemaNumber("float"),
						"propE": schemaNumber("double"),
						"propF": schemaBoolean(),
					}),
					schemaObject(map[string]*openapi3.SchemaRef{
						"propG": schemaArray(schemaObject(map[string]*openapi3.SchemaRef{})),
						"propH": schemaObject(map[string]*openapi3.SchemaRef{
							"propA": schemaString(""),
							"propB": schemaInteger("int32"),
							"propC": schemaInteger("int64"),
							"propD": schemaNumber("float"),
							"propE": schemaNumber("double"),
							"propF": schemaBoolean(),
						}),
					}),
				},
			}),
			want: csharp.Type{Kind: csharp.KindClass, ClassMembers: map[string]*openapi3.SchemaRef{
				"propA": schemaString(""),
				"propB": schemaInteger("int32"),
				"propC": schemaInteger("int64"),
				"propD": schemaNumber("float"),
				"propE": schemaNumber("double"),
				"propF": schemaBoolean(),
				"propG": schemaArray(schemaObject(map[string]*openapi3.SchemaRef{})),
				"propH": schemaObject(map[string]*openapi3.SchemaRef{
					"propA": schemaString(""),
					"propB": schemaInteger("int32"),
					"propC": schemaInteger("int64"),
					"propD": schemaNumber("float"),
					"propE": schemaNumber("double"),
					"propF": schemaBoolean(),
				}),
			}},
			wantLen: 19,
		},
		{
			name: "object with allOf",
			root: schemaOf(&openapi3.Schema{
				Type: "object",
				AllOf: []*openapi3.SchemaRef{
					schemaObject(map[string]*openapi3.SchemaRef{
						"propA": schemaString(""),
						"propB": schemaInteger("int32"),
						"propC": schemaInteger("int64"),
					}),
					schemaObject(map[string]*openapi3.SchemaRef{
						"propD": schemaNumber("float"),
						"propE": schemaNumber("double"),
						"propF": schemaBoolean(),
					}),
					schemaObject(map[string]*openapi3.SchemaRef{
						"propG": schemaArray(schemaObject(map[string]*openapi3.SchemaRef{})),
						"propH": schemaObject(map[string]*openapi3.SchemaRef{
							"propA": schemaString(""),
							"propB": schemaInteger("int32"),
							"propC": schemaInteger("int64"),
							"propD": schemaNumber("float"),
							"propE": schemaNumber("double"),
							"propF": schemaBoolean(),
						}),
					}),
				},
			}),
			want: csharp.Type{Kind: csharp.KindClass, ClassMembers: map[string]*openapi3.SchemaRef{
				"propA": schemaString(""),
				"propB": schemaInteger("int32"),
				"propC": schemaInteger("int64"),
				"propD": schemaNumber("float"),
				"propE": schemaNumber("double"),
				"propF": schemaBoolean(),
				"propG": schemaArray(schemaObject(map[string]*openapi3.SchemaRef{})),
				"propH": schemaObject(map[string]*openapi3.SchemaRef{
					"propA": schemaString(""),
					"propB": schemaInteger("int32"),
					"propC": schemaInteger("int64"),
					"propD": schemaNumber("float"),
					"propE": schemaNumber("double"),
					"propF": schemaBoolean(),
				}),
			}},
			wantLen: 19,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			got := csharp.ResolveType(testcase.root)
			assert.Equal(t, testcase.wantLen, len(got))
			matchType(t, testcase.want, got[testcase.root])
			validateResolvedType(t, testcase.root, got)
		})
	}
}

func validateResolvedType(t *testing.T, root *openapi3.SchemaRef, result map[*openapi3.SchemaRef]csharp.Type) {
	t.Helper()

	if _, ok := result[root]; !ok {
		t.Error("root must be included in result")
	}
	for addr, typ := range result {
		switch addr.Value.Type {
		case "string":
			if addr.Value.Enum == nil {
				if typ.Kind != csharp.KindString {
					t.Error("type must be string")
				}
			} else {
				if typ.Kind != csharp.KindEnum {
					t.Error("type must be enum")
				}
				assert.ElementsMatch(t, addr.Value.Enum, typ.EnumValues)
			}
		case "boolean":
			if typ.Kind != csharp.KindBool {
				t.Error("type must be bool")
			}
		case "number":
			if addr.Value.Format == "float" {
				if typ.Kind != csharp.KindFloat {
					t.Error("type must be float")
				}
			} else {
				if typ.Kind != csharp.KindDouble {
					t.Error("type must be double")
				}
			}
		case "integer":
			if addr.Value.Format == "int32" {
				if typ.Kind != csharp.KindInt {
					t.Error("type must be int")
				}
			} else {
				if typ.Kind != csharp.KindLong {
					t.Error("type must be long")
				}
			}
		case "array":
			if typ.Kind != csharp.KindList {
				t.Error("type must be list")
			}
			if _, ok := result[addr.Value.Items]; !ok {
				t.Error("result must include items schema")
			}
		case "object":
			if typ.Kind != csharp.KindClass {
				t.Error("type must be list")
			}
			for prop := range addr.Value.Properties {
				_, ok := typ.ClassMembers[prop]
				if !ok {
					t.Errorf("class member must include property schema: %s", prop)
				}
			}
			for _, memberSchema := range typ.ClassMembers {
				if _, ok := result[memberSchema]; !ok {
					t.Error("result must include member schema")
				}
			}

		default:
		}
	}
}

type testOperationVisitor struct {
	Operations map[visitor.OperationKey]*openapi3.Operation
}

func (v *testOperationVisitor) Visit(key visitor.OperationKey, operation *openapi3.Operation) error {
	v.Operations[key] = operation
	return nil
}

func TestResolveOperation(t *testing.T) {
	loader := openapi3.NewLoader()
	doc, _ := loader.LoadFromData(data.ExampleJSON)
	v := &testOperationVisitor{Operations: make(map[visitor.OperationKey]*openapi3.Operation)}
	_ = visitor.IterateOperations(doc, v)

	wants := map[string]struct {
		params  []csharp.Parameter
		reqBody []csharp.RequestBody
		res     []csharp.Response
	}{
		"listPets": {
			params: []csharp.Parameter{
				{
					Name: "limit",
					In:   csharp.ParamInQuery,
					Schema: &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type:   "integer",
							Format: "int32",
						},
					},
				},
			},
			res: []csharp.Response{
				{
					Code: "200",
					Headers: []csharp.ResponseHeader{
						{
							Name: "x-next",
							Schema: &openapi3.SchemaRef{
								Value: &openapi3.Schema{
									Type: "string",
								},
							},
						},
					},
					Bodies: []csharp.ResponseBody{
						{
							Media: "application/json",
							Schema: &openapi3.SchemaRef{
								Value: &openapi3.Schema{
									Type: "array",
								},
							},
						},
					},
				},
				{
					Code: "default",
					Bodies: []csharp.ResponseBody{
						{
							Media: "application/json",
							Schema: &openapi3.SchemaRef{
								Value: &openapi3.Schema{
									Type: "object",
									Properties: map[string]*openapi3.SchemaRef{
										"code":    nil,
										"message": nil,
									},
								},
							},
						},
					},
				},
			},
		},
		"createPets": {
			res: []csharp.Response{
				{
					Code: "201",
				},
				{
					Code: "default",
					Bodies: []csharp.ResponseBody{
						{
							Media: "application/json",
							Schema: &openapi3.SchemaRef{
								Value: &openapi3.Schema{
									Type: "object",
									Properties: map[string]*openapi3.SchemaRef{
										"code":    nil,
										"message": nil,
									},
								},
							},
						},
					},
				},
			},
		},
		"showPetById": {
			params: []csharp.Parameter{
				{
					Name: "petId",
					In:   "path",
					Schema: &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type: "string",
						},
					},
				},
			},
			res: []csharp.Response{
				{
					Code: "200",
					Bodies: []csharp.ResponseBody{
						{
							Media: "application/json",
							Schema: &openapi3.SchemaRef{
								Value: &openapi3.Schema{
									Type: "object",
									Properties: map[string]*openapi3.SchemaRef{
										"id":   nil,
										"name": nil,
										"tag":  nil,
									},
								},
							},
						},
					},
				},
				{
					Code: "default",
					Bodies: []csharp.ResponseBody{
						{
							Media: "application/json",
							Schema: &openapi3.SchemaRef{
								Value: &openapi3.Schema{
									Type: "object",
									Properties: map[string]*openapi3.SchemaRef{
										"code":    nil,
										"message": nil,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, op := range v.Operations {
		want := wants[op.OperationID]

		t.Run(op.OperationID, func(t *testing.T) {
			wantParams, wantReqBody, wantRes := want.params, want.reqBody, want.res
			gotParams, gotReqBody, gotRes := csharp.ResolveOperation(op)

			{
				t.Logf(`params`)
				assert.Equal(t, len(wantParams), len(gotParams))
				for i, wantParam := range wantParams {
					gotParam := gotParams[i]
					assert.Equal(t, wantParam.Name, gotParam.Name)
					assert.Equal(t, wantParam.In, gotParam.In)
					assert.Equal(t, wantParam.Media, gotParam.Media)
					equalSchema(t, wantParam.Schema, gotParam.Schema)
				}
			}

			{
				t.Logf(`reqBody`)
				assert.Equal(t, len(wantReqBody), len(gotReqBody))
				for i, wantReqBody := range wantReqBody {
					gotReqBody := gotReqBody[i]
					assert.Equal(t, wantReqBody.Media, gotReqBody.Media)
					equalSchema(t, wantReqBody.Schema, gotReqBody.Schema)
				}
			}

			{
				t.Logf(`res`)
				assert.Equal(t, len(wantRes), len(gotRes))
				for i, wantRes := range wantRes {
					gotRes := gotRes[i]
					assert.Equal(t, wantRes.Code, gotRes.Code)

					t.Logf(`resHeaders`)
					assert.Equal(t, len(wantRes.Headers), len(gotRes.Headers))
					for i, wantResHeader := range wantRes.Headers {
						gotResHeader := gotRes.Headers[i]
						assert.Equal(t, wantResHeader.Name, gotResHeader.Name)
						assert.Equal(t, wantResHeader.Media, gotResHeader.Media)
						equalSchema(t, wantResHeader.Schema, gotResHeader.Schema)
					}

					t.Logf(`resBodies`)
					assert.Equal(t, len(wantRes.Bodies), len(gotRes.Bodies))
					for i, wantResBody := range wantRes.Bodies {
						gotResBody := gotRes.Bodies[i]
						assert.Equal(t, wantResBody.Media, gotResBody.Media)
						equalSchema(t, wantResBody.Schema, gotResBody.Schema)
					}
				}
			}

		})
	}
}

func equalSchema(t *testing.T, want, got *openapi3.SchemaRef) {
	t.Helper()

	assert.Equal(t, want.Value.Type, got.Value.Type)
	assert.Equal(t, want.Value.Format, got.Value.Format)

	wantProp, gotProp := maps.Keys(want.Value.Properties), maps.Keys(got.Value.Properties)
	assert.ElementsMatch(t, wantProp, gotProp)
}
