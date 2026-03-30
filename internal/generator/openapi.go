package generator

import (
	"encoding/json"
	"path"
	"strings"

	"github.com/prongbang/codegen/internal/analyzer"
	"github.com/prongbang/codegen/internal/graph"
	"github.com/prongbang/codegen/internal/loader"
)

type Document struct {
	OpenAPI    string              `json:"openapi"`
	Info       Info                `json:"info"`
	Paths      map[string]PathItem `json:"paths"`
	Components Components          `json:"components,omitempty"`
}

type Info struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

type Components struct {
	Schemas map[string]*graph.Schema `json:"schemas,omitempty"`
}

type PathItem map[string]Operation

type Operation struct {
	OperationID string              `json:"operationId,omitempty"`
	Tags        []string            `json:"tags,omitempty"`
	RequestBody *RequestBody        `json:"requestBody,omitempty"`
	Responses   map[string]Response `json:"responses"`
}

type RequestBody struct {
	Required bool                 `json:"required"`
	Content  map[string]MediaType `json:"content"`
}

type Response struct {
	Description string               `json:"description"`
	Content     map[string]MediaType `json:"content,omitempty"`
}

type MediaType struct {
	Schema *graph.Schema `json:"schema,omitempty"`
}

func Build(mod *loader.Module, operations []analyzer.Operation) ([]byte, error) {
	builder := graph.NewBuilder(mod)
	paths := map[string]PathItem{}

	for _, op := range operations {
		if _, ok := paths[op.Path]; !ok {
			paths[op.Path] = PathItem{}
		}
		responseSchema := builder.Build(op.Response)
		responseContentType := "application/json"
		responseBodySchema := successSchema(responseSchema)
		if isStreamType(op.Response) {
			responseContentType = "application/octet-stream"
			responseBodySchema = &graph.Schema{Type: "string", Format: "binary"}
		}
		item := Operation{
			OperationID: op.OperationID,
			Tags:        op.Tags,
			Responses: map[string]Response{
				"200": {
					Description: "OK",
					Content: map[string]MediaType{
						responseContentType: {
							Schema: responseBodySchema,
						},
					},
				},
			},
		}
		if op.Request != nil {
			requestSchema := builder.Build(op.Request)
			requestContentType := "application/json"
			if graph.HasBinary(requestSchema, builder.Components) {
				requestContentType = "multipart/form-data"
			}
			item.RequestBody = &RequestBody{
				Required: true,
				Content: map[string]MediaType{
					requestContentType: {
						Schema: requestSchema,
					},
				},
			}
		}
		paths[op.Path][op.Method] = item
	}

	doc := Document{
		OpenAPI: "3.0.3",
		Info: Info{
			Title:   path.Base(mod.ModulePath),
			Version: "1.0.0",
		},
		Paths: paths,
		Components: Components{
			Schemas: builder.Components,
		},
	}

	return json.MarshalIndent(doc, "", "  ")
}

func isStreamType(ref *analyzer.TypeRef) bool {
	if ref == nil {
		return false
	}
	return ref.Kind == analyzer.TypeNamed && ref.Name == "Stream" && strings.HasSuffix(ref.Package, "/streamx")
}

func successSchema(data *graph.Schema) *graph.Schema {
	return &graph.Schema{
		Type: "object",
		Properties: map[string]*graph.Schema{
			"data":    data,
			"message": {Type: "string"},
		},
	}
}
