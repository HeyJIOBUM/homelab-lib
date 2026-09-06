package configloader

import (
	"reflect"
	"slices"
	"strconv"
	"strings"
)

type TagsResolver struct {
	SupportedTags []string
}

func NewTagsResolver(supportedTags []string) *TagsResolver {
	return &TagsResolver{
		SupportedTags: supportedTags,
	}
}

type TagInfo struct {
	Name  string
	Value string
}

func (tr *TagsResolver) ExtractTags(field reflect.StructField) []TagInfo {
	tag := strings.TrimSpace(string(field.Tag))
	if tag == "" {
		return []TagInfo{}
	}

	var result []TagInfo
	for tag != "" {
		i := 0
		for i < len(tag) && tag[i] == ' ' {
			i++
		}
		tag = tag[i:]
		if tag == "" {
			break
		}

		i = 0
		for i < len(tag) && tag[i] > ' ' && tag[i] != ':' && tag[i] != '"' {
			i++
		}

		if i == 0 || i+1 >= len(tag) || tag[i] != ':' || tag[i+1] != '"' {
			break
		}
		name := string(tag[:i])
		tag = tag[i+1:]

		i = 1
		for i < len(tag) && tag[i] != '"' {
			if tag[i] == '\\' {
				i++
			}
			i++
		}
		if i >= len(tag) {
			break
		}
		qvalue := string(tag[:i+1])
		tag = tag[i+1:]

		if slices.Contains(tr.SupportedTags, name) {
			value, err := strconv.Unquote(qvalue)
			if err != nil {
				break
			}
			result = append(result, TagInfo{Name: name, Value: value})
		}
	}

	return result
}
