package schema

import (
	"encoding/json"
	"errors"
	"math"
	"net/url"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/fastygo/codex/content"
	"github.com/fastygo/codex/validation"
	"github.com/fastygo/formset"
)

func (resource Resource) ValidateEntry(entry content.Entry) error {
	if err := resource.Validate(); err != nil {
		return err
	}
	if content.Kind(resource.Record.ID) != entry.Kind {
		return validation.New("schema.entry.kind_mismatch", "kind", "entry kind does not match resource")
	}
	if err := entry.Validate(); err != nil {
		return err
	}

	sharedFields, localizedFields := splitFields(resource.Record.Fields)
	for _, field := range localizedFields {
		if _, exists := entry.Metadata[string(field.ID)]; exists {
			return validation.New(
				"schema.entry.field_location",
				string(field.ID),
				"localized field must be stored in locale data",
			)
		}
	}
	for locale, document := range entry.Locales {
		for _, field := range sharedFields {
			if _, exists := document.Data[string(field.ID)]; exists {
				return validation.New(
					"schema.entry.field_location",
					locale+"."+string(field.ID),
					"non-localized field must be stored in metadata",
				)
			}
		}
	}
	shared := make(map[string]any, len(entry.Metadata))
	for key, value := range entry.Metadata {
		shared[key] = value.Value
	}
	if err := bindFields(resource.Record, sharedFields, "und", shared); err != nil {
		return err
	}
	for _, field := range sharedFields {
		value, exists := shared[string(field.ID)]
		if exists {
			if err := validateCodexRules(field, value); err != nil {
				return validation.Wrap("schema.entry.field_invalid", string(field.ID), err)
			}
		}
	}

	for _, locale := range entryLocales(entry) {
		document := map[string]any{}
		if stored, exists := entry.Locales[locale]; exists {
			for key, value := range stored.Data {
				document[key] = value
			}
		}
		for _, field := range localizedFields {
			value, exists := entryLocalizedValue(entry, locale, string(field.ID))
			if !exists {
				continue
			}
			if stored, already := document[string(field.ID)]; already && !reflect.DeepEqual(stored, value) {
				return validation.New(
					"schema.entry.localized_conflict",
					locale+"."+string(field.ID),
					"field differs between entry chrome and locale document",
				)
			}
			document[string(field.ID)] = value
		}
		if err := bindFields(resource.Record, localizedFields, locale, document); err != nil {
			return err
		}
		for _, field := range localizedFields {
			value, exists := document[string(field.ID)]
			if exists {
				if err := validateCodexRules(field, value); err != nil {
					return validation.Wrap("schema.entry.field_invalid", locale+"."+string(field.ID), err)
				}
			}
		}
	}

	return validateRelationValues(resource, entry.Metadata)
}

func (resource Resource) PublicProjection(entry content.Entry) (content.Entry, error) {
	if err := resource.ValidateEntry(entry); err != nil {
		return content.Entry{}, err
	}
	projected := entry.PublicProjection()
	for _, field := range resource.Record.Fields {
		if field.Localized {
			for locale, document := range projected.Locales {
				if value, exists := document.Data[string(field.ID)]; exists {
					value, keep := projectSensitive(field, value)
					if keep {
						document.Data[string(field.ID)] = value
					} else {
						delete(document.Data, string(field.ID))
					}
				}
				projected.Locales[locale] = document
			}
			if field.Sensitive {
				clearEntryChromeField(&projected, string(field.ID))
			}
			continue
		}
		value, exists := projected.Metadata[string(field.ID)]
		if !exists {
			continue
		}
		projectedValue, keep := projectSensitive(field, value.Value)
		if !keep {
			delete(projected.Metadata, string(field.ID))
			continue
		}
		value.Value = projectedValue
		projected.Metadata[string(field.ID)] = value
	}
	return projected, nil
}

func projectSensitive(field formset.Field, value any) (any, bool) {
	if field.Sensitive {
		return nil, false
	}
	if field.Type == formset.FieldObject {
		document, ok := value.(map[string]any)
		if !ok {
			return value, true
		}
		for _, nested := range field.Fields {
			nestedValue, exists := document[string(nested.ID)]
			if !exists {
				continue
			}
			projected, keep := projectSensitive(nested, nestedValue)
			if keep {
				document[string(nested.ID)] = projected
			} else {
				delete(document, string(nested.ID))
			}
		}
	}
	if field.Type == formset.FieldCollection && field.Items != nil {
		if field.Items.Sensitive {
			reflected := reflect.ValueOf(value)
			if reflected.IsValid() && reflected.Kind() == reflect.Slice {
				return reflect.MakeSlice(reflected.Type(), 0, 0).Interface(), true
			}
			return []any{}, true
		}
		values, ok := value.([]any)
		if !ok {
			return value, true
		}
		for index, item := range values {
			projected, keep := projectSensitive(*field.Items, item)
			if keep {
				values[index] = projected
			}
		}
	}
	return value, true
}

func splitFields(fields []formset.Field) (shared, localized []formset.Field) {
	for _, field := range fields {
		if field.Localized {
			localized = append(localized, field)
		} else {
			shared = append(shared, field)
		}
	}
	return shared, localized
}

func bindFields(record formset.RecordType, fields []formset.Field, locale string, values map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	projected := record
	projected.Fields = fields
	projected.Relations = nil
	formValues := make(map[string]any, len(values))
	for key, value := range values {
		formValues[key] = valueForFormSet(value)
	}
	form, err := formset.BindLocale(projected, locale, formValues)
	if err != nil {
		return err
	}
	if len(form.Issues) == 0 {
		return nil
	}
	parts := make([]string, 0, len(form.Issues))
	for _, issue := range form.Issues {
		parts = append(parts, issue.Field+":"+issue.Code)
	}
	slices.Sort(parts)
	return validation.New(
		"schema.entry.formset_invalid",
		locale,
		"entry field validation failed: "+strings.Join(parts, ","),
	)
}

func valueForFormSet(value any) any {
	switch typed := value.(type) {
	case json.Number:
		if integer, err := typed.Int64(); err == nil {
			return integer
		}
		if decimal, err := typed.Float64(); err == nil {
			return decimal
		}
		return value
	case map[string]any:
		cloned := make(map[string]any, len(typed))
		for key, nested := range typed {
			cloned[key] = valueForFormSet(nested)
		}
		return cloned
	case []any:
		cloned := make([]any, len(typed))
		for index, nested := range typed {
			cloned[index] = valueForFormSet(nested)
		}
		return cloned
	default:
		return value
	}
}

func entryLocales(entry content.Entry) []string {
	set := make(map[string]struct{}, len(entry.Locales))
	for locale := range entry.Locales {
		set[locale] = struct{}{}
	}
	for _, values := range []content.LocalizedText{entry.Slug, entry.Title, entry.Content, entry.Excerpt} {
		for locale := range values {
			set[locale] = struct{}{}
		}
	}
	locales := make([]string, 0, len(set))
	for locale := range set {
		locales = append(locales, locale)
	}
	slices.Sort(locales)
	return locales
}

func entryLocalizedValue(entry content.Entry, locale, fieldID string) (any, bool) {
	var values content.LocalizedText
	switch fieldID {
	case "slug":
		values = entry.Slug
	case "title":
		values = entry.Title
	case "content":
		values = entry.Content
	case "excerpt":
		values = entry.Excerpt
	default:
		return nil, false
	}
	value, exists := values[locale]
	return value, exists
}

func clearEntryChromeField(entry *content.Entry, fieldID string) {
	switch fieldID {
	case "slug":
		entry.Slug = nil
	case "title":
		entry.Title = nil
	case "content":
		entry.Content = nil
	case "excerpt":
		entry.Excerpt = nil
	}
}

func validateRelationValues(resource Resource, metadata map[string]content.MetadataValue) error {
	for _, relation := range resource.Record.Relations {
		value, exists := metadata[string(relation.ID)]
		if !exists || value.Value == nil {
			continue
		}
		switch relation.Cardinality {
		case formset.RelationOneToOne:
			if !nonEmptyID(value.Value) {
				return validation.New("schema.entry.relation_scalar", string(relation.ID), "relation requires one id")
			}
		case formset.RelationOneToMany, formset.RelationManyToMany:
			if !idCollection(value.Value) {
				return validation.New("schema.entry.relation_array", string(relation.ID), "relation requires an id array")
			}
		default:
			return validation.New("schema.entry.relation_cardinality", string(relation.ID), "relation has unsupported cardinality")
		}
	}
	return nil
}

func nonEmptyID(value any) bool {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed) != ""
	case content.ID:
		return strings.TrimSpace(string(typed)) != ""
	default:
		return false
	}
}

func idCollection(value any) bool {
	reflected := reflect.ValueOf(value)
	if !reflected.IsValid() || (reflected.Kind() != reflect.Slice && reflected.Kind() != reflect.Array) {
		return false
	}
	for index := 0; index < reflected.Len(); index++ {
		if !nonEmptyID(reflected.Index(index).Interface()) {
			return false
		}
	}
	return true
}

func validateCodexRules(field formset.Field, value any) error {
	if value == nil {
		if _, nullable := FieldRule(field, RuleNullable); nullable {
			return nil
		}
		return errors.New("null requires nullable rule")
	}
	for _, rule := range field.Rules {
		var err error
		switch rule.Name {
		case RuleInteger:
			err = validateInteger(value)
		case RuleDecimal:
			err = validateDecimal(value)
		case RuleMoney:
			err = validateMoney(value)
		case RuleDate:
			err = validateDate(value)
		case RuleURI:
			err = validateURI(value)
		case RuleUUID:
			err = validateUUID(value)
		case RuleJSONAny:
			err = content.ValidateJSONValue(value)
		}
		if err != nil {
			return err
		}
	}
	for _, nested := range field.Fields {
		document, ok := value.(map[string]any)
		if !ok {
			continue
		}
		if nestedValue, exists := document[string(nested.ID)]; exists {
			if err := validateCodexRules(nested, nestedValue); err != nil {
				return err
			}
		}
	}
	if field.Items != nil {
		reflected := reflect.ValueOf(value)
		if reflected.IsValid() && (reflected.Kind() == reflect.Slice || reflected.Kind() == reflect.Array) {
			for index := 0; index < reflected.Len(); index++ {
				if err := validateCodexRules(*field.Items, reflected.Index(index).Interface()); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func validateInteger(value any) error {
	switch typed := value.(type) {
	case json.Number:
		if _, err := strconv.ParseInt(typed.String(), 10, 64); err != nil {
			return errors.New("value must be an integer")
		}
	case float64:
		if math.IsNaN(typed) || math.IsInf(typed, 0) || math.Trunc(typed) != typed {
			return errors.New("value must be an integer")
		}
	case float32:
		if math.IsNaN(float64(typed)) || math.IsInf(float64(typed), 0) || float32(math.Trunc(float64(typed))) != typed {
			return errors.New("value must be an integer")
		}
	default:
		reflected := reflect.ValueOf(value)
		if !reflected.IsValid() {
			return errors.New("value must be an integer")
		}
		switch reflected.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		default:
			return errors.New("value must be an integer")
		}
	}
	return nil
}

func validateDecimal(value any) error {
	switch typed := value.(type) {
	case json.Number:
		if _, err := strconv.ParseFloat(typed.String(), 64); err != nil {
			return errors.New("value must be a decimal")
		}
	case float64:
		if math.IsNaN(typed) || math.IsInf(typed, 0) {
			return errors.New("value must be a decimal")
		}
	case float32:
		if math.IsNaN(float64(typed)) || math.IsInf(float64(typed), 0) {
			return errors.New("value must be a decimal")
		}
	default:
		reflected := reflect.ValueOf(value)
		if !reflected.IsValid() {
			return errors.New("value must be a decimal")
		}
		switch reflected.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		default:
			return errors.New("value must be a decimal")
		}
	}
	return nil
}

func validateMoney(value any) error {
	return validateDecimal(value)
}

func validateDate(value any) error {
	text, ok := value.(string)
	if !ok {
		return errors.New("date value must be a string")
	}
	if _, err := time.Parse("2006-01-02", text); err != nil {
		return errors.New("date value is invalid")
	}
	return nil
}

func validateURI(value any) error {
	text, ok := value.(string)
	if !ok {
		return errors.New("URI value must be a string")
	}
	parsed, err := url.Parse(text)
	if err != nil || parsed.Scheme == "" {
		return errors.New("URI value must be absolute")
	}
	return nil
}

func validateUUID(value any) error {
	text, ok := value.(string)
	if !ok || !uuid.MatchString(text) {
		return errors.New("UUID value is invalid")
	}
	return nil
}

var (
	uuid = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-8][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)
)
