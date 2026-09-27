package schema

import (
	"fmt"
	"slices"
	"strings"
)

type fieldIssue struct {
	Field string
	Code  string
}

func bindDeclared(fields []Field, values map[string]any) []fieldIssue {
	var issues []fieldIssue
	for _, field := range fields {
		value, exists := values[string(field.ID)]
		if !exists || value == nil || value == "" {
			if field.Required {
				issues = append(issues, fieldIssue{Field: string(field.ID), Code: "required"})
			}
			continue
		}
		if issue := typeIssue(field, value); issue != nil {
			issues = append(issues, *issue)
		}
	}
	return issues
}

func typeIssue(field Field, value any) *fieldIssue {
	switch field.Type {
	case FieldNumber:
		switch value.(type) {
		case float64, float32, int, int32, int64:
			return nil
		default:
			return &fieldIssue{Field: string(field.ID), Code: "type"}
		}
	case FieldBoolean:
		if _, ok := value.(bool); !ok {
			return &fieldIssue{Field: string(field.ID), Code: "type"}
		}
	case FieldSelect:
		text, ok := value.(string)
		if !ok {
			return &fieldIssue{Field: string(field.ID), Code: "type"}
		}
		if len(field.Options) == 0 {
			return nil
		}
		for _, option := range field.Options {
			if option.Value == text {
				return nil
			}
		}
		return &fieldIssue{Field: string(field.ID), Code: "option"}
	case FieldText, FieldString, FieldTextarea, FieldRichText, FieldMarkdown, FieldDateTime:
		if _, ok := value.(string); !ok {
			return &fieldIssue{Field: string(field.ID), Code: "type"}
		}
	case FieldObject:
		document, ok := value.(map[string]any)
		if !ok {
			return &fieldIssue{Field: string(field.ID), Code: "type"}
		}
		for _, nested := range field.Fields {
			item, exists := document[string(nested.ID)]
			if !exists || item == nil || item == "" {
				if nested.Required {
					return &fieldIssue{Field: string(nested.ID), Code: "required"}
				}
				continue
			}
			if issue := typeIssue(nested, item); issue != nil {
				return issue
			}
		}
	case FieldCollection:
		if field.Items == nil {
			return nil
		}
		switch items := value.(type) {
		case []any:
			for _, item := range items {
				if issue := typeIssue(*field.Items, item); issue != nil {
					return issue
				}
			}
		case []string:
			for _, item := range items {
				if issue := typeIssue(*field.Items, item); issue != nil {
					return issue
				}
			}
		default:
			return &fieldIssue{Field: string(field.ID), Code: "type"}
		}
	}
	return nil
}

func formatFieldIssues(issues []fieldIssue) string {
	parts := make([]string, len(issues))
	for index, issue := range issues {
		parts[index] = fmt.Sprintf("%s:%s", issue.Field, issue.Code)
	}
	slices.Sort(parts)
	return strings.Join(parts, ",")
}
