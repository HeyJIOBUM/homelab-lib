package loaders

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

type YamlLoader struct {
	name         string
	priority     int
	supportedTag string
	filePath     string
	data         map[string]any
}

func NewYamlLoader(filePath string) (*YamlLoader, error) {
	return NewYamlLoaderWithOptions("YamlLoader", 100, "yaml", filePath)
}

func NewYamlLoaderWithOptions(name string, priority int, supportedTag, filePath string) (*YamlLoader, error) {
	loader := &YamlLoader{
		name:         name,
		priority:     priority,
		supportedTag: supportedTag,
		filePath:     filePath,
	}

	if err := loader.loadYAML(); err != nil {
		return nil, err
	}

	return loader, nil
}

func (yl *YamlLoader) Name() string         { return yl.name }
func (yl *YamlLoader) Priority() int        { return yl.priority }
func (yl *YamlLoader) SupportedTag() string { return yl.supportedTag }

func (yl *YamlLoader) LoadValue(tagValue string, field reflect.Value, structField reflect.StructField) (bool, error) {
	if tagValue == "" {
		return false, nil
	}

	value, ok := yl.findValue(tagValue)
	if !ok {
		return false, nil
	}

	if err := setFieldValueFromInterface(field, value); err != nil {
		return false, fmt.Errorf("yaml loader: field %s: %w", structField.Name, err)
	}

	return true, nil
}

func (yl *YamlLoader) loadYAML() error {
	if yl.filePath == "" {
		return fmt.Errorf("file path is not set")
	}

	data, err := os.ReadFile(yl.filePath)
	if err != nil {
		return fmt.Errorf("read yaml file: %w", err)
	}

	var yamlData map[string]any
	if err := yaml.Unmarshal(data, &yamlData); err != nil {
		return fmt.Errorf("unmarshal yaml: %w", err)
	}

	yl.data = yamlData
	return nil
}

func (yl *YamlLoader) findValue(path string) (any, bool) {
	parts := strings.Split(path, ".")
	current := yl.data

	for i, part := range parts {
		if i == len(parts)-1 {
			val, ok := current[part]
			return val, ok
		}

		if next, ok := current[part].(map[string]any); ok {
			current = next
		} else {
			return nil, false
		}
	}

	return nil, false
}
