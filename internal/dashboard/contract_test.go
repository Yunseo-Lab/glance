package dashboard

import (
	"gopkg.in/yaml.v3"
	"os"
	"reflect"
	"slices"
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
	for name, model := range map[string]any{"Board": Board{}, "Widget": Widget{}, "Source": Source{}, "FeedData": FeedData{}, "WeatherData": WeatherData{}, "Snapshot": Snapshot{}, "Link": Link{}, "Error": ErrorResponse{}} {
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
