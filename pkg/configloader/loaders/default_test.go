package loaders

import (
	"reflect"
	"testing"
)

func TestDefaultLoader_Name(t *testing.T) {
	loader := NewDefaultLoader()
	if loader.Name() != "DefaultLoader" {
		t.Errorf("Expected name 'DefaultLoader', got '%s'", loader.Name())
	}
}

func TestDefaultLoader_Priority(t *testing.T) {
	loader := NewDefaultLoader()
	if loader.Priority() != 100 {
		t.Errorf("Expected priority 100, got %d", loader.Priority())
	}

	loaderWithOpts := NewDefaultLoaderWithOptions("Custom", 50, "custom")
	if loaderWithOpts.Priority() != 50 {
		t.Errorf("Expected priority 50, got %d", loaderWithOpts.Priority())
	}
}

func TestDefaultLoader_SupportedTag(t *testing.T) {
	loader := NewDefaultLoader()
	if loader.SupportedTag() != "default" {
		t.Errorf("Expected supportedTag 'default', got '%s'", loader.SupportedTag())
	}

	loaderWithOpts := NewDefaultLoaderWithOptions("Custom", 50, "custom")
	if loaderWithOpts.SupportedTag() != "custom" {
		t.Errorf("Expected supportedTag 'custom', got '%s'", loaderWithOpts.SupportedTag())
	}
}

func TestDefaultLoader_LoadValue(t *testing.T) {
	loader := NewDefaultLoader()

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
			tagValue:  "hello",
			fieldType: reflect.TypeFor[string](),
			expected:  "hello",
			expectOk:  true,
			expectErr: false,
		},
		{
			name:      "int value",
			tagValue:  "123",
			fieldType: reflect.TypeFor[int](),
			expected:  123,
			expectOk:  true,
			expectErr: false,
		},
		{
			name:      "int64 value",
			tagValue:  "123456789",
			fieldType: reflect.TypeFor[int64](),
			expected:  int64(123456789),
			expectOk:  true,
			expectErr: false,
		},
		{
			name:      "uint value",
			tagValue:  "42",
			fieldType: reflect.TypeFor[uint](),
			expected:  uint(42),
			expectOk:  true,
			expectErr: false,
		},
		{
			name:      "float value",
			tagValue:  "3.14",
			fieldType: reflect.TypeFor[float64](),
			expected:  3.14,
			expectOk:  true,
			expectErr: false,
		},
		{
			name:      "bool true",
			tagValue:  "true",
			fieldType: reflect.TypeFor[bool](),
			expected:  true,
			expectOk:  true,
			expectErr: false,
		},
		{
			name:      "bool false",
			tagValue:  "false",
			fieldType: reflect.TypeFor[bool](),
			expected:  false,
			expectOk:  true,
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
		{
			name:      "invalid int",
			tagValue:  "not_a_number",
			fieldType: reflect.TypeFor[int](),
			expected:  nil,
			expectOk:  false,
			expectErr: true,
		},
		{
			name:      "invalid bool",
			tagValue:  "not_a_bool",
			fieldType: reflect.TypeFor[bool](),
			expected:  nil,
			expectOk:  false,
			expectErr: true,
		},
		{
			name:      "pointer to string",
			tagValue:  "pointer_value",
			fieldType: reflect.TypeFor[*string](),
			expected:  nil,
			expectOk:  true,
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

			if tt.expectOk && tt.expected != nil {
				got := field.Interface()

				if tt.fieldType.Kind() == reflect.Pointer {
					if field.IsNil() {
						t.Error("Expected pointer to be non-nil, got nil")
						return
					}
					got = field.Elem().Interface()
				}

				if !reflect.DeepEqual(got, tt.expected) {
					t.Errorf("Expected value %v, got %v", tt.expected, got)
				}
			}
		})
	}
}

func TestDefaultLoader_LoadValue_Pointer(t *testing.T) {
	loader := NewDefaultLoader()

	field := reflect.New(reflect.TypeFor[*string]()).Elem()
	structField := reflect.StructField{
		Name: "TestField",
		Type: reflect.TypeFor[*string](),
	}

	ok, err := loader.LoadValue("pointer_value", field, structField)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("Expected ok = true")
	}

	if field.IsNil() {
		t.Fatal("Expected pointer to be non-nil")
	}

	got := field.Elem().String()
	expected := "pointer_value"
	if got != expected {
		t.Errorf("Expected %q, got %q", expected, got)
	}
}

func TestDefaultLoader_LoadValue_UnsupportedType(t *testing.T) {
	loader := NewDefaultLoader()

	field := reflect.New(reflect.TypeFor[chan int]()).Elem()
	structField := reflect.StructField{
		Name: "TestField",
		Type: reflect.TypeFor[chan int](),
	}

	ok, err := loader.LoadValue("test", field, structField)
	if err == nil {
		t.Error("Expected error for unsupported type, got nil")
	}
	if ok {
		t.Error("Expected ok = false")
	}
}
