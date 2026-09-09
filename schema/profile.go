package schema

import (
	"strings"

	"github.com/fastygo/codex/validation"
	"github.com/fastygo/formset"
)

// Codex validation-rule names preserve content semantics that are more
// specific than FormSet's renderer-oriented field types.
const (
	RuleInteger  = "fastygo.codex/integer"
	RuleDecimal  = "fastygo.codex/decimal"
	RuleMoney    = "fastygo.codex/money"
	RuleDate     = "fastygo.codex/date"
	RuleURI      = "fastygo.codex/uri"
	RuleUUID     = "fastygo.codex/uuid"
	RuleNullable = "fastygo.codex/nullable"
	RuleReadOnly = "fastygo.codex/read-only"
	RuleJSONAny  = "fastygo.codex/json-any"
)

const UIHintMedia = "media"

func FieldRule(field formset.Field, name string) (formset.ValidationRule, bool) {
	for _, rule := range field.Rules {
		if rule.Name == name {
			return rule, true
		}
	}
	return formset.ValidationRule{}, false
}

// ValidateFieldProfile verifies Codex semantics layered on a FormSet field.
func ValidateFieldProfile(field formset.Field) error {
	return validateFieldProfile(field, false)
}

func validateFieldProfile(field formset.Field, nested bool) error {
	if nested && field.Localized {
		return validation.New("schema.field.nested_localized", string(field.ID), "nested field cannot be localized")
	}
	if nested && field.Type == formset.FieldRelation {
		return validation.New("schema.field.nested_relation", string(field.ID), "relation field must be top-level")
	}
	seen := make(map[string]struct{}, len(field.Rules))
	for _, rule := range field.Rules {
		if strings.TrimSpace(rule.Name) == "" {
			return validation.New("schema.field.rule_name_required", string(field.ID), "field rule name is required")
		}
		if _, exists := seen[rule.Name]; exists {
			return validation.New("schema.field.rule_duplicated", string(field.ID), "field rule is duplicated")
		}
		seen[rule.Name] = struct{}{}
		switch rule.Name {
		case RuleInteger, RuleDecimal:
			if field.Type != formset.FieldNumber {
				return validation.New("schema.field.rule_type", string(field.ID), "numeric Codex rule requires number field")
			}
		case RuleMoney:
			if field.Type != formset.FieldNumber {
				return validation.New("schema.field.rule_type", string(field.ID), "money Codex rule requires number field")
			}
		case RuleDate:
			if field.Type != formset.FieldDateTime {
				return validation.New("schema.field.rule_type", string(field.ID), "date Codex rule requires datetime field")
			}
		case RuleURI, RuleUUID:
			if field.Type != formset.FieldString {
				return validation.New("schema.field.rule_type", string(field.ID), "string Codex rule requires string field")
			}
		case RuleJSONAny:
			if field.Type != formset.FieldJSON {
				return validation.New("schema.field.rule_type", string(field.ID), "JSON-any Codex rule requires JSON field")
			}
		case RuleNullable, RuleReadOnly:
			// These rules add policy without changing the FormSet renderer type.
		}
	}
	if _, nullable := FieldRule(field, RuleNullable); nullable && field.Required {
		return validation.New("schema.field.required_nullable", string(field.ID), "required field cannot be nullable")
	}
	for _, nested := range field.Fields {
		if err := validateFieldProfile(nested, true); err != nil {
			return err
		}
	}
	if field.Items != nil {
		if err := validateFieldProfile(*field.Items, true); err != nil {
			return err
		}
	}
	return nil
}
