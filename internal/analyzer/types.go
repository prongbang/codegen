package analyzer

import (
	"go/ast"
	"strings"

	"github.com/prongbang/codegen/internal/loader"
)

type TypeKind string

const (
	TypeInvalid TypeKind = "invalid"
	TypeNamed   TypeKind = "named"
	TypeArray   TypeKind = "array"
	TypeMap     TypeKind = "map"
	TypeObject  TypeKind = "object"
	TypeAny     TypeKind = "any"
	TypeScalar  TypeKind = "scalar"
)

type TypeRef struct {
	Kind    TypeKind
	Package string
	Name    string
	Elem    *TypeRef
	TypeArgs []*TypeRef
	Expr    ast.Expr
}

type Operation struct {
	Method      string
	Path        string
	Package     string
	Handler     string
	Request     *TypeRef
	Response    *TypeRef
	OperationID string
	Tags        []string
}

func ExprToTypeRef(pkg *loader.Package, file *ast.File, expr ast.Expr) *TypeRef {
	switch value := expr.(type) {
	case *ast.StarExpr:
		return ExprToTypeRef(pkg, file, value.X)
	case *ast.ArrayType:
		return &TypeRef{Kind: TypeArray, Elem: ExprToTypeRef(pkg, file, value.Elt), Expr: expr}
	case *ast.MapType:
		return &TypeRef{Kind: TypeMap, Elem: ExprToTypeRef(pkg, file, value.Value), Expr: expr}
	case *ast.InterfaceType:
		return &TypeRef{Kind: TypeAny, Expr: expr}
	case *ast.StructType:
		return &TypeRef{Kind: TypeObject, Expr: expr}
	case *ast.Ident:
		if isScalar(value.Name) {
			return &TypeRef{Kind: TypeScalar, Name: value.Name, Expr: expr}
		}
		return &TypeRef{Kind: TypeNamed, Package: pkg.ImportPath, Name: value.Name, Expr: expr}
	case *ast.SelectorExpr:
		if ident, ok := value.X.(*ast.Ident); ok {
			importPath := importPathForAlias(file, ident.Name)
			if importPath == "" {
				return &TypeRef{Kind: TypeNamed, Name: value.Sel.Name, Expr: expr}
			}
			return &TypeRef{Kind: TypeNamed, Package: importPath, Name: value.Sel.Name, Expr: expr}
		}
	case *ast.IndexExpr:
		ref := ExprToTypeRef(pkg, file, value.X)
		if ref != nil {
			ref.TypeArgs = []*TypeRef{ExprToTypeRef(pkg, file, value.Index)}
			ref.Expr = expr
		}
		return ref
	case *ast.IndexListExpr:
		ref := ExprToTypeRef(pkg, file, value.X)
		if ref != nil {
			typeArgs := make([]*TypeRef, 0, len(value.Indices))
			for _, index := range value.Indices {
				typeArgs = append(typeArgs, ExprToTypeRef(pkg, file, index))
			}
			ref.TypeArgs = typeArgs
			ref.Expr = expr
		}
		return ref
	case *ast.Ellipsis:
		return &TypeRef{Kind: TypeArray, Elem: ExprToTypeRef(pkg, file, value.Elt), Expr: expr}
	}
	return &TypeRef{Kind: TypeInvalid, Expr: expr}
}

func importPathForAlias(file *ast.File, alias string) string {
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, "\"")
		if imp.Name != nil {
			if imp.Name.Name == alias {
				return path
			}
			continue
		}
		parts := strings.Split(path, "/")
		if parts[len(parts)-1] == alias {
			return path
		}
	}
	return ""
}

func isScalar(name string) bool {
	switch name {
	case "string", "bool", "byte", "rune",
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64":
		return true
	}
	return false
}
