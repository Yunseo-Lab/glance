package dashboard

import (
	"go/ast"
	"go/parser"
	"go/token"
	"gopkg.in/yaml.v3"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestOpenAPIModelsAndCatalogDoNotDrift(t *testing.T) {
	data, err := os.ReadFile("../../docs/dashboard-api.openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Components struct {
			Schemas map[string]struct {
				Properties map[string]struct {
					Enum []string `yaml:"enum"`
				} `yaml:"properties"`
				Required []string `yaml:"required"`
			} `yaml:"schemas"`
		} `yaml:"components"`
	}
	if err = yaml.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	for name, model := range map[string]any{"Board": Board{}, "Widget": Widget{}, "Source": Source{}, "FeedData": FeedData{}, "WeatherData": WeatherData{}, "Snapshot": Snapshot{}, "Link": Link{}, "Error": ErrorResponse{}, "Connection": Connection{}, "SourceStatus": SourceStatus{}, "ConnectionTestReport": ConnectionTestReport{}} {
		schema, ok := document.Components.Schemas[name]
		if !ok {
			t.Fatal("missing API model", name)
		}
		kind := reflect.TypeOf(model)
		for i := range kind.NumField() {
			field := kind.Field(i)
			parts := strings.Split(field.Tag.Get("json"), ",")
			if parts[0] == "" || parts[0] == "-" {
				continue
			}
			if _, ok := schema.Properties[parts[0]]; !ok {
				t.Errorf("%s.%s missing from OpenAPI", name, parts[0])
			}
			if !slices.Contains(parts, "omitempty") && !slices.Contains(schema.Required, parts[0]) {
				t.Errorf("required %s.%s missing from OpenAPI", name, parts[0])
			}
		}
	}
	for schemaName, definitions := range map[string][]string{"Widget": {}, "Source": {}} {
		if schemaName == "Widget" {
			for kind := range widgetDefinitions {
				definitions = append(definitions, kind)
			}
		} else {
			for kind := range providerDefinitions {
				definitions = append(definitions, kind)
			}
		}
		actual := document.Components.Schemas[schemaName].Properties["type"].Enum
		slices.Sort(definitions)
		slices.Sort(actual)
		if !slices.Equal(actual, definitions) {
			t.Errorf("%s type enum drift", schemaName)
		}
	}
}

func TestEveryRegisteredAPIHasAnOpenAPIContract(t *testing.T) {
	data, err := os.ReadFile("../../docs/dashboard-api.openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Paths map[string]map[string]any `yaml:"paths"`
	}
	if err = yaml.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"server.go", "auth.go", "boards_http.go", "connections_http.go"} {
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || len(call.Args) == 0 {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "HandleFunc" {
				return true
			}
			literal, ok := call.Args[0].(*ast.BasicLit)
			if !ok {
				return true
			}
			value, err := strconv.Unquote(literal.Value)
			if err != nil {
				t.Fatal(err)
			}
			parts := strings.Fields(value)
			if len(parts) != 2 || !strings.HasPrefix(parts[1], "/api/") {
				return true
			}
			if document.Paths[parts[1]][strings.ToLower(parts[0])] == nil {
				t.Errorf("undocumented API: %s", value)
			}
			return true
		})
	}
}
