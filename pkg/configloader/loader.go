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
	sortedLoaders := make([]LoaderInterface, len(loaders))
	copy(sortedLoaders, loaders)
	sort.Slice(sortedLoaders, func(i, j int) bool {
		return sortedLoaders[i].Priority() > sortedLoaders[j].Priority()
	})

	firstPriorityGroup := make(map[string]LoaderInterface)
	firstPriorityGroup[sortedLoaders[0].SupportedTag()] = sortedLoaders[0]
	lastSavedPriority := sortedLoaders[0].Priority()

	prioritizedLoaders := make([]map[string]LoaderInterface, 0)
	prioritizedLoaders = append(prioritizedLoaders, firstPriorityGroup)

	for i := 1; i < len(sortedLoaders); i++ {
		curLoader := sortedLoaders[i]
		if lastSavedPriority == curLoader.Priority() {
			prioritizedLoaders[len(prioritizedLoaders)-1][curLoader.SupportedTag()] = curLoader
		} else {
			lastSavedPriority = curLoader.Priority()
			newPriorityMap := make(map[string]LoaderInterface)
			newPriorityMap[curLoader.SupportedTag()] = curLoader
			prioritizedLoaders = append(prioritizedLoaders, newPriorityMap)
		}
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

		if _, err := l.processField(fieldVal, field); err != nil {
			return fmt.Errorf("field %s: %w", field.Name, err)
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
