package csharp

import (
	"spec-type-mapper/visitor"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"golang.org/x/exp/slices"
)

type Kind int

const (
	KindUnknown Kind = iota
	KindInt
	KindLong
	KindFloat
	KindDouble
	KindString
	KindBool
	KindClass
	KindList
	KindEnum
)

func (k Kind) String() string {
	switch k {
	default:
		return ""
	case KindInt:
		return "KindInt"
	case KindLong:
		return "KindLong"
	case KindFloat:
		return "KindFloat"
	case KindDouble:
		return "KindDouble"
	case KindString:
		return "KindString"
	case KindBool:
		return "KindBool"
	case KindClass:
		return "KindClass"
	case KindList:
		return "KindList"
	case KindEnum:
		return "KindEnum"
	}
}

type Type struct {
	Description     Description
	Kind            Kind
	ClassMembers    map[string]*openapi3.SchemaRef
	ListElementType *openapi3.SchemaRef
	EnumValues      []string
	Nullable        bool
}

type Description map[DescriptionKind]any

type DescriptionKind int

const (
	DescriptionKindDescription DescriptionKind = iota
	DescriptionKindTitle
	DescriptionKindExample
	DescriptionKindStringFormat
	DescriptionKindStringMaxLength
	DescriptionKindStringMinLength
	DescriptionKindStringPattern
	DescriptionKindNumberMax
	DescriptionKindNumberExclusiveMax
	DescriptionKindNumberMin
	DescriptionKindNumberExclusiveMin
	DescriptionKindNumberMultipleOf
	DescriptionKindArrayMaxItems
	DescriptionKindArrayMinItems
)

func ExtractDescription(k Kind, s *openapi3.SchemaRef) Description {
	desc := Description{}
	v := s.Value
	if v.Title != "" {
		desc[DescriptionKindTitle] = v.Title
	}
	if v.Description != "" {
		desc[DescriptionKindDescription] = v.Description
	}

	switch k {
	case KindInt, KindLong, KindFloat, KindDouble:
		if v.Max != nil {
			if v.ExclusiveMax {
				desc[DescriptionKindNumberExclusiveMax] = v.ExclusiveMax
			}
			desc[DescriptionKindNumberMax] = *v.Max
		}
		if v.Min != nil {
			if v.ExclusiveMin {
				desc[DescriptionKindNumberExclusiveMin] = v.ExclusiveMin
			}
			desc[DescriptionKindNumberMin] = *v.Min
		}
	case KindString:
		if v.Format != "" {
			desc[DescriptionKindStringFormat] = v.Format
		}
		if v.MaxLength != nil {
			desc[DescriptionKindStringMaxLength] = v.MaxLength
		}
		if v.MinLength > 0 {
			desc[DescriptionKindStringMinLength] = v.MinLength
		}
		if v.Pattern != "" {
			desc[DescriptionKindStringPattern] = v.Pattern
		}
	case KindList:
		if v.MaxItems != nil {
			desc[DescriptionKindArrayMaxItems] = v.MaxItems
		}
		if v.MinLength > 0 {
			desc[DescriptionKindArrayMinItems] = v.MinItems
		}
	}

	return desc
}

type resolver struct {
	typeMap map[*openapi3.SchemaRef]Type
}

func (r *resolver) PreVisit(schema *openapi3.SchemaRef) error {
	return nil
}
func (r *resolver) PostVisit(schema *openapi3.SchemaRef) error {
	v := schema.Value
	var t Type

	t.Nullable = v.Nullable

	switch v.Type {
	case "integer":
		switch v.Format {
		case "int32":
			t.Kind = KindInt
		default:
			t.Kind = KindLong
		}
	case "number":
		switch v.Format {
		case "float":
			t.Kind = KindFloat
		default:
			t.Kind = KindDouble
		}
	case "string":
		if v.Enum != nil {
			t.Kind = KindEnum
			for _, e := range v.Enum {
				t.EnumValues = append(t.EnumValues, e.(string))
			}
		} else {
			t.Kind = KindString
		}
	case "boolean":
		t.Kind = KindBool
	case "array":
		t.Kind = KindList
		t.ListElementType = schema.Value.Items
	case "object":
		t.Kind = KindClass
		t.ClassMembers = map[string]*openapi3.SchemaRef{}
		for name, schema := range v.Properties {
			t.ClassMembers[name] = schema
		}
	}

	for _, child := range v.AllOf {
		t.Kind = KindClass
		for name, schema := range r.typeMap[child].ClassMembers {
			t.ClassMembers[name] = schema
		}
	}
	for _, child := range v.AnyOf {
		t.Kind = KindClass
		for name, schema := range r.typeMap[child].ClassMembers {
			t.ClassMembers[name] = schema
		}
	}
	for _, child := range v.OneOf {
		t.Kind = KindClass
		for name, schema := range r.typeMap[child].ClassMembers {
			t.ClassMembers[name] = schema
		}
	}

	//t.Description = ExtractDescription(t.Kind, schema)

	r.typeMap[schema] = t

	return nil
}

func ResolveType(rootSchema *openapi3.SchemaRef) map[*openapi3.SchemaRef]Type {
	v := &resolver{typeMap: map[*openapi3.SchemaRef]Type{}}
	_ = visitor.WalkSchemas(rootSchema, v)
	return v.typeMap
}

type ParamIn string

const (
	ParamInUnspecified ParamIn = ""
	ParamInQuery       ParamIn = "query"
	ParamInHeader      ParamIn = "header"
	ParamInPath        ParamIn = "path"
	ParamInCookie      ParamIn = "cookie"
)

type Parameter struct {
	Name   string
	In     ParamIn
	Media  string
	Schema *openapi3.SchemaRef
}
type RequestBody struct {
	Media  string
	Schema *openapi3.SchemaRef
}
type ResponseHeader struct {
	Name   string
	Media  string
	Schema *openapi3.SchemaRef
}
type ResponseBody struct {
	Media  string
	Schema *openapi3.SchemaRef
}
type Response struct {
	Code    string
	Headers []ResponseHeader
	Bodies  []ResponseBody
}

func ResolveOperation(operation *openapi3.Operation) ([]Parameter, []RequestBody, []Response) {
	params := []Parameter{}
	for _, operationParameter := range operation.Parameters {
		param := Parameter{
			Name:   operationParameter.Value.Name,
			In:     ParamIn(operationParameter.Value.In),
			Schema: &openapi3.SchemaRef{},
		}
		// A parameter MUST contain either a schema property, or a content property, but not both.
		for parameterMedia, parameterContent := range operationParameter.Value.Content {
			// Content MUST be a map that contain only one entry.
			param.Media = parameterMedia
			param.Schema = parameterContent.Schema
		}
		param.Schema = operationParameter.Value.Schema
		params = append(params, param)
	}
	slices.SortFunc(params, func(p0, p1 Parameter) int {
		if p0.In != p1.In {
			return strings.Compare(string(p0.In), string(p1.In))
		}
		return strings.Compare(p0.Name, p1.Name)
	})

	reqBody := []RequestBody{}
	if operation.RequestBody != nil {
		for media, content := range operation.RequestBody.Value.Content {
			reqBody = append(reqBody, RequestBody{Media: media, Schema: content.Schema})
		}
	}
	slices.SortFunc(reqBody, func(r0, r1 RequestBody) int {
		return strings.Compare(string(r0.Media), string(r1.Media))
	})

	res := []Response{}
	for code, operationResponse := range operation.Responses {
		headers := []ResponseHeader{}
		for name, header := range operationResponse.Value.Headers {
			resHeader := ResponseHeader{Name: name, Schema: &openapi3.SchemaRef{}}
			for headerMedia, headerContent := range header.Value.Content {
				resHeader.Media = headerMedia
				resHeader.Schema = headerContent.Schema
			}
			resHeader.Schema = header.Value.Schema
			headers = append(headers, resHeader)
		}
		slices.SortFunc(headers, func(r0, r1 ResponseHeader) int {
			return strings.Compare(r0.Name, r1.Name)
		})

		bodies := []ResponseBody{}
		for media, content := range operationResponse.Value.Content {
			bodies = append(bodies, ResponseBody{Media: media, Schema: content.Schema})
		}
		slices.SortFunc(bodies, func(b0, b1 ResponseBody) int {
			return strings.Compare(b0.Media, b1.Media)
		})

		res = append(res, Response{Code: code, Headers: headers, Bodies: bodies})
	}
	slices.SortFunc(res, func(r0, r1 Response) int {
		return strings.Compare(string(r0.Code), string(r1.Code))
	})

	return params, reqBody, res
}
