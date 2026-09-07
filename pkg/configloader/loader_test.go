package configloader

import (
	"errors"
	"reflect"
	"testing"
)

type mockLoader struct {
	name         string
	priority     int
	supportedTag string
	loadValueFn  func(tagValue string, field reflect.Value, structField reflect.StructField) (bool, error)
}

func (m *mockLoader) Name() string {
	return m.name
}

func (m *mockLoader) Priority() int {
	return m.priority
}

func (m *mockLoader) SupportedTag() string {
	return m.supportedTag
}

func (m *mockLoader) LoadValue(tagValue string, field reflect.Value, structField reflect.StructField) (bool, error) {
	if m.loadValueFn != nil {
		return m.loadValueFn(tagValue, field, structField)
	}
	return false, nil
}

func newMockLoader(name string, priority int, supportedTag string) *mockLoader {
	return &mockLoader{
		name:         name,
		priority:     priority,
		supportedTag: supportedTag,
	}
}

func TestNewLoader(t *testing.T) {
	loaders := []LoaderInterface{
		newMockLoader("loader1", 10, "tag1"),
		newMockLoader("loader2", 5, "tag2"),
		newMockLoader("loader3", 10, "tag3"),
	}

	loader := NewLoader(loaders, true)

	if loader == nil {
		t.Fatal("NewLoader returned nil")
	}

	if len(loader.prioritizedLoaders) != 2 {
		t.Errorf("Expected 2 priority groups, got %d", len(loader.prioritizedLoaders))
	}

	if len(loader.prioritizedLoaders[0]) != 1 {
		t.Errorf("Expected 1 loader in group 0, got %d", len(loader.prioritizedLoaders[0]))
	}

	if _, ok := loader.prioritizedLoaders[0]["tag2"]; !ok {
		t.Error("Expected tag2 in first priority group")
	}

	if len(loader.prioritizedLoaders[1]) != 2 {
		t.Errorf("Expected 2 loaders in group 1, got %d", len(loader.prioritizedLoaders[1]))
	}
}

func TestLoader_Load_InvalidStruct(t *testing.T) {
	loader := NewLoader([]LoaderInterface{}, false)

	tests := []struct {
		name    string
		v       any
		wantErr bool
	}{
		{
			name:    "nil value",
			v:       nil,
			wantErr: true,
		},
		{
			name:    "non-pointer",
			v:       struct{}{},
			wantErr: true,
		},
		{
			name:    "pointer to non-struct",
			v:       new(string),
			wantErr: true,
		},
		{
			name:    "pointer to struct",
			v:       &struct{}{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := loader.Load(tt.v)
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoader_Load_WithLoaders(t *testing.T) {
	type TestConfig struct {
		Host string `tag1:"host"`
		Port int    `tag2:"port"`
	}

	loader1 := newMockLoader("loader1", 10, "tag1")
	loader1.loadValueFn = func(tagValue string, field reflect.Value, structField reflect.StructField) (bool, error) {
		if tagValue == "host" {
			field.SetString("localhost")
			return true, nil
		}
		return false, nil
	}

	loader2 := newMockLoader("loader2", 20, "tag2")
	loader2.loadValueFn = func(tagValue string, field reflect.Value, structField reflect.StructField) (bool, error) {
		if tagValue == "port" {
			field.SetInt(8080)
			return true, nil
		}
		return false, nil
	}

	loaders := []LoaderInterface{loader1, loader2}
	loader := NewLoader(loaders, true)

	var cfg TestConfig
	err := loader.Load(&cfg)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Host != "localhost" {
		t.Errorf("Expected Host = 'localhost', got '%s'", cfg.Host)
	}
	if cfg.Port != 8080 {
		t.Errorf("Expected Port = 8080, got %d", cfg.Port)
	}
}

func TestLoader_Load_LoaderError(t *testing.T) {
	type TestConfig struct {
		Host string `tag1:"host"`
	}

	mockLoader := newMockLoader("loader1", 10, "tag1")
	mockLoader.loadValueFn = func(tagValue string, field reflect.Value, structField reflect.StructField) (bool, error) {
		return false, errors.New("loader error")
	}

	loader := NewLoader([]LoaderInterface{mockLoader}, true)

	var cfg TestConfig
	err := loader.Load(&cfg)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestLoader_Load_StrictMode(t *testing.T) {
	type TestConfig struct {
		Host string `tag1:"host"`
	}

	mockLoader := newMockLoader("loader1", 10, "tag1")
	mockLoader.loadValueFn = func(tagValue string, field reflect.Value, structField reflect.StructField) (bool, error) {
		return false, nil // Не загружаем значение
	}

	tests := []struct {
		name       string
		strictMode bool
		wantErr    bool
	}{
		{
			name:       "strict mode enabled",
			strictMode: true,
			wantErr:    true,
		},
		{
			name:       "strict mode disabled",
			strictMode: false,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := NewLoader([]LoaderInterface{mockLoader}, tt.strictMode)
			var cfg TestConfig
			err := loader.Load(&cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoader_Load_NestedStruct(t *testing.T) {
	type InnerConfig struct {
		Host string `tag1:"host"`
	}
	type TestConfig struct {
		Inner InnerConfig
	}

	mockLoader := newMockLoader("loader1", 10, "tag1")
	mockLoader.loadValueFn = func(tagValue string, field reflect.Value, structField reflect.StructField) (bool, error) {
		if tagValue == "host" {
			field.SetString("localhost")
			return true, nil
		}
		return false, nil
	}

	loader := NewLoader([]LoaderInterface{mockLoader}, true)

	var cfg TestConfig
	err := loader.Load(&cfg)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Inner.Host != "localhost" {
		t.Errorf("Expected Inner.Host = 'localhost', got '%s'", cfg.Inner.Host)
	}
}

func TestLoader_Load_FieldNotSet(t *testing.T) {
	type TestConfig struct {
		Host string // Без тегов
	}

	loader := NewLoader([]LoaderInterface{}, true)

	var cfg TestConfig
	err := loader.Load(&cfg)
	if err == nil {
		t.Error("Expected error for unset field in strict mode, got nil")
	}
}

func TestGroupByPriority(t *testing.T) {
	loaders := []LoaderInterface{
		newMockLoader("loader1", 10, "tag1"),
		newMockLoader("loader2", 5, "tag2"),
		newMockLoader("loader3", 10, "tag3"),
		newMockLoader("loader4", 1, "tag4"),
	}

	groups := groupByPriority(loaders)

	if len(groups) != 3 {
		t.Errorf("Expected 3 groups, got %d", len(groups))
	}

	expectedTags := []string{"tag4", "tag2", "tag1", "tag3"}
	var actualTags []string
	for _, group := range groups {
		for tag := range group {
			actualTags = append(actualTags, tag)
		}
	}

	if len(actualTags) != len(expectedTags) {
		t.Errorf("Expected %d tags, got %d", len(expectedTags), len(actualTags))
	}

	if _, ok := groups[0]["tag4"]; !ok {
		t.Error("Expected tag4 in first group (highest priority)")
	}
}

func TestLoader_Load_PriorityOrder(t *testing.T) {
	type TestConfig struct {
		Value string `tag1:"value" tag2:"value"`
	}

	// tag1 имеет более высокий приоритет (5)
	loader1 := newMockLoader("loader1", 5, "tag1")
	loader1.loadValueFn = func(tagValue string, field reflect.Value, structField reflect.StructField) (bool, error) {
		field.SetString("from tag1")
		return true, nil
	}

	loader2 := newMockLoader("loader2", 10, "tag2")
	loader2.loadValueFn = func(tagValue string, field reflect.Value, structField reflect.StructField) (bool, error) {
		field.SetString("from tag2")
		return true, nil
	}

	loader := NewLoader([]LoaderInterface{loader1, loader2}, true)

	var cfg TestConfig
	err := loader.Load(&cfg)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Value != "from tag1" {
		t.Errorf("Expected 'from tag1', got '%s'", cfg.Value)
	}
}
