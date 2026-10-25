package schema

import (
	"encoding/json"
	"math"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fastygo/codex/canonical"
	"github.com/fastygo/codex/content"
	"github.com/fastygo/codex/validation"
)

func (resource Resource) ValidateEntry(entry content.Entry) error {
	return resource.validateEntry(entry, "")
}

func (resource Resource) ValidateEntryWithCurrency(entry content.Entry, defaultCurrency string) error {
	return resource.validateEntry(entry, defaultCurrency)
}

func (resource Resource) validateEntry(entry content.Entry, defaultCurrency string) error {
	if err := resource.Validate(); err != nil {
		return err
	}
	if content.Kind(resource.Record.ID) != entry.Kind {
		return validation.New("schema.entry.kind_mismatch", "kind", "entry kind does not match resource")
	}
	if err := entry.Validate(); err != nil {
		return err
	}

	core := coreEntryKeys()
	fields := make([]Field, 0, len(resource.Record.Fields))
	for _, field := range resource.Record.Fields {
		if core[string(field.ID)] {
			continue
		}
		fields = append(fields, field)
	}
	md := entry.Metadata
	if md == nil {
		md = map[string]content.MetadataValue{}
	}
	byID := map[string]content.Kind{}
	issues := CheckMetadataFields(resource, fields, md, "metadata", byID)
	if moneyIssue := CheckMoneyCurrency(fields, md, defaultCurrency, ""); moneyIssue.Code != "" {
		issues = append(issues, moneyIssue)
	}
	if len(issues) > 0 {
		return issues[0].AsError()
	}
	return validateRelationValues(resource, md)
}

func coreEntryKeys() map[string]bool {
	keys := map[string]bool{}
	for _, k := range []string{
		"id", "locale", "kind", "status", "visibility", "slug", "title", "content", "excerpt",
		"author_id", "parent_id", "featured_media_id", "template", "terms", "order", "pinned",
		"format", "discussion", "origin", "metadata", "version", "created_at", "updated_at",
		"published_at", "deleted_at",
	} {
		keys[k] = true
	}
	return keys
}

// CheckMetadataFields reports declared-field issues for a metadata map.
func CheckMetadataFields(resource Resource, fields []Field, md map[string]content.MetadataValue, prefix string, byID map[string]content.Kind) []validation.Issue {
	var issues []validation.Issue
	for _, field := range fields {
		item, exists := md[string(field.ID)]
		p := prefix + "." + string(field.ID)
		if !exists || isAbsent(item.Value) {
			if field.Required {
				issues = append(issues, validation.IssueOf(validation.ManifestFieldRequired, p, `field "`+string(field.ID)+`" is required`))
			}
			continue
		}
		var rel *Relation
		for i := range resource.Record.Relations {
			if resource.Record.Relations[i].ID == RelationID(field.ID) {
				rel = &resource.Record.Relations[i]
				break
			}
		}
		if msg := valueIssue(field, item.Value, rel, byID); msg != "" {
			issues = append(issues, validation.IssueOf(validation.ManifestFieldInvalid, p, msg))
		}
	}
	return issues
}

func isAbsent(v any) bool {
	if v == nil {
		return true
	}
	s, ok := v.(string)
	return ok && s == ""
}

// CheckMoneyCurrency reports a missing or invalid ISO 4217 currency for money fields.
func CheckMoneyCurrency(fields []Field, md map[string]content.MetadataValue, defaultCurrency, file string) validation.Issue {
	var money []Field
	for _, field := range fields {
		if len(moneyRules(field)) == 0 {
			continue
		}
		item, exists := md[string(field.ID)]
		if !exists || item.Value == nil {
			continue
		}
		money = append(money, field)
	}
	if len(money) == 0 {
		return validation.Issue{}
	}
	cfs := map[string]struct{}{}
	var order []string
	for _, field := range money {
		for _, rule := range moneyRules(field) {
			cf := "currency"
			if rule.Args != nil && rule.Args["currency_field"] != "" {
				cf = rule.Args["currency_field"]
			}
			if _, seen := cfs[cf]; !seen {
				order = append(order, cf)
				cfs[cf] = struct{}{}
			}
		}
	}
	var chosen string
	for _, cf := range order {
		if _, exists := md[cf]; exists {
			chosen = cf
			break
		}
	}
	pathPrefix := "metadata"
	if file != "" {
		pathPrefix = file + ".metadata"
	}
	if chosen != "" {
		own := md[chosen].Value
		text, ok := own.(string)
		if !ok || !content.ValidCurrency(text) {
			return validation.IssueOf(validation.ManifestCurrencyInvalid, pathPrefix+"."+chosen, "currency must be an ISO 4217 code like USD")
		}
		return validation.Issue{}
	}
	if defaultCurrency == "" {
		return validation.IssueOf(validation.ManifestCurrencyMissing, pathPrefix+"."+string(money[0].ID), `money field "`+string(money[0].ID)+`" needs a currency`)
	}
	if !content.ValidCurrency(defaultCurrency) {
		return validation.IssueOf(validation.ManifestCurrencyInvalid, pathPrefix+"."+string(money[0].ID), "currency must be an ISO 4217 code like USD")
	}
	return validation.Issue{}
}

func moneyRules(field Field) []ValidationRule {
	var out []ValidationRule
	for _, rule := range field.Rules {
		if rule.Name == RuleMoney {
			out = append(out, rule)
		}
	}
	if field.Items != nil {
		out = append(out, moneyRules(*field.Items)...)
	}
	for _, nested := range field.Fields {
		out = append(out, moneyRules(nested)...)
	}
	return out
}

func valueIssue(field Field, v any, rel *Relation, byID map[string]content.Kind) string {
	id := string(field.ID)
	switch field.Type {
	case FieldNumber:
		if !isFiniteNumber(v) {
			return `"` + id + `" must be a number`
		}
	case FieldBoolean:
		if _, ok := v.(bool); !ok {
			return `"` + id + `" must be a boolean`
		}
	case FieldSelect:
		text, ok := v.(string)
		if !ok {
			return `"` + id + `" must be a string`
		}
		if len(field.Options) > 0 {
			ok = false
			for _, option := range field.Options {
				if option.Value == text {
					ok = true
					break
				}
			}
			if !ok {
				return `"` + id + `" must be one of the declared options`
			}
		}
	case FieldDateTime:
		text, ok := v.(string)
		if !ok {
			return `"` + id + `" must be a string`
		}
		if _, hasDate := FieldRule(field, RuleDate); !hasDate && !rfc3339Like(text) {
			return `"` + id + `" must be an RFC 3339 date-time`
		}
	case FieldText, FieldString, FieldTextarea, FieldRichText, FieldMarkdown:
		if _, ok := v.(string); !ok {
			return `"` + id + `" must be a string`
		}
	case FieldObject:
		document, ok := v.(map[string]any)
		if !ok {
			return `"` + id + `" must be an object`
		}
		for _, nested := range field.Fields {
			nv, exists := document[string(nested.ID)]
			if !exists || isAbsent(nv) {
				if nested.Required {
					return `"` + id + `.` + string(nested.ID) + `" is required`
				}
				continue
			}
			if msg := valueIssue(nested, nv, nil, byID); msg != "" {
				return strings.Replace(msg, `"`+string(nested.ID)+`"`, `"`+id+`.`+string(nested.ID)+`"`, 1)
			}
		}
	case FieldCollection:
		values, ok := asSlice(v)
		if !ok {
			return `"` + id + `" must be an array`
		}
		if field.Items != nil {
			for i, item := range values {
				if msg := valueIssue(*field.Items, item, nil, byID); msg != "" {
					return `"` + id + `[` + strconv.Itoa(i) + `]": ` + msg
				}
			}
		}
	case FieldRelation:
		many := false
		if rel != nil {
			many = rel.Cardinality != RelationOneToOne
		} else {
			_, many = asSlice(v)
		}
		var ids []any
		if many {
			var ok bool
			ids, ok = asSlice(v)
			if !ok {
				return `"` + id + `" must be an array of ids`
			}
		} else {
			ids = []any{v}
		}
		for _, item := range ids {
			if !nonEmptyID(item) {
				return `"` + id + `" must hold non-empty id strings`
			}
		}
		if len(byID) > 0 && rel != nil {
			for _, item := range ids {
				targetID := idString(item)
				kind, exists := byID[targetID]
				if !exists {
					return `"` + id + `" references missing entry "` + targetID + `"`
				}
				if kind != content.Kind(rel.Target) {
					return `"` + id + `" targets kind "` + string(rel.Target) + `", entry "` + targetID + `" has kind "` + string(kind) + `"`
				}
			}
		}
	}
	for _, rule := range field.Rules {
		switch rule.Name {
		case RuleInteger:
			if !isSafeInteger(v) {
				return `"` + id + `" must be a safe integer (|n| <= 2^53 - 1)`
			}
		case RuleMoney:
			// Q9: money is a safe integer in minor units; negatives are allowed; no min rule.
			if !isSafeInteger(v) {
				return `"` + id + `" is money: an integer in minor units (cents), e.g. 1250 for 12.50`
			}
		case RuleURI:
			text, ok := v.(string)
			if !ok {
				return `"` + id + `" must be a URI`
			}
			parsed, err := url.Parse(text)
			if err != nil || parsed.Scheme == "" {
				return `"` + id + `" must be a URI`
			}
		case RuleDecimal:
			if !isFiniteNumber(v) {
				return `"` + id + `" must be a decimal`
			}
		case RuleDate:
			text, ok := v.(string)
			if !ok {
				return `"` + id + `" must be a date`
			}
			if _, err := time.Parse("2006-01-02", text); err != nil {
				return `"` + id + `" must be a date`
			}
		case RuleUUID:
			text, ok := v.(string)
			if !ok || !uuidPattern.MatchString(text) {
				return `"` + id + `" must be a UUID`
			}
		case RuleJSONAny:
			if err := content.ValidateJSONValue(v); err != nil {
				return `"` + id + `" must be JSON-native`
			}
		case RuleNullable, RuleReadOnly:
		}
	}
	return ""
}

func rfc3339Like(s string) bool {
	if _, err := time.Parse(time.RFC3339, s); err == nil {
		return true
	}
	if _, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return true
	}
	return false
}

func asSlice(v any) ([]any, bool) {
	switch t := v.(type) {
	case []any:
		return t, true
	case []string:
		out := make([]any, len(t))
		for i, s := range t {
			out[i] = s
		}
		return out, true
	default:
		reflected := reflect.ValueOf(v)
		if !reflected.IsValid() || (reflected.Kind() != reflect.Slice && reflected.Kind() != reflect.Array) {
			return nil, false
		}
		out := make([]any, reflected.Len())
		for i := 0; i < reflected.Len(); i++ {
			out[i] = reflected.Index(i).Interface()
		}
		return out, true
	}
}

func idString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case content.ID:
		return string(t)
	default:
		return ""
	}
}

func isFiniteNumber(v any) bool {
	switch t := v.(type) {
	case json.Number:
		f, err := t.Float64()
		return err == nil && !math.IsNaN(f) && !math.IsInf(f, 0)
	case float64:
		return !math.IsNaN(t) && !math.IsInf(t, 0)
	case float32:
		return !math.IsNaN(float64(t)) && !math.IsInf(float64(t), 0)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return true
	default:
		return false
	}
}

func isSafeInteger(v any) bool {
	switch t := v.(type) {
	case json.Number:
		s := t.String()
		if strings.ContainsAny(s, ".eE") {
			f, err := t.Float64()
			return err == nil && f == math.Trunc(f) && math.Abs(f) <= float64(content.MaxSafeInteger)
		}
		i, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return false
		}
		if i < 0 {
			return i >= -content.MaxSafeInteger
		}
		return i <= content.MaxSafeInteger
	case float64:
		return t == math.Trunc(t) && math.Abs(t) <= float64(content.MaxSafeInteger) && !math.IsNaN(t) && !math.IsInf(t, 0)
	case int:
		return int64(t) <= content.MaxSafeInteger && int64(t) >= -content.MaxSafeInteger
	case int64:
		return t <= content.MaxSafeInteger && t >= -content.MaxSafeInteger
	case int32:
		return true
	case uint64:
		return t <= uint64(content.MaxSafeInteger)
	case uint:
		return uint64(t) <= uint64(content.MaxSafeInteger)
	default:
		reflected := reflect.ValueOf(v)
		if !reflected.IsValid() {
			return false
		}
		switch reflected.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n := reflected.Int()
			return n <= content.MaxSafeInteger && n >= -content.MaxSafeInteger
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return reflected.Uint() <= uint64(content.MaxSafeInteger)
		default:
			return false
		}
	}
}

func (resource Resource) PublicProjection(entry content.Entry) (content.Entry, error) {
	if err := resource.ValidateEntry(entry); err != nil {
		return content.Entry{}, err
	}
	projected := entry.PublicProjection()
	for _, field := range resource.Record.Fields {
		if content.ChromeField(string(field.ID)) {
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
	if len(projected.Metadata) == 0 {
		projected.Metadata = nil
	}
	return projected, nil
}

func projectSensitive(field Field, value any) (any, bool) {
	if field.Sensitive {
		return nil, false
	}
	if field.Type == FieldObject {
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
	if field.Type == FieldCollection && field.Items != nil {
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

func clearEntryChromeField(entry *content.Entry, fieldID string) {
	switch fieldID {
	case "slug":
		entry.Slug = ""
	case "title":
		entry.Title = ""
	case "content":
		entry.Content = ""
	case "excerpt":
		entry.Excerpt = ""
	}
}

func validateRelationValues(resource Resource, metadata map[string]content.MetadataValue) error {
	for _, relation := range resource.Record.Relations {
		value, exists := metadata[string(relation.ID)]
		if !exists || value.Value == nil {
			continue
		}
		switch relation.Cardinality {
		case RelationOneToOne:
			if !nonEmptyID(value.Value) {
				return validation.New("schema.entry.relation_scalar", string(relation.ID), "relation requires one id")
			}
		case RelationOneToMany, RelationManyToMany:
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
	values, ok := asSlice(value)
	if !ok {
		return false
	}
	for _, item := range values {
		if !nonEmptyID(item) {
			return false
		}
	}
	return true
}

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-8][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)

// EqualCanonical reports whether two JSON-native values have the same JCS form.
// Used for Q7 localized:false equality across translations of one id.
func EqualCanonical(left, right any) bool {
	return canonical.Format(left) == canonical.Format(right)
}
