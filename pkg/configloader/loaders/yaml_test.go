package loaders

import (
	"os"
	"reflect"
	"testing"
)

func TestYamlLoader_LoadValue(t *testing.T) {
	yamlContent := `
server:
  host: localhost
  port: 8080
database:
  host: postgres.example.com
  port: 5432
  name: myapp
  user: admin
  password: secret123
debug: true
timeout: 30s
features:
  - auth
  - logging
  - metrics
ports:
  - 8080
  - 8081
  - 8082
`

	tmpFile, err := os.CreateTemp("", "test_*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(yamlContent)); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	loader, err := NewYamlLoader(tmpFile.Name())
	if err != nil {
		t.Fatalf("NewYamlLoader error: %v", err)
	}

	tests := []struct {
		name      string
		tagValue  string
		fieldType reflect.Type
		expected  any
		wantOk    bool
		wantErr   bool
	}{
		{
			name:      "string value",
			tagValue:  "server.host",
			fieldType: reflect.TypeFor[string](),
			expected:  "localhost",
			wantOk:    true,
			wantErr:   false,
		},
		{
			name:      "int value",
			tagValue:  "server.port",
			fieldType: reflect.TypeFor[int](),
			expected:  8080,
			wantOk:    true,
			wantErr:   false,
		},
		{
			name:      "bool value",
			tagValue:  "debug",
			fieldType: reflect.TypeOf(false),
			expected:  true,
			wantOk:    true,
			wantErr:   false,
		},
		{
			name:      "nested string",
			tagValue:  "database.host",
			fieldType: reflect.TypeFor[string](),
			expected:  "postgres.example.com",
			wantOk:    true,
			wantErr:   false,
		},
		{
			name:      "nested int",
			tagValue:  "database.port",
			fieldType: reflect.TypeFor[int](),
			expected:  5432,
			wantOk:    true,
			wantErr:   false,
		},
		{
			name:      "slice of strings",
			tagValue:  "features",
			fieldType: reflect.TypeOf([]string{}),
			expected:  []string{"auth", "logging", "metrics"},
			wantOk:    true,
			wantErr:   false,
		},
		{
			name:      "slice of ints",
			tagValue:  "ports",
			fieldType: reflect.TypeOf([]int{}),
			expected:  []int{8080, 8081, 8082},
			wantOk:    true,
			wantErr:   false,
		},
		{
			name:      "non-existent key",
			tagValue:  "nonexistent",
			fieldType: reflect.TypeFor[string](),
			expected:  nil,
			wantOk:    false,
			wantErr:   false,
		},
		{
			name:      "empty tag value",
			tagValue:  "",
			fieldType: reflect.TypeFor[string](),
			expected:  nil,
			wantOk:    false,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := reflect.New(tt.fieldType).Elem()
			structField := reflect.StructField{
				Name: "TestField",
				Type: tt.fieldType,
			}

			ok, err := loader.LoadValue(tt.tagValue, field, structField)

			if (err != nil) != tt.wantErr {
				t.Errorf("LoadValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if ok != tt.wantOk {
				t.Errorf("LoadValue() ok = %v, wantOk %v", ok, tt.wantOk)
				return
			}

			if tt.wantOk {
				got := field.Interface()
				if !reflect.DeepEqual(got, tt.expected) {
					t.Errorf("LoadValue() got = %v, expected %v", got, tt.expected)
				}
			}
		})
	}
}

func TestYamlLoader_FileNotFound(t *testing.T) {
	_, err := NewYamlLoader("nonexistent_file.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestYamlLoader_InvalidYAML(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "invalid_*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte("invalid: yaml: [bad")); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	_, err = NewYamlLoader(tmpFile.Name())
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}

func TestYamlLoader_EmptyFilePath(t *testing.T) {
	_, err := NewYamlLoader("")
	if err == nil {
		t.Error("Expected error for empty file path, got nil")
	}
}
