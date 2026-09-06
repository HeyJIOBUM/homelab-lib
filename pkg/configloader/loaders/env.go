package loaders

import (
	"fmt"
	"os"
	"reflect"
)

type EnvLoader struct {
	name         string
	priority     int
	supportedTag string
	prefix       string
}

func NewEnvLoader() *EnvLoader {
	return NewEnvLoaderWithOptions("EnvLoader", 0, "env", "")
}

func NewEnvLoaderWithOptions(name string, priority int, supportedTag, prefix string) *EnvLoader {
	return &EnvLoader{
		name:         name,
		priority:     priority,
		supportedTag: supportedTag,
		prefix:       prefix,
	}
}

func (el *EnvLoader) Name() string         { return el.name }
func (el *EnvLoader) Priority() int        { return el.priority }
func (el *EnvLoader) SupportedTag() string { return el.supportedTag }

func (el *EnvLoader) LoadValue(tagValue string, field reflect.Value, structField reflect.StructField) (bool, error) {
	if tagValue == "" {
		return false, nil
	}

	key := el.prefix + tagValue
	value := os.Getenv(key)
	if value == "" {
		return false, nil
	}

	if err := setFieldValue(field, tagValue); err != nil {
		return false, fmt.Errorf("Env loader: field %s: %w", structField.Name, err)
	}

	return true, nil
}
