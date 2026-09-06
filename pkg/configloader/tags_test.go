package configloader

import (
	"reflect"
	"testing"
)

func TestTagResolver_ExtractTags(t *testing.T) {
	tests := []struct {
		name          string
		tag           string
		supportedTags []string
		expected      []TagInfo
		expectedLen   int
	}{
		{
			name:          "all supported tags in order",
			tag:           `env:"DB_PASSWORD" yaml:"password" vault:"db#password"`,
			supportedTags: []string{"env", "yaml", "vault"},
			expected: []TagInfo{
				{Name: "env", Value: "DB_PASSWORD"},
				{Name: "yaml", Value: "password"},
				{Name: "vault", Value: "db#password"},
			},
		},
		{
			name:          "only env and vault (yaml filtered out)",
			tag:           `env:"DB_PASSWORD" yaml:"password" vault:"db#password"`,
			supportedTags: []string{"env", "vault"},
			expected: []TagInfo{
				{Name: "env", Value: "DB_PASSWORD"},
				{Name: "vault", Value: "db#password"},
			},
		},
		{
			name:          "different order",
			tag:           `vault:"db#password" env:"DB_PASSWORD" yaml:"password"`,
			supportedTags: []string{"env", "yaml", "vault"},
			expected: []TagInfo{
				{Name: "vault", Value: "db#password"},
				{Name: "env", Value: "DB_PASSWORD"},
				{Name: "yaml", Value: "password"},
			},
		},
		{
			name:          "escaped quotes in value",
			tag:           `env:"DB_\"PASSWORD\"" yaml:"password"`,
			supportedTags: []string{"env", "yaml"},
			expected: []TagInfo{
				{Name: "env", Value: `DB_"PASSWORD"`},
				{Name: "yaml", Value: "password"},
			},
		},
		{
			name:          "empty tag",
			tag:           "",
			supportedTags: []string{"env", "yaml"},
			expected:      []TagInfo{},
		},
		{
			name:          "no supported tags",
			tag:           `env:"DB_PASSWORD" yaml:"password"`,
			supportedTags: []string{"vault"},
			expected:      []TagInfo{},
		},
		{
			name:          "tag with only env",
			tag:           `env:"DB_PASSWORD"`,
			supportedTags: []string{"env"},
			expected: []TagInfo{
				{Name: "env", Value: "DB_PASSWORD"},
			},
		},
		{
			name:          "tag with spaces",
			tag:           ` env:"DB_PASSWORD"  yaml:"password" `,
			supportedTags: []string{"env", "yaml"},
			expected: []TagInfo{
				{Name: "env", Value: "DB_PASSWORD"},
				{Name: "yaml", Value: "password"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			type TestStruct struct {
				Field string `json:"field"`
			}

			field := reflect.StructField{
				Name: "Field",
				Type: reflect.TypeOf(""),
				Tag:  reflect.StructTag(tt.tag),
			}

			resolver := NewTagsResolver(tt.supportedTags)
			result := resolver.ExtractTags(field)

			if len(result) != len(tt.expected) {
				t.Errorf("ExtractTags() returned %d tags, expected %d", len(result), len(tt.expected))
				return
			}

			for i, tag := range result {
				if tag.Name != tt.expected[i].Name {
					t.Errorf("Tag %d: expected name %q, got %q", i, tt.expected[i].Name, tag.Name)
				}
				if tag.Value != tt.expected[i].Value {
					t.Errorf("Tag %d: expected value %q, got %q", i, tt.expected[i].Value, tag.Value)
				}
			}
		})
	}
}
