package loaders

import (
	"fmt"
	"reflect"
)

type DefaultLoader struct {
	name         string
	priority     int
	supportedTag string
}

func NewDefaultLoader() *DefaultLoader {
	return NewDefaultLoaderWithOptions("DefaultLoader", 100, "default")
}

func NewDefaultLoaderWithOptions(name string, priority int, supportedTag string) *DefaultLoader {
	return &DefaultLoader{
		name:         name,
		priority:     priority,
		supportedTag: supportedTag,
	}
}

func (dl *DefaultLoader) Name() string         { return dl.name }
func (dl *DefaultLoader) Priority() int        { return dl.priority }
func (dl *DefaultLoader) SupportedTag() string { return dl.supportedTag }

func (dl *DefaultLoader) LoadValue(tagValue string, field reflect.Value, structField reflect.StructField) (bool, error) {
	if tagValue == "" {
		return false, nil
	}
	if err := setFieldValue(field, tagValue); err != nil {
		return false, fmt.Errorf("default loader: field %s: %w", structField.Name, err)
	}
	return true, nil
}
