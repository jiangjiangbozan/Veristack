// Package openapi_test 校验 OpenAPI 契约与真实后端路由、DTO 字段逐项一致。
//
// 这些检查放在 go test 里，CI 只需执行 go test ./... 就能阻止契约漂移。
package openapi_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"api/internal/change"
	"api/internal/identity"
	"api/internal/platform/errs"
	"api/internal/platform/response"
	"api/internal/platform/views"
	"api/internal/project"
	"api/internal/scope"
)

const openAPIFile = "openapi.yaml"

// schemaTypeRegistry 明确每个组件 schema 对应的 Go 类型。
//
// 不按名字自动猜测，避免重命名后测试静默跳过；新增组件时必须同时登记。
var schemaTypeRegistry = map[string]reflect.Type{
	"AppError":              reflect.TypeOf(errs.AppError{}),
	"AuditEvent":            reflect.TypeOf(views.AuditEventView{}),
	"AuthResult":            reflect.TypeOf(identity.LoginResponse{}),
	"Baseline":              reflect.TypeOf(views.BaselineView{}),
	"BaselineDiff":          reflect.TypeOf(scope.DiffBaselineResponse{}),
	"ChangeRequest":         reflect.TypeOf(views.ChangeRequestView{}),
	"CreateChangeRequest":   reflect.TypeOf(change.CreateChangeRequestRequest{}),
	"CreateProjectRequest":  reflect.TypeOf(project.CreateProjectRequest{}),
	"Envelope":              reflect.TypeOf(response.Envelope{}),
	"ImportMilestone":       reflect.TypeOf(project.ImportMilestone{}),
	"ImportProjectRequest":  reflect.TypeOf(project.ImportProjectRequest{}),
	"ImportProjectResponse": reflect.TypeOf(project.ImportProjectResponse{}),
	"Member":                reflect.TypeOf(project.MemberView{}),
	"Notification":          reflect.TypeOf(views.NotificationView{}),
	"Page":                  reflect.TypeOf(response.Page{}),
	"Project":               reflect.TypeOf(views.ProjectView{}),
	"ProjectOverview":       reflect.TypeOf(project.ProjectOverviewResponse{}),
	"RegisterRequest":       reflect.TypeOf(identity.RegisterRequest{}),
	"Requirement":           reflect.TypeOf(views.RequirementView{}),
	"RequirementInput":      reflect.TypeOf(scope.RequirementInput{}),
	"UpdateProjectRequest":  reflect.TypeOf(project.UpdateProjectRequest{}),
	"User":                  reflect.TypeOf(views.UserView{}),
}

// httpMethods 是 OpenAPI Path Item 中允许的 HTTP 方法键。
var httpMethods = map[string]bool{
	"delete":  true,
	"get":     true,
	"head":    true,
	"options": true,
	"patch":   true,
	"post":    true,
	"put":     true,
	"trace":   true,
}

// routeMethods 是 handler 中注册路由时使用的方法名。
var routeMethods = map[string]string{
	"Delete": "DELETE",
	"Get":    "GET",
	"Patch":  "PATCH",
	"Post":   "POST",
	"Put":    "PUT",
}

// TestOpenAPISchemaFieldsMatchDTOs 逐字段比对组件 schema 与 Go DTO 的 JSON 标签。
func TestOpenAPISchemaFieldsMatchDTOs(t *testing.T) {
	document := loadDocument(t)
	components := mapValue(document["components"])
	schemas := mapValue(components["schemas"])

	if len(schemas) != len(schemaTypeRegistry) {
		t.Fatalf("组件 schema 数量为 %d，登记 Go 类型 %d；新增或删除组件必须同步登记", len(schemas), len(schemaTypeRegistry))
	}
	for name := range schemaTypeRegistry {
		if _, ok := schemas[name]; !ok {
			t.Errorf("登记了 Go 类型但 OpenAPI 缺少组件 %s", name)
		}
	}
	for name := range schemas {
		if _, ok := schemaTypeRegistry[name]; !ok {
			t.Errorf("OpenAPI 组件 %s 未登记对应 Go 类型，无法防止字段漂移", name)
		}
	}

	for name, typ := range schemaTypeRegistry {
		t.Run(name, func(t *testing.T) {
			schema, ok := schemas[name].(map[string]any)
			if !ok {
				t.Fatalf("组件 %s 不是对象", name)
			}
			checkSchemaType(t, name, schema, typ, schemas, map[string]bool{})
		})
	}
}

// TestOpenAPIRoutesMatchHandlers 比对 OpenAPI 路径/操作与 handler 注册的路由。
func TestOpenAPIRoutesMatchHandlers(t *testing.T) {
	document := loadDocument(t)
	openAPIRoutes := collectOpenAPIRoutes(t, document)
	handlerRoutes := collectHandlerRoutes(t)

	if len(openAPIRoutes) != 69 {
		t.Errorf("OpenAPI 操作数为 %d，期望 69；修改路由后请同步更新契约与计数", len(openAPIRoutes))
	}
	if len(handlerRoutes) != len(openAPIRoutes) {
		t.Fatalf("Handler 路由数 %d 与 OpenAPI 操作数 %d 不一致", len(handlerRoutes), len(openAPIRoutes))
	}
	assertSameRoutes(t, openAPIRoutes, handlerRoutes)
}

// checkSchemaType 递归检查一个 schema 与 Go 结构体的字段集合。
func checkSchemaType(t *testing.T, schemaName string, schema map[string]any, typ reflect.Type, schemas map[string]any, visiting map[string]bool) {
	t.Helper()
	typ = indirectType(typ)
	if typ.Kind() != reflect.Struct {
		t.Fatalf("%s 映射的 Go 类型 %s 不是结构体", schemaName, typ)
	}
	if visiting[schemaName] {
		return
	}
	visiting[schemaName] = true

	properties, required := resolvedProperties(schema, schemas, map[string]bool{})
	goFields := jsonFields(typ)

	if len(properties) != len(goFields) {
		t.Errorf("%s 字段数量不一致：OpenAPI %d 个，Go %d 个；OpenAPI=%v Go=%v",
			schemaName, len(properties), len(goFields), sortedKeys(properties), sortedFieldKeys(goFields))
	}
	for name := range properties {
		if _, ok := goFields[name]; !ok {
			t.Errorf("%s 的 OpenAPI 字段 %s 在 Go 类型 %s 中不存在", schemaName, name, typ)
		}
	}
	for name, field := range goFields {
		if _, ok := properties[name]; !ok {
			t.Errorf("%s 的 Go 字段 %s 未写入 OpenAPI 组件", schemaName, name)
		}
		if field.Required && !required[name] {
			t.Errorf("%s 的 Go 字段 %s 标为 required，但 OpenAPI required 未包含", schemaName, name)
		}
	}

	for name, property := range properties {
		propertySchema := mapValue(property)
		refName := refName(propertySchema)
		if refName == "" {
			continue
		}
		goField, ok := goFields[name]
		if !ok {
			continue
		}
		refType, registered := schemaTypeRegistry[refName]
		if !registered {
			t.Errorf("%s.%s 引用的组件 %s 未登记 Go 类型", schemaName, name, refName)
			continue
		}
		refSchema, ok := schemas[refName].(map[string]any)
		if !ok {
			t.Errorf("组件 %s 不是对象", refName)
			continue
		}
		checkSchemaTypeForField(t, schemaName+"."+name, refSchema, refType, goField.Type, schemas, visiting)
	}
}

// checkSchemaTypeForField 处理数组字段：schema 的 items 引用与 Go 的切片元素类型对齐。
func checkSchemaTypeForField(t *testing.T, path string, refSchema map[string]any, _ reflect.Type, goType reflect.Type, schemas map[string]any, visiting map[string]bool) {
	t.Helper()
	goType = indirectType(goType)
	if goType.Kind() == reflect.Slice {
		checkSchemaType(t, path, refSchema, goType.Elem(), schemas, visiting)
		return
	}
	checkSchemaType(t, path, refSchema, goType, schemas, visiting)
}

// loadDocument 读取并解析 OpenAPI YAML。
func loadDocument(t *testing.T) map[string]any {
	t.Helper()
	content, err := os.ReadFile(openAPIFile)
	if err != nil {
		t.Fatalf("读取 %s 失败: %v", openAPIFile, err)
	}
	var document map[string]any
	if err := yaml.Unmarshal(content, &document); err != nil {
		t.Fatalf("解析 OpenAPI 失败: %v", err)
	}
	return document
}

// resolvedProperties 合并 allOf 并返回属性与 required 集合。
func resolvedProperties(schema map[string]any, schemas map[string]any, seen map[string]bool) (map[string]any, map[string]bool) {
	properties := map[string]any{}
	required := map[string]bool{}
	if schema == nil {
		return properties, required
	}
	if name := refName(schema); name != "" {
		if seen[name] {
			return properties, required
		}
		seen[name] = true
		if target, ok := schemas[name].(map[string]any); ok {
			return resolvedProperties(target, schemas, seen)
		}
		return properties, required
	}
	for _, item := range sliceValue(schema["allOf"]) {
		part, partRequired := resolvedProperties(mapValue(item), schemas, seen)
		for key, value := range part {
			properties[key] = value
		}
		for key := range partRequired {
			required[key] = true
		}
	}
	for key, value := range mapValue(schema["properties"]) {
		properties[key] = value
	}
	for _, item := range sliceValue(schema["required"]) {
		if name, ok := item.(string); ok {
			required[name] = true
		}
	}
	return properties, required
}

// jsonField 描述一个 Go 结构体字段的 JSON 类型与 required 标记。
type jsonField struct {
	Required bool
	Type     reflect.Type
}

// jsonFields 收集结构体的 JSON 字段名、类型与 required 标记。
func jsonFields(typ reflect.Type) map[string]jsonField {
	fields := map[string]jsonField{}
	for index := 0; index < typ.NumField(); index++ {
		field := typ.Field(index)
		if field.PkgPath != "" {
			continue
		}
		if field.Anonymous {
			for name, nested := range jsonFields(indirectType(field.Type)) {
				fields[name] = nested
			}
			continue
		}
		tag := field.Tag.Get("json")
		name := strings.Split(tag, ",")[0]
		if name == "-" || name == "" {
			continue
		}
		fields[name] = jsonField{
			Required: hasRequiredValidate(field.Tag.Get("validate")),
			Type:     field.Type,
		}
	}
	return fields
}

// hasRequiredValidate 判断 validate 标签是否要求字段必填。
func hasRequiredValidate(tag string) bool {
	for _, part := range strings.Split(tag, ",") {
		if part == "dive" {
			return false
		}
		if part == "required" {
			return true
		}
	}
	return false
}

// collectOpenAPIRoutes 提取 OpenAPI 中的「方法 + 路径」集合。
func collectOpenAPIRoutes(t *testing.T, document map[string]any) map[string]bool {
	t.Helper()
	routes := map[string]bool{}
	for path, item := range mapValue(document["paths"]) {
		for method := range mapValue(item) {
			if httpMethods[method] {
				routes[strings.ToUpper(method)+" "+path] = true
			}
		}
	}
	return routes
}

// collectHandlerRoutes 解析 handler.go 中的 Routes 方法，展开嵌套 Route 前缀。
func collectHandlerRoutes(t *testing.T) map[string]bool {
	t.Helper()
	root := apiRoot(t)
	routes := map[string]bool{"GET /healthz": true, "GET /readyz": true}
	err := filepath.Walk(filepath.Join(root, "internal"), func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() || info.Name() != "handler.go" {
			return nil
		}
		collectFileRoutes(t, path, routes)
		return nil
	})
	if err != nil {
		t.Fatalf("遍历 handler.go 失败: %v", err)
	}
	return routes
}

// collectFileRoutes 解析单个 handler.go 的路由。
func collectFileRoutes(t *testing.T, path string, routes map[string]bool) {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("解析 %s 失败: %v", path, err)
	}
	consts := collectStringConsts(file)
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Body == nil || !isRouteFunction(function.Name.Name) {
			continue
		}
		walkRouteBlock(t, function.Body, "", consts, routes)
	}
}

// walkRouteBlock 递归遍历路由注册语句。
func walkRouteBlock(t *testing.T, block *ast.BlockStmt, base string, consts map[string]string, routes map[string]bool) {
	t.Helper()
	ast.Inspect(block, func(node ast.Node) bool {
		if _, ok := node.(*ast.FuncLit); ok {
			return false
		}
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		receiver, ok := selector.X.(*ast.Ident)
		if !ok || receiver.Name != "r" {
			return true
		}
		if len(call.Args) == 0 {
			return true
		}
		path, ok := routePath(call.Args[0], consts)
		if !ok {
			return true
		}
		if method, ok := routeMethods[selector.Sel.Name]; ok {
			routes[method+" "+joinRoute(base, path)] = true
			return true
		}
		if selector.Sel.Name != "Route" && selector.Sel.Name != "Group" {
			return true
		}
		for _, argument := range call.Args[1:] {
			if function, ok := argument.(*ast.FuncLit); ok {
				nextBase := base
				if selector.Sel.Name == "Route" {
					nextBase = joinRoute(base, path)
				}
				walkRouteBlock(t, function.Body, nextBase, consts, routes)
			}
		}
		return true
	})
}

// routePath 读取字符串字面量或同文件字符串常量。
func routePath(expression ast.Expr, consts map[string]string) (string, bool) {
	switch value := expression.(type) {
	case *ast.BasicLit:
		if value.Kind == token.STRING {
			path, err := strconv.Unquote(value.Value)
			return path, err == nil && strings.HasPrefix(path, "/")
		}
	case *ast.Ident:
		path, ok := consts[value.Name]
		return path, ok
	}
	return "", false
}

// collectStringConsts 收集文件级的字符串常量。
func collectStringConsts(file *ast.File) map[string]string {
	consts := map[string]string{}
	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.CONST {
			continue
		}
		for _, spec := range general.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for index, name := range valueSpec.Names {
				if index >= len(valueSpec.Values) {
					continue
				}
				literal, ok := valueSpec.Values[index].(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					continue
				}
				consts[name.Name] = strings.Trim(literal.Value, "\"")
			}
		}
	}
	return consts
}

// joinRoute 拼接嵌套路由前缀，根路径归一化为前缀本身。
func joinRoute(base, path string) string {
	if path == "/" || path == "" {
		if base == "" {
			return "/"
		}
		return base
	}
	if base == "" {
		return path
	}
	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(path, "/")
}

// isRouteFunction 判断函数是否负责注册路由。
func isRouteFunction(name string) bool {
	return name == "Routes" || strings.HasSuffix(name, "Routes")
}

// assertSameRoutes 输出缺失与多余的路由，便于定位契约漂移。
func assertSameRoutes(t *testing.T, want, got map[string]bool) {
	t.Helper()
	for route := range want {
		if !got[route] {
			t.Errorf("OpenAPI 有但 Handler 未注册: %s", route)
		}
	}
	for route := range got {
		if !want[route] {
			t.Errorf("Handler 注册了但 OpenAPI 未声明: %s", route)
		}
	}
}

// apiRoot 通过测试文件位置定位 api 目录。
func apiRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("无法定位测试文件")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

// refName 提取本地组件引用名，非本地引用返回空。
func refName(schema map[string]any) string {
	raw, _ := schema["$ref"].(string)
	const prefix = "#/components/schemas/"
	if strings.HasPrefix(raw, prefix) {
		return strings.TrimPrefix(raw, prefix)
	}
	return ""
}

// indirectType 解引用指针类型。
func indirectType(typ reflect.Type) reflect.Type {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return typ
}

// mapValue 把 YAML 对象节点转为 map。
func mapValue(value any) map[string]any {
	result, _ := value.(map[string]any)
	return result
}

// sliceValue 把 YAML 数组节点转为切片。
func sliceValue(value any) []any {
	result, _ := value.([]any)
	return result
}

// sortedKeys 输出排序后的字段名，便于比较。
func sortedKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// sortedFieldKeys 输出排序后的 Go JSON 字段名。
func sortedFieldKeys(values map[string]jsonField) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
