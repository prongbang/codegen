package analyzer

import (
	"fmt"
	"go/ast"
	"go/token"
	"path"
	"strconv"
	"strings"

	"github.com/prongbang/codegen/internal/loader"
)

var httpMethods = map[string]bool{
	"GET":     true,
	"POST":    true,
	"PUT":     true,
	"PATCH":   true,
	"DELETE":  true,
	"OPTIONS": true,
	"HEAD":    true,
}

func AnalyzeFiber(mod *loader.Module) ([]Operation, error) {
	routes := []Operation{}
	for _, pkg := range mod.Packages {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Body == nil {
					continue
				}
				groupPaths := map[string]string{}
				for _, field := range fn.Type.Params.List {
					if isFiberApp(field.Type) {
						for _, name := range field.Names {
							groupPaths[name.Name] = ""
						}
					}
				}
				if len(groupPaths) == 0 {
					continue
				}
				ops, err := analyzeRouteBlock(pkg, fn.Body.List, groupPaths)
				if err != nil {
					return nil, err
				}
				routes = append(routes, ops...)
			}
		}
	}
	return routes, nil
}

func analyzeRouteBlock(pkg *loader.Package, stmts []ast.Stmt, groupPaths map[string]string) ([]Operation, error) {
	ops := []Operation{}
	for _, stmt := range stmts {
		switch value := stmt.(type) {
		case *ast.AssignStmt:
			for idx, rhs := range value.Rhs {
				call, ok := rhs.(*ast.CallExpr)
				if !ok {
					continue
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || sel.Sel.Name != "Group" {
					continue
				}
				receiver, ok := sel.X.(*ast.Ident)
				if !ok {
					continue
				}
				base, ok := groupPaths[receiver.Name]
				if !ok || len(call.Args) == 0 {
					continue
				}
				groupPath, ok := stringLiteral(call.Args[0])
				if !ok || idx >= len(value.Lhs) {
					continue
				}
				name, ok := value.Lhs[idx].(*ast.Ident)
				if !ok {
					continue
				}
				groupPaths[name.Name] = joinPath(base, groupPath)
			}
		case *ast.BlockStmt:
			nestedPaths := copyGroupPaths(groupPaths)
			nestedOps, err := analyzeRouteBlock(pkg, value.List, nestedPaths)
			if err != nil {
				return nil, err
			}
			ops = append(ops, nestedOps...)
		case *ast.ExprStmt:
			call, ok := value.X.(*ast.CallExpr)
			if !ok {
				continue
			}
			op, found, err := analyzeRouteCall(pkg, call, groupPaths)
			if err != nil {
				return nil, err
			}
			if found {
				ops = append(ops, op)
			}
		}
	}
	return ops, nil
}

func analyzeRouteCall(pkg *loader.Package, call *ast.CallExpr, groupPaths map[string]string) (Operation, bool, error) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return Operation{}, false, nil
	}

	method := strings.ToUpper(sel.Sel.Name)
	if !httpMethods[method] {
		return Operation{}, false, nil
	}

	receiver, ok := sel.X.(*ast.Ident)
	if !ok {
		return Operation{}, false, nil
	}
	basePath, ok := groupPaths[receiver.Name]
	if !ok {
		return Operation{}, false, nil
	}
	if len(call.Args) < 2 {
		return Operation{}, false, nil
	}

	routePath, ok := stringLiteral(call.Args[0])
	if !ok {
		return Operation{}, false, nil
	}
	handlerMethod, ok := handlerMethodName(call.Args[len(call.Args)-1])
	if !ok {
		return Operation{}, false, nil
	}

	requestType, directResponseType, usecaseMethod, err := analyzeHandler(pkg, handlerMethod)
	if err != nil {
		return Operation{}, false, err
	}
	responseType := directResponseType
	if usecaseMethod != "" {
		var fallbackRequest *TypeRef
		var err error
		responseType, fallbackRequest, err = analyzeUsecase(pkg, usecaseMethod)
		if err != nil {
			return Operation{}, false, err
		}
		if requestType == nil {
			requestType = fallbackRequest
		}
	}

	fullPath := joinPath(basePath, routePath)
	return Operation{
		Method:      strings.ToLower(method),
		Path:        fullPath,
		Package:     pkg.ImportPath,
		Handler:     handlerMethod,
		Request:     requestType,
		Response:    responseType,
		OperationID: path.Base(pkg.ImportPath) + "_" + handlerMethod,
		Tags:        []string{path.Base(pkg.ImportPath)},
	}, true, nil
}

func analyzeHandler(pkg *loader.Package, methodName string) (*TypeRef, *TypeRef, string, error) {
	if requestType, responseType, usecaseMethod, ok := analyzeHandlerMethod(pkg, methodName, true); ok {
		return requestType, responseType, usecaseMethod, nil
	}
	if requestType, responseType, usecaseMethod, ok := analyzeHandlerMethod(pkg, methodName, false); ok {
		return requestType, responseType, usecaseMethod, nil
	}
	return nil, nil, "", fmt.Errorf("handler method %s not found", methodName)
}

func analyzeHandlerMethod(pkg *loader.Package, methodName string, preferHandlerReceiver bool) (*TypeRef, *TypeRef, string, bool) {
	for _, file := range pkg.Files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != methodName || fn.Body == nil || fn.Recv == nil {
				continue
			}
			if preferHandlerReceiver != isHandlerReceiver(fn) {
				continue
			}

			valueTypes := map[string]*TypeRef{}
			var requestType *TypeRef
			var responseType *TypeRef
			var usecaseMethod string
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				switch value := n.(type) {
				case *ast.AssignStmt:
					for idx, lhs := range value.Lhs {
						ident, ok := lhs.(*ast.Ident)
						if !ok || idx >= len(value.Rhs) {
							continue
						}
						if typ := trackedTypeFromExpr(pkg, file, value.Rhs[idx], valueTypes); typ != nil {
							valueTypes[ident.Name] = typ
							if ident.Name == "request" || requestType == nil {
								requestType = typ
							}
						}
						if method, req := analyzeUsecaseCallExpr(pkg, file, value.Rhs[idx], valueTypes); method != "" {
							usecaseMethod = method
							if requestType == nil {
								requestType = req
							}
						}
					}
				case *ast.ValueSpec:
					for idx, name := range value.Names {
						if idx >= len(value.Values) {
							continue
						}
						if typ := trackedTypeFromExpr(pkg, file, value.Values[idx], valueTypes); typ != nil {
							valueTypes[name.Name] = typ
							if name.Name == "request" || requestType == nil {
								requestType = typ
							}
						}
						if method, req := analyzeUsecaseCallExpr(pkg, file, value.Values[idx], valueTypes); method != "" {
							usecaseMethod = method
							if requestType == nil {
								requestType = req
							}
						}
					}
				case *ast.ReturnStmt:
					if method, req := analyzeUsecaseCall(pkg, file, value.Results, valueTypes); method != "" {
						usecaseMethod = method
						if requestType == nil {
							requestType = req
						}
						return false
					}
					if responseType == nil {
						responseType = analyzeDirectResponse(pkg, file, value.Results, valueTypes)
					}
				case *ast.FuncLit:
					ast.Inspect(value.Body, func(inner ast.Node) bool {
						switch nested := inner.(type) {
						case *ast.AssignStmt:
							for idx, lhs := range nested.Lhs {
								ident, ok := lhs.(*ast.Ident)
								if !ok || idx >= len(nested.Rhs) {
									continue
								}
								if typ := trackedTypeFromExpr(pkg, file, nested.Rhs[idx], valueTypes); typ != nil {
									valueTypes[ident.Name] = typ
									if ident.Name == "request" || requestType == nil {
										requestType = typ
									}
								}
								if method, req := analyzeUsecaseCallExpr(pkg, file, nested.Rhs[idx], valueTypes); method != "" {
									usecaseMethod = method
									if requestType == nil {
										requestType = req
									}
								}
							}
							return usecaseMethod == ""
						case *ast.ReturnStmt:
							if method, req := analyzeUsecaseCall(pkg, file, nested.Results, valueTypes); method != "" {
								usecaseMethod = method
								if requestType == nil {
									requestType = req
								}
								return false
							}
							if responseType == nil {
								responseType = analyzeDirectResponse(pkg, file, nested.Results, valueTypes)
							}
							return true
						case *ast.ValueSpec:
							for idx, name := range nested.Names {
								if idx >= len(nested.Values) {
									continue
								}
								if typ := trackedTypeFromExpr(pkg, file, nested.Values[idx], valueTypes); typ != nil {
									valueTypes[name.Name] = typ
									if name.Name == "request" || requestType == nil {
										requestType = typ
									}
								}
								if method, req := analyzeUsecaseCallExpr(pkg, file, nested.Values[idx], valueTypes); method != "" {
									usecaseMethod = method
									if requestType == nil {
										requestType = req
									}
								}
							}
							return usecaseMethod == ""
						default:
							return true
						}
					})
				}
				return usecaseMethod == ""
			})

			return requestType, responseType, usecaseMethod, true
		}
	}
	return nil, nil, "", false
}

func analyzeUsecaseCall(pkg *loader.Package, file *ast.File, results []ast.Expr, valueTypes map[string]*TypeRef) (string, *TypeRef) {
	for _, result := range results {
		if method, requestType := analyzeUsecaseCallExpr(pkg, file, result, valueTypes); method != "" {
			return method, requestType
		}
	}
	return "", nil
}

func analyzeUsecaseCallExpr(pkg *loader.Package, file *ast.File, expr ast.Expr, valueTypes map[string]*TypeRef) (string, *TypeRef) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return "", nil
	}
	method, ok := usecaseMethodName(call.Fun)
	if !ok {
		return "", nil
	}
	for _, arg := range call.Args {
		if ident, ok := arg.(*ast.Ident); ok {
			if requestType, found := valueTypes[ident.Name]; found {
				return method, requestType
			}
		}
	}
	if len(call.Args) > 0 {
		last := call.Args[len(call.Args)-1]
		if requestType := requestTypeFromExpr(pkg, file, last); requestType != nil {
			return method, requestType
		}
	}
	return method, nil
}

func analyzeDirectResponse(pkg *loader.Package, file *ast.File, results []ast.Expr, valueTypes map[string]*TypeRef) *TypeRef {
	for _, result := range results {
		if ref := responseTypeFromExpr(pkg, file, result, valueTypes); ref != nil {
			return ref
		}
		if ref := trackedTypeFromExpr(pkg, file, result, valueTypes); ref != nil {
			return ref
		}
	}
	return nil
}

func responseTypeFromExpr(pkg *loader.Package, file *ast.File, expr ast.Expr, valueTypes map[string]*TypeRef) *TypeRef {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return nil
	}

	switch fun := call.Fun.(type) {
	case *ast.SelectorExpr:
		if fun.Sel.Name == "JSON" && len(call.Args) > 0 {
			return inferResponseArgType(pkg, file, call.Args[0], valueTypes)
		}
		if isResponseHelper(fun) && len(call.Args) > 1 {
			return inferResponseArgType(pkg, file, call.Args[1], valueTypes)
		}
		if innerCall, ok := fun.X.(*ast.CallExpr); ok && len(call.Args) > 0 {
			innerSel, ok := innerCall.Fun.(*ast.SelectorExpr)
			if ok && innerSel.Sel.Name == "Status" {
				return inferResponseArgType(pkg, file, call.Args[0], valueTypes)
			}
		}
	}

	return nil
}

func inferResponseArgType(pkg *loader.Package, file *ast.File, expr ast.Expr, valueTypes map[string]*TypeRef) *TypeRef {
	switch value := expr.(type) {
	case *ast.Ident:
		if ref, ok := valueTypes[value.Name]; ok {
			return ref
		}
		if value.Name == "nil" {
			return nil
		}
		return ExprToTypeRef(pkg, file, value)
	case *ast.UnaryExpr:
		if value.Op == token.AND {
			if lit, ok := value.X.(*ast.CompositeLit); ok {
				return ExprToTypeRef(pkg, file, lit.Type)
			}
		}
	case *ast.CompositeLit:
		return ExprToTypeRef(pkg, file, value.Type)
	case *ast.CallExpr:
		if ref := responseTypeFromExpr(pkg, file, value, valueTypes); ref != nil {
			return ref
		}
	case *ast.BasicLit:
		switch value.Kind {
		case token.STRING:
			return &TypeRef{Kind: TypeScalar, Name: "string"}
		case token.INT:
			return &TypeRef{Kind: TypeScalar, Name: "int"}
		case token.FLOAT:
			return &TypeRef{Kind: TypeScalar, Name: "float64"}
		}
	}
	return nil
}

func trackedTypeFromExpr(pkg *loader.Package, file *ast.File, expr ast.Expr, valueTypes map[string]*TypeRef) *TypeRef {
	if ref := requestTypeFromExpr(pkg, file, expr); ref != nil {
		return ref
	}
	switch value := expr.(type) {
	case *ast.Ident:
		if ref, ok := valueTypes[value.Name]; ok {
			return ref
		}
		if value.Name == "nil" {
			return nil
		}
		return ExprToTypeRef(pkg, file, value)
	case *ast.CallExpr:
		if ref := responseTypeFromExpr(pkg, file, value, valueTypes); ref != nil {
			return ref
		}
	}
	return nil
}

func isResponseHelper(sel *ast.SelectorExpr) bool {
	switch sel.Sel.Name {
	case "Ok", "Created", "BadRequest", "NotFound", "NoContent", "Unauthorized", "Forbidden", "JSON":
		return true
	default:
		return false
	}
}

func requestTypeFromExpr(pkg *loader.Package, file *ast.File, expr ast.Expr) *TypeRef {
	switch value := expr.(type) {
	case *ast.UnaryExpr:
		if value.Op == token.AND {
			if lit, ok := value.X.(*ast.CompositeLit); ok {
				return ExprToTypeRef(pkg, file, lit.Type)
			}
		}
	case *ast.CompositeLit:
		return ExprToTypeRef(pkg, file, value.Type)
	}
	return nil
}

func analyzeUsecase(pkg *loader.Package, methodName string) (*TypeRef, *TypeRef, error) {
	for _, file := range pkg.Files {
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if typeSpec.Name.Name != "UseCase" {
					continue
				}
				iface, ok := typeSpec.Type.(*ast.InterfaceType)
				if !ok {
					continue
				}
				for _, field := range iface.Methods.List {
					if len(field.Names) == 0 || field.Names[0].Name != methodName {
						continue
					}
					fnType, ok := field.Type.(*ast.FuncType)
					if !ok {
						continue
					}
					requestType := requestParamType(pkg, file, fnType)
					responseType := firstResultType(pkg, file, fnType)
					return responseType, requestType, nil
				}
			}
		}
	}

	for _, file := range pkg.Files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != methodName || fn.Type == nil {
				continue
			}
			requestType := requestParamType(pkg, file, fn.Type)
			responseType := firstResultType(pkg, file, fn.Type)
			return responseType, requestType, nil
		}
	}

	return nil, nil, fmt.Errorf("usecase method %s not found", methodName)
}

func requestParamType(pkg *loader.Package, file *ast.File, fnType *ast.FuncType) *TypeRef {
	if fnType.Params == nil {
		return nil
	}
	params := fnType.Params.List
	if len(params) == 0 {
		return nil
	}
	return ExprToTypeRef(pkg, file, params[len(params)-1].Type)
}

func firstResultType(pkg *loader.Package, file *ast.File, fnType *ast.FuncType) *TypeRef {
	if fnType.Results == nil || len(fnType.Results.List) == 0 {
		return nil
	}
	return ExprToTypeRef(pkg, file, fnType.Results.List[0].Type)
}

func isFiberApp(expr ast.Expr) bool {
	star, ok := expr.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "fiber" && sel.Sel.Name == "App"
}

func handlerMethodName(expr ast.Expr) (string, bool) {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	return sel.Sel.Name, true
}

func usecaseMethodName(expr ast.Expr) (string, bool) {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	switch parent := sel.X.(type) {
	case *ast.SelectorExpr:
		if !strings.EqualFold(parent.Sel.Name, "uc") && !strings.EqualFold(parent.Sel.Name, "usecase") {
			return "", false
		}
		return sel.Sel.Name, true
	case *ast.Ident:
		if strings.EqualFold(parent.Name, "uc") || strings.EqualFold(parent.Name, "usecase") {
			return sel.Sel.Name, true
		}
	}
	return "", false
}

func isHandlerReceiver(fn *ast.FuncDecl) bool {
	if fn == nil || fn.Recv == nil || len(fn.Recv.List) == 0 {
		return false
	}
	name := receiverTypeName(fn.Recv.List[0].Type)
	name = strings.ToLower(name)
	return name == "handler" || strings.HasSuffix(name, "handler")
}

func receiverTypeName(expr ast.Expr) string {
	switch value := expr.(type) {
	case *ast.StarExpr:
		return receiverTypeName(value.X)
	case *ast.Ident:
		return value.Name
	case *ast.IndexExpr:
		return receiverTypeName(value.X)
	case *ast.IndexListExpr:
		return receiverTypeName(value.X)
	default:
		return ""
	}
}

func copyGroupPaths(groupPaths map[string]string) map[string]string {
	clone := map[string]string{}
	for key, value := range groupPaths {
		clone[key] = value
	}
	return clone
}

func stringLiteral(expr ast.Expr) (string, bool) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	value, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return value, true
}

func joinPath(base, value string) string {
	switch {
	case base == "":
		return value
	case value == "":
		return base
	case strings.HasSuffix(base, "/") && strings.HasPrefix(value, "/"):
		return base + strings.TrimPrefix(value, "/")
	case !strings.HasSuffix(base, "/") && !strings.HasPrefix(value, "/"):
		return base + "/" + value
	default:
		return base + value
	}
}
