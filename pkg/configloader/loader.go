package configloader

import (
	"fmt"
	"reflect"
	"slices"
	"sort"
)

type LoaderInterface interface {
	Name() string
	Priority() int
	SupportedTag() string
	LoadValue(tagValue string, field reflect.Value, structField reflect.StructField) (bool, error)
}

type Loader struct {
	strictMode         bool
	tagsResolver       *TagsResolver
	prioritizedLoaders []map[string]LoaderInterface
}

func NewLoader(loaders []LoaderInterface, strictMode bool) *Loader {
	usedTags := make([]string, 0, len(loaders))
	for _, loader := range loaders {
		if tag := loader.SupportedTag(); tag != "" && tag != "-" {
			usedTags = append(usedTags, tag)
		}
	}
	tagsResolver := NewTagsResolver(usedTags)

	prioritizedLoaders := groupByPriority(loaders)

	return &Loader{
		strictMode:         strictMode,
		tagsResolver:       tagsResolver,
		prioritizedLoaders: prioritizedLoaders,
	}
}

func (l *Loader) Load(v any) error {
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Pointer {
		return ErrInvalidStruct
	}

	structVal := val.Elem()
	if structVal.Kind() != reflect.Struct {
		return ErrInvalidStruct
	}

	if err := l.loadStruct(structVal, ""); err != nil {
		return err
	}

	return nil
}

func groupByPriority(loaders []LoaderInterface) []map[string]LoaderInterface {
	if len(loaders) == 0 {
		return []map[string]LoaderInterface{}
	}

	sortedLoaders := make([]LoaderInterface, len(loaders))
	copy(sortedLoaders, loaders)
	sort.Slice(sortedLoaders, func(i, j int) bool {
		return sortedLoaders[i].Priority() < sortedLoaders[j].Priority()
	})

	prioritizedLoaders := make([]map[string]LoaderInterface, 0)
	var currentGroup map[string]LoaderInterface
	var currentPriority int

	for i, loader := range sortedLoaders {
		if i == 0 || loader.Priority() != currentPriority {
			currentGroup = make(map[string]LoaderInterface)
			currentPriority = loader.Priority()
			prioritizedLoaders = append(prioritizedLoaders, currentGroup)
		}
		currentGroup[loader.SupportedTag()] = loader
	}

	return prioritizedLoaders
}

func (l *Loader) loadStruct(val reflect.Value, prefix string) error {
	typ := val.Type()

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		if !fieldVal.CanSet() {
			continue
		}

		if fieldVal.Kind() == reflect.Struct {
			if err := l.loadStruct(fieldVal, prefix+field.Name+"."); err != nil {
				return fmt.Errorf("field %s: %w", field.Name, err)
			}
			continue
		}

		ok, err := l.processField(fieldVal, field)
		if err != nil {
			return fmt.Errorf("field %s: %w", field.Name, err)
		}

		if !ok && l.strictMode {
			return fmt.Errorf("field %s is not set while strict mode is enabled", field.Name)
		}
	}

	return nil
}

func (l *Loader) processField(field reflect.Value, structField reflect.StructField) (bool, error) {
	tags := l.tagsResolver.ExtractTags(structField)

	for _, priorityGroup := range l.prioritizedLoaders {
		tagsInPriorityGroup := make([]string, 0, len(priorityGroup))
		for k := range priorityGroup {
			tagsInPriorityGroup = append(tagsInPriorityGroup, k)
		}

		for _, tag := range tags {
			if slices.Contains(tagsInPriorityGroup, tag.Name) {
				ok, err := priorityGroup[tag.Name].LoadValue(tag.Value, field, structField)
				if err != nil {
					return ok, err
				}
				if ok {
					return true, nil
				}
			}
		}
	}

	return false, nil
}
