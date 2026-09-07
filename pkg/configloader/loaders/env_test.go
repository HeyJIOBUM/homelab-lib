package loaders

import (
	"os"
	"reflect"
	"testing"
)

func TestEnvLoader_Name(t *testing.T) {
	loader := NewEnvLoader()
	if loader.Name() != "EnvLoader" {
		t.Errorf("Expected name 'EnvLoader', got '%s'", loader.Name())
	}
}

func TestEnvLoader_Priority(t *testing.T) {
	loader := NewEnvLoader()
	if loader.Priority() != 0 {
		t.Errorf("Expected priority 0, got %d", loader.Priority())
	}

	loaderWithOpts := NewEnvLoaderWithOptions("Custom", 50, "env", "PREFIX")
	if loaderWithOpts.Priority() != 50 {
		t.Errorf("Expected priority 50, got %d", loaderWithOpts.Priority())
	}
}

func TestEnvLoader_SupportedTag(t *testing.T) {
	loader := NewEnvLoader()
	if loader.SupportedTag() != "env" {
		t.Errorf("Expected supportedTag 'env', got '%s'", loader.SupportedTag())
	}

	loaderWithOpts := NewEnvLoaderWithOptions("Custom", 50, "custom", "PREFIX")
	if loaderWithOpts.SupportedTag() != "custom" {
		t.Errorf("Expected supportedTag 'custom', got '%s'", loaderWithOpts.SupportedTag())
	}
}

func TestEnvLoader_LoadValue(t *testing.T) {
	os.Setenv("TEST_STRING", "hello")
	os.Setenv("TEST_INT", "123")
	os.Setenv("TEST_BOOL", "true")
	os.Setenv("TEST_FLOAT", "3.14")
	defer func() {
		os.Unsetenv("TEST_STRING")
		os.Unsetenv("TEST_INT")
		os.Unsetenv("TEST_BOOL")
		os.Unsetenv("TEST_FLOAT")
	}()

	loader := NewEnvLoaderWithOptions("EnvLoader", 10, "env", "")

	tests := []struct {
		name      string
		tagValue  string
		fieldType reflect.Type
		expected  any
		expectOk  bool
		expectErr bool
	}{
		{
			name:      "string value",
			tagValue:  "TEST_STRING",
			fieldType: reflect.TypeFor[string](),
			expected:  "hello",
			expectOk:  true,
			expectErr: false,
		},
		{
			name:      "int value",
			tagValue:  "TEST_INT",
			fieldType: reflect.TypeFor[int](),
			expected:  123,
			expectOk:  true,
			expectErr: false,
		},
		{
			name:      "bool value",
			tagValue:  "TEST_BOOL",
			fieldType: reflect.TypeFor[bool](),
			expected:  true,
			expectOk:  true,
			expectErr: false,
		},
		{
			name:      "float value",
			tagValue:  "TEST_FLOAT",
			fieldType: reflect.TypeFor[float64](),
			expected:  3.14,
			expectOk:  true,
			expectErr: false,
		},
		{
			name:      "empty env variable",
			tagValue:  "NON_EXISTENT",
			fieldType: reflect.TypeFor[string](),
			expected:  nil,
			expectOk:  false,
			expectErr: false,
		},
		{
			name:      "empty tag value",
			tagValue:  "",
			fieldType: reflect.TypeFor[string](),
			expected:  nil,
			expectOk:  false,
			expectErr: false,
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

			if tt.expectErr && err == nil {
				t.Errorf("Expected error, got nil")
				return
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if ok != tt.expectOk {
				t.Errorf("Expected ok = %v, got %v", tt.expectOk, ok)
				return
			}

			if tt.expectOk {
				got := field.Interface()
				if !reflect.DeepEqual(got, tt.expected) {
					t.Errorf("Expected value %v, got %v", tt.expected, got)
				}
			}
		})
	}
}

func TestEnvLoader_LoadValue_WithPrefix(t *testing.T) {
	os.Setenv("APP_HOST", "localhost")
	os.Setenv("APP_PORT", "9090")
	defer func() {
		os.Unsetenv("APP_HOST")
		os.Unsetenv("APP_PORT")
	}()

	loader := NewEnvLoaderWithOptions("EnvLoader", 10, "env", "APP_")

	tests := []struct {
		name      string
		tagValue  string
		fieldType reflect.Type
		expected  any
		expectOk  bool
	}{
		{
			name:      "with prefix string",
			tagValue:  "HOST",
			fieldType: reflect.TypeFor[string](),
			expected:  "localhost",
			expectOk:  true,
		},
		{
			name:      "with prefix int",
			tagValue:  "PORT",
			fieldType: reflect.TypeFor[int](),
			expected:  9090,
			expectOk:  true,
		},
		{
			name:      "without prefix (should not find)",
			tagValue:  "APP_HOST",
			fieldType: reflect.TypeFor[string](),
			expected:  nil,
			expectOk:  false,
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
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if ok != tt.expectOk {
				t.Errorf("Expected ok = %v, got %v", tt.expectOk, ok)
			}
		})
	}
}

func TestEnvLoader_LoadValue_InvalidType(t *testing.T) {
	os.Setenv("TEST_INVALID", "not_a_number")
	defer os.Unsetenv("TEST_INVALID")

	loader := NewEnvLoader()

	field := reflect.New(reflect.TypeFor[int]()).Elem()
	structField := reflect.StructField{
		Name: "TestField",
		Type: reflect.TypeFor[int](),
	}

	ok, err := loader.LoadValue("TEST_INVALID", field, structField)
	if err == nil {
		t.Error("Expected error for invalid int, got nil")
	}
	if ok {
		t.Error("Expected ok = false")
	}
}
