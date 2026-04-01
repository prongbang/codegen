package graph

import (
	"go/ast"
	"path"
	"reflect"
	"sort"
	"strings"

	"github.com/prongbang/codegen/internal/analyzer"
	"github.com/prongbang/codegen/internal/loader"
)

type Schema struct {
	Ref                  string             `json:"$ref,omitempty"`
	AllOf                []*Schema          `json:"allOf,omitempty"`
	Type                 string             `json:"type,omitempty"`
	Format               string             `json:"format,omitempty"`
	Nullable             bool               `json:"nullable,omitempty"`
	Properties           map[string]*Schema `json:"properties,omitempty"`
	Items                *Schema            `json:"items,omitempty"`
	AdditionalProperties *Schema            `json:"additionalProperties,omitempty"`
	Required             []string           `json:"required,omitempty"`
}

type FieldMeta struct {
	Name     string
	Required bool
	Ignore   bool
}

type Builder struct {
	Module     *loader.Module
	Components map[string]*Schema
	visiting   map[string]bool
}

func NewBuilder(mod *loader.Module) *Builder {
	return &Builder{
		Module:     mod,
		Components: map[string]*Schema{},
		visiting:   map[string]bool{},
	}
}

func (b *Builder) Build(ref *analyzer.TypeRef) *Schema {
	if ref == nil {
		return &Schema{Type: "object"}
	}

	switch ref.Kind {
	case analyzer.TypeScalar:
		return scalarSchema(ref.Name)
	case analyzer.TypeArray:
		return &Schema{Type: "array", Items: b.Build(ref.Elem)}
	case analyzer.TypeMap:
		return &Schema{Type: "object", AdditionalProperties: b.Build(ref.Elem)}
	case analyzer.TypeAny:
		return &Schema{Type: "object"}
	case analyzer.TypeObject:
		if structType, ok := ref.Expr.(*ast.StructType); ok {
			return b.buildInlineStruct(structType, nil, nil, nil)
		}
		return &Schema{Type: "object"}
	case analyzer.TypeNamed:
		if ref.Package == "mime/multipart" && ref.Name == "FileHeader" {
			return &Schema{Type: "string", Format: "binary"}
		}
		if ref.Package == "io" && ref.Name == "Reader" {
			return &Schema{Type: "string", Format: "binary"}
		}
		if ref.Package == "time" && ref.Name == "Time" {
			return &Schema{Type: "string", Format: "date-time"}
		}
		typeDef := b.Module.FindType(ref.Package, ref.Name)
		if typeDef == nil {
			if ref.Name == "Time" {
				return &Schema{Type: "string", Format: "date-time"}
			}
			return &Schema{Type: "object"}
		}
		componentName := componentName(ref)
		if _, ok := b.Components[componentName]; ok {
			return &Schema{Ref: "#/components/schemas/" + componentName}
		}
		if b.visiting[componentName] {
			return &Schema{Ref: "#/components/schemas/" + componentName}
		}
		b.visiting[componentName] = true
		b.Components[componentName] = b.buildNamed(typeDef, ref)
		delete(b.visiting, componentName)
		return &Schema{Ref: "#/components/schemas/" + componentName}
	}

	return &Schema{Type: "object"}
}

func (b *Builder) buildNamed(typeDef *loader.TypeSpecRef, ref *analyzer.TypeRef) *Schema {
	bindings := typeParamBindings(typeDef, ref)
	switch value := typeDef.Spec.Type.(type) {
	case *ast.StructType:
		return b.buildInlineStruct(value, typeDef.Pkg, typeDef.File, bindings)
	case *ast.ArrayType, *ast.MapType, *ast.InterfaceType:
		return b.Build(exprToTypeRefWithBindings(typeDef.Pkg, typeDef.File, value, bindings))
	case *ast.Ident, *ast.SelectorExpr, *ast.StarExpr, *ast.IndexExpr, *ast.IndexListExpr:
		return b.Build(exprToTypeRefWithBindings(typeDef.Pkg, typeDef.File, value, bindings))
	default:
		return &Schema{Type: "object"}
	}
}

func (b *Builder) flattenEmbedded(properties map[string]*Schema, required *[]string, ref *analyzer.TypeRef) {
	schema := b.Build(ref)
	if schema.Ref == "" {
		for key, value := range schema.Properties {
			properties[key] = value
		}
		*required = append(*required, schema.Required...)
		return
	}
	component := b.Components[componentName(ref)]
	if component == nil {
		return
	}
	for key, value := range component.Properties {
		properties[key] = value
	}
	*required = append(*required, component.Required...)
}

func (b *Builder) buildInlineStruct(structType *ast.StructType, pkg *loader.Package, file *ast.File, bindings map[string]*analyzer.TypeRef) *Schema {
	properties := map[string]*Schema{}
	required := []string{}
	for _, field := range structType.Fields.List {
		if len(field.Names) == 0 {
			if pkg != nil && file != nil {
				embedded := exprToTypeRefWithBindings(pkg, file, field.Type, bindings)
				b.flattenEmbedded(properties, &required, embedded)
			}
			continue
		}

		meta := fieldMeta(field)
		if meta.Ignore {
			continue
		}
		for _, name := range field.Names {
			propName := meta.Name
			if propName == "" {
				propName = name.Name
			}
			var schema *Schema
			if pkg != nil && file != nil {
				schema = b.Build(exprToTypeRefWithBindings(pkg, file, field.Type, bindings))
			} else {
				schema = inlineExprSchema(field.Type)
			}
			if isPointerType(field.Type) {
				schema = nullableSchema(schema)
			}
			properties[propName] = schema
			if meta.Required || !isPointerLike(field.Type) && propName != "" && !hasOptionalTag(field) {
				required = append(required, propName)
			}
		}
	}
	sort.Strings(required)
	return &Schema{
		Type:       "object",
		Properties: properties,
		Required:   unique(required),
	}
}

func componentName(ref *analyzer.TypeRef) string {
	base := path.Base(ref.Package)
	if base == "." || base == "" || base == "/" {
		return ref.Name + typeArgsSuffix(ref)
	}
	replacer := strings.NewReplacer("-", "_", ".", "_")
	return replacer.Replace(base) + "_" + ref.Name + typeArgsSuffix(ref)
}

func typeArgsSuffix(ref *analyzer.TypeRef) string {
	if ref == nil || len(ref.TypeArgs) == 0 {
		return ""
	}
	parts := make([]string, 0, len(ref.TypeArgs))
	for _, arg := range ref.TypeArgs {
		parts = append(parts, sanitizeComponentPart(typeRefComponentPart(arg)))
	}
	return "__" + strings.Join(parts, "__")
}

func typeRefComponentPart(ref *analyzer.TypeRef) string {
	if ref == nil {
		return "nil"
	}
	switch ref.Kind {
	case analyzer.TypeScalar:
		return ref.Name
	case analyzer.TypeNamed:
		base := ref.Name
		if ref.Package != "" {
			base = path.Base(ref.Package) + "_" + ref.Name
		}
		if len(ref.TypeArgs) == 0 {
			return base
		}
		parts := make([]string, 0, len(ref.TypeArgs)+1)
		parts = append(parts, base)
		for _, arg := range ref.TypeArgs {
			parts = append(parts, typeRefComponentPart(arg))
		}
		return strings.Join(parts, "_")
	case analyzer.TypeArray:
		return "array_" + typeRefComponentPart(ref.Elem)
	case analyzer.TypeMap:
		return "map_" + typeRefComponentPart(ref.Elem)
	case analyzer.TypeAny:
		return "any"
	case analyzer.TypeObject:
		return "object"
	default:
		return "invalid"
	}
}

func sanitizeComponentPart(value string) string {
	replacer := strings.NewReplacer("-", "_", ".", "_", "/", "_", "*", "ptr", "[", "_", "]", "_", ",", "_", " ", "_")
	return replacer.Replace(value)
}

func typeParamBindings(typeDef *loader.TypeSpecRef, ref *analyzer.TypeRef) map[string]*analyzer.TypeRef {
	if typeDef == nil || typeDef.Spec == nil || typeDef.Spec.TypeParams == nil || ref == nil || len(ref.TypeArgs) == 0 {
		return nil
	}
	params := typeDef.Spec.TypeParams.List
	bindings := map[string]*analyzer.TypeRef{}
	argIndex := 0
	for _, field := range params {
		for _, name := range field.Names {
			if argIndex >= len(ref.TypeArgs) {
				return bindings
			}
			bindings[name.Name] = cloneTypeRef(ref.TypeArgs[argIndex])
			argIndex++
		}
	}
	return bindings
}

func exprToTypeRefWithBindings(pkg *loader.Package, file *ast.File, expr ast.Expr, bindings map[string]*analyzer.TypeRef) *analyzer.TypeRef {
	ref := analyzer.ExprToTypeRef(pkg, file, expr)
	return applyTypeBindings(ref, pkg, bindings)
}

func applyTypeBindings(ref *analyzer.TypeRef, pkg *loader.Package, bindings map[string]*analyzer.TypeRef) *analyzer.TypeRef {
	if ref == nil {
		return nil
	}
	if len(bindings) == 0 {
		return ref
	}
	switch ref.Kind {
	case analyzer.TypeNamed:
		if pkg != nil && ref.Package == pkg.ImportPath {
			if bound, ok := bindings[ref.Name]; ok {
				return cloneTypeRef(bound)
			}
		}
		cloned := cloneTypeRef(ref)
		if len(cloned.TypeArgs) > 0 {
			for i, arg := range cloned.TypeArgs {
				cloned.TypeArgs[i] = applyTypeBindings(arg, pkg, bindings)
			}
		}
		return cloned
	case analyzer.TypeArray, analyzer.TypeMap:
		cloned := cloneTypeRef(ref)
		cloned.Elem = applyTypeBindings(ref.Elem, pkg, bindings)
		return cloned
	default:
		return ref
	}
}

func cloneTypeRef(ref *analyzer.TypeRef) *analyzer.TypeRef {
	if ref == nil {
		return nil
	}
	cloned := *ref
	if ref.Elem != nil {
		cloned.Elem = cloneTypeRef(ref.Elem)
	}
	if len(ref.TypeArgs) > 0 {
		cloned.TypeArgs = make([]*analyzer.TypeRef, len(ref.TypeArgs))
		for i, arg := range ref.TypeArgs {
			cloned.TypeArgs[i] = cloneTypeRef(arg)
		}
	}
	return &cloned
}

func scalarSchema(name string) *Schema {
	switch name {
	case "string":
		return &Schema{Type: "string"}
	case "bool":
		return &Schema{Type: "boolean"}
	case "float32":
		return &Schema{Type: "number", Format: "float"}
	case "float64":
		return &Schema{Type: "number", Format: "double"}
	case "byte":
		return &Schema{Type: "integer", Format: "int32"}
	case "int", "int8", "int16", "int32":
		return &Schema{Type: "integer", Format: "int32"}
	case "int64":
		return &Schema{Type: "integer", Format: "int64"}
	case "uint", "uint8", "uint16", "uint32":
		return &Schema{Type: "integer", Format: "int32"}
	case "uint64":
		return &Schema{Type: "integer", Format: "int64"}
	default:
		return &Schema{Type: "string"}
	}
}

func fieldMeta(field *ast.Field) FieldMeta {
	if field.Tag == nil {
		return FieldMeta{}
	}
	tagValue := strings.Trim(field.Tag.Value, "`")
	tag := reflect.StructTag(tagValue)
	if tag.Get("swaggerignore") == "true" {
		return FieldMeta{Ignore: true}
	}
	jsonTag := tag.Get("json")
	if jsonTag == "-" {
		return FieldMeta{Ignore: true}
	}
	if jsonTag != "" {
		parts := strings.Split(jsonTag, ",")
		return FieldMeta{
			Name:     parts[0],
			Required: hasRequiredValidate(tag.Get("validate")),
		}
	}
	formTag := tag.Get("form")
	if formTag != "" && formTag != "-" {
		return FieldMeta{
			Name:     strings.Split(formTag, ",")[0],
			Required: hasRequiredValidate(tag.Get("validate")),
		}
	}
	return FieldMeta{
		Required: hasRequiredValidate(tag.Get("validate")),
	}
}

func isPointerLike(expr ast.Expr) bool {
	switch expr.(type) {
	case *ast.StarExpr, *ast.ArrayType, *ast.MapType:
		return true
	default:
		return false
	}
}

func unique(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]bool{}
	result := []string{}
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func inlineExprSchema(expr ast.Expr) *Schema {
	switch value := expr.(type) {
	case *ast.StarExpr:
		return nullableSchema(inlineExprSchema(value.X))
	case *ast.StructType:
		properties := map[string]*Schema{}
		required := []string{}
		for _, field := range value.Fields.List {
			if len(field.Names) == 0 {
				continue
			}
			meta := fieldMeta(field)
			if meta.Ignore {
				continue
			}
			for _, name := range field.Names {
				propName := meta.Name
				if propName == "" {
					propName = name.Name
				}
				fieldSchema := inlineExprSchema(field.Type)
				if isPointerType(field.Type) {
					fieldSchema = nullableSchema(fieldSchema)
				}
				properties[propName] = fieldSchema
				if meta.Required || !isPointerLike(field.Type) && propName != "" && !hasOptionalTag(field) {
					required = append(required, propName)
				}
			}
		}
		sort.Strings(required)
		return &Schema{Type: "object", Properties: properties, Required: unique(required)}
	case *ast.ArrayType:
		return &Schema{Type: "array", Items: inlineExprSchema(value.Elt)}
	case *ast.MapType:
		return &Schema{Type: "object", AdditionalProperties: inlineExprSchema(value.Value)}
	case *ast.InterfaceType:
		return &Schema{Type: "object"}
	case *ast.Ident:
		return scalarSchema(value.Name)
	case *ast.SelectorExpr:
		if ident, ok := value.X.(*ast.Ident); ok {
			if ident.Name == "multipart" && value.Sel.Name == "FileHeader" {
				return &Schema{Type: "string", Format: "binary"}
			}
			if ident.Name == "io" && value.Sel.Name == "Reader" {
				return &Schema{Type: "string", Format: "binary"}
			}
		}
		return &Schema{Type: "object"}
	default:
		return &Schema{Type: "object"}
	}
}

func hasRequiredValidate(validate string) bool {
	if validate == "" {
		return false
	}
	for _, part := range strings.Split(validate, ",") {
		if strings.TrimSpace(part) == "required" {
			return true
		}
	}
	return false
}

func hasOptionalTag(field *ast.Field) bool {
	if field.Tag == nil {
		return false
	}
	tagValue := strings.Trim(field.Tag.Value, "`")
	tag := reflect.StructTag(tagValue)
	jsonTag := tag.Get("json")
	for _, part := range strings.Split(jsonTag, ",") {
		if part == "omitempty" {
			return true
		}
	}
	return false
}

func isPointerType(expr ast.Expr) bool {
	_, ok := expr.(*ast.StarExpr)
	return ok
}

func nullableSchema(schema *Schema) *Schema {
	if schema == nil {
		return &Schema{Type: "object", Nullable: true}
	}
	cloned := cloneSchema(schema)
	if cloned.Ref != "" {
		return &Schema{
			AllOf:    []*Schema{{Ref: cloned.Ref}},
			Nullable: true,
		}
	}
	cloned.Nullable = true
	return cloned
}

func cloneSchema(schema *Schema) *Schema {
	if schema == nil {
		return nil
	}
	cloned := *schema
	if len(schema.AllOf) > 0 {
		cloned.AllOf = make([]*Schema, len(schema.AllOf))
		for i, item := range schema.AllOf {
			cloned.AllOf[i] = cloneSchema(item)
		}
	}
	if schema.Properties != nil {
		cloned.Properties = make(map[string]*Schema, len(schema.Properties))
		for key, value := range schema.Properties {
			cloned.Properties[key] = cloneSchema(value)
		}
	}
	if schema.Items != nil {
		cloned.Items = cloneSchema(schema.Items)
	}
	if schema.AdditionalProperties != nil {
		cloned.AdditionalProperties = cloneSchema(schema.AdditionalProperties)
	}
	if schema.Required != nil {
		cloned.Required = append([]string(nil), schema.Required...)
	}
	return &cloned
}

func HasBinary(schema *Schema, components map[string]*Schema) bool {
	if schema == nil {
		return false
	}
	if schema.Format == "binary" {
		return true
	}
	if schema.Ref != "" {
		name := strings.TrimPrefix(schema.Ref, "#/components/schemas/")
		return HasBinary(components[name], components)
	}
	if schema.Items != nil && HasBinary(schema.Items, components) {
		return true
	}
	if schema.AdditionalProperties != nil && HasBinary(schema.AdditionalProperties, components) {
		return true
	}
	for _, prop := range schema.Properties {
		if HasBinary(prop, components) {
			return true
		}
	}
	return false
}
