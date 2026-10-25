package schema_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/fastygo/codex/content"
	"github.com/fastygo/codex/schema"
	"github.com/fastygo/codex/validation"
)

func TestResourceValidateEntry(t *testing.T) {
	entry := validMessage()
	if err := messageResource().ValidateEntry(entry); err != nil {
		t.Fatalf("ValidateEntry() error = %v", err)
	}
}

func TestResourceValidateEntryRejectsWrongLocationAndRule(t *testing.T) {
	tests := map[string]func(*content.Entry){
		"missing shared required": func(entry *content.Entry) {
			delete(entry.Metadata, "telegram_message_id")
		},
		"fractional integer": func(entry *content.Entry) {
			entry.Metadata["telegram_message_id"] = content.MetadataValue{Value: 10.5}
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			entry := validMessage()
			mutate(&entry)
			if err := messageResource().ValidateEntry(entry); err == nil {
				t.Fatal("ValidateEntry() error = nil")
			}
		})
	}
}

func TestResourceValidateEntryRelationCardinality(t *testing.T) {
	now := testTime()
	entry := content.Entry{
		ID: "conversation-1", Locale: "en", Kind: "conversation",
		Status: content.StatusPublished, Visibility: content.VisibilityPrivate,
		Slug: "conversation-1", Title: "Conversation 1",
		Metadata: map[string]content.MetadataValue{
			"telegram_chat_id": {Value: "42"},
			"messages":         {Value: []string{"message-1", "message-2"}},
		},
		Version: 1, CreatedAt: now, UpdatedAt: now,
	}
	if err := conversationResource().ValidateEntry(entry); err != nil {
		t.Fatalf("ValidateEntry() error = %v", err)
	}
	entry.Metadata["messages"] = content.MetadataValue{Value: "message-1"}
	if err := conversationResource().ValidateEntry(entry); err == nil {
		t.Fatal("ValidateEntry() accepted scalar one-to-many relation")
	}
}

func TestResourcePublicProjectionUsesSchemaSensitivity(t *testing.T) {
	resource := messageResource()
	resource.Record.Fields[0].Sensitive = true
	resource.Record.Fields[1].Sensitive = true
	entry := validMessage()

	projected, err := resource.PublicProjection(entry)
	if err != nil {
		t.Fatalf("PublicProjection() error = %v", err)
	}
	if _, exists := projected.Metadata["telegram_message_id"]; exists {
		t.Fatal("sensitive shared field survived projection")
	}
	if projected.Content != "" {
		t.Fatal("sensitive content chrome survived projection")
	}
}

func TestResourcePublicProjectionRemovesNestedSensitiveFields(t *testing.T) {
	resource := schema.Resource{Record: schema.RecordType{
		ID: "profile", Label: "Profiles", Scope: schema.ScopeTenant,
		Fields: []schema.Field{{
			ID: "details", Label: "Details", Type: schema.FieldObject,
			Fields: []schema.Field{
				{ID: "display_name", Label: "Display name", Type: schema.FieldString},
				{ID: "secret", Label: "Secret", Type: schema.FieldString, Sensitive: true},
			},
		}, {
			ID: "tokens", Label: "Tokens", Type: schema.FieldCollection,
			Items: &schema.Field{ID: "token", Label: "Token", Type: schema.FieldString, Sensitive: true},
		}},
	}}
	now := testTime()
	entry := content.Entry{
		ID: "profile-1", Locale: "en", Kind: "profile",
		Status: content.StatusPublished, Visibility: content.VisibilityPublic,
		Slug: "profile-1", Title: "Profile 1",
		Metadata: map[string]content.MetadataValue{
			"details": {Value: map[string]any{"display_name": "Alice", "secret": "hidden"}},
			"tokens":  {Value: []string{"first-secret", "second-secret"}},
		},
		Version: 1, CreatedAt: now, UpdatedAt: now,
	}
	projected, err := resource.PublicProjection(entry)
	if err != nil {
		t.Fatalf("PublicProjection() error = %v", err)
	}
	details := projected.Metadata["details"].Value.(map[string]any)
	if _, exists := details["secret"]; exists {
		t.Fatal("nested sensitive field survived projection")
	}
	if details["display_name"] != "Alice" {
		t.Fatalf("public nested field = %#v", details["display_name"])
	}
	if tokens := projected.Metadata["tokens"].Value.([]string); len(tokens) != 0 {
		t.Fatalf("sensitive collection items survived projection: %#v", tokens)
	}
}

func TestCodexSemanticRules(t *testing.T) {
	tests := []struct {
		name  string
		field int
		value any
		valid bool
	}{
		{name: "integer", field: 0, value: 42, valid: true},
		{name: "integer fraction", field: 0, value: 4.2},
		{name: "unsafe integer", field: 0, value: json.Number("9007199254740993")},
	}
	resource := messageResource()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			entry := validMessage()
			entry.Metadata[string(resource.Record.Fields[test.field].ID)] = content.MetadataValue{Value: test.value}
			err := resource.ValidateEntry(entry)
			if test.valid && err != nil {
				t.Fatalf("ValidateEntry() error = %v", err)
			}
			if !test.valid && err == nil {
				t.Fatal("ValidateEntry() error = nil")
			}
		})
	}
}

func TestMoneyIsSafeIntegerAndAllowsNegatives(t *testing.T) {
	resource := semanticResource()
	now := testTime()
	entry := semanticEntry(now)
	if err := resource.ValidateEntry(entry); err != nil {
		t.Fatalf("ValidateEntry() error = %v", err)
	}
	entry.Metadata["money_value"] = content.MetadataValue{Value: int64(-1250)}
	if err := resource.ValidateEntry(entry); err != nil {
		t.Fatalf("negative money error = %v", err)
	}
	entry.Metadata["money_value"] = content.MetadataValue{Value: json.Number("99.95")}
	if err := resource.ValidateEntry(entry); err == nil {
		t.Fatal("fractional money accepted")
	}
	delete(entry.Metadata, "currency")
	entry.Metadata["money_value"] = content.MetadataValue{Value: int64(1250)}
	err := resource.ValidateEntry(entry)
	if err == nil {
		t.Fatal("money without currency accepted")
	}
	if got := validation.Code(err); got != validation.ManifestCurrencyMissing {
		t.Fatalf("currency missing code = %q", got)
	}
	if err := resource.ValidateEntryWithCurrency(entry, "EUR"); err != nil {
		t.Fatalf("default currency error = %v", err)
	}
}

func TestGoBackendFieldSemanticsRoundTrip(t *testing.T) {
	resource := semanticResource()
	now := testTime()
	entry := semanticEntry(now)
	if err := resource.ValidateEntry(entry); err != nil {
		t.Fatalf("ValidateEntry() semantic profile error = %v", err)
	}

	entry.Metadata["date_value"] = content.MetadataValue{Value: "09/09/2026"}
	if err := resource.ValidateEntry(entry); err == nil {
		t.Fatal("ValidateEntry() accepted invalid Codex date")
	}
}

func TestMediaUsesARelation(t *testing.T) {
	manifest := schema.Manifest{
		Name: "media-example", Version: "1",
		Resources: []schema.Resource{
			{
				Record: schema.RecordType{
					ID: "article", Label: "Articles", Scope: schema.ScopeTenant,
					Fields: []schema.Field{{
						ID: "attachment", Label: "Attachment",
						Type: schema.FieldRelation, UIHint: schema.UIHintMedia,
					}},
					Relations: []schema.Relation{{
						ID: "attachment", Source: "article", Target: "media",
						Cardinality:    schema.RelationOneToOne,
						DeleteBehavior: schema.DeleteNullify,
					}},
				},
			},
			{Record: schema.RecordType{ID: "media", Label: "Media", Scope: schema.ScopeTenant}},
		},
	}
	if err := manifest.Validate(); err != nil {
		t.Fatalf("Validate() media relation error = %v", err)
	}
}

func TestEqualCanonical(t *testing.T) {
	if !schema.EqualCanonical(map[string]any{"b": 1, "a": 2}, map[string]any{"a": 2, "b": 1}) {
		t.Fatal("EqualCanonical() key order should not matter")
	}
	if schema.EqualCanonical(1, "1") {
		t.Fatal("EqualCanonical() mixed types")
	}
}

func validMessage() content.Entry {
	now := testTime()
	return content.Entry{
		ID: "telegram:group:42:1001", Locale: "und", Kind: "message",
		Status: content.StatusPublished, Visibility: content.VisibilityPrivate,
		Slug: "message-1001", Title: "Message 1001", Content: "Hello",
		Metadata: map[string]content.MetadataValue{
			"telegram_message_id": {Value: 1001},
		},
		Version: 1, CreatedAt: now, UpdatedAt: now,
	}
}

func semanticEntry(now time.Time) content.Entry {
	return content.Entry{
		ID: "semantic-1", Locale: "en", Kind: "semantic",
		Status: content.StatusPublished, Visibility: content.VisibilityPrivate,
		Slug: "semantic-1", Title: "Semantic 1",
		Metadata: map[string]content.MetadataValue{
			"integer_value":    {Value: json.Number("42")},
			"decimal_value":    {Value: json.Number("10.25")},
			"money_value":      {Value: json.Number("9995")},
			"currency":         {Value: "USD"},
			"date_value":       {Value: "2026-09-09"},
			"uri_value":        {Value: "https://example.test/item"},
			"uuid_value":       {Value: "018f0f4c-9b7a-7cc1-8d5f-5f4f44582000"},
			"nullable_value":   {Value: nil},
			"readonly_value":   {Value: "derived"},
			"json_value":       {Value: "scalar JSON is retained"},
			"enum_value":       {Value: "beta"},
			"object_value":     {Value: map[string]any{"name": "nested"}},
			"collection_value": {Value: []any{"first", "second"}},
		},
		Version: 1, CreatedAt: now, UpdatedAt: now,
	}
}

func semanticResource() schema.Resource {
	rule := func(name string) []schema.ValidationRule {
		return []schema.ValidationRule{{Name: name}}
	}
	return schema.Resource{Record: schema.RecordType{
		ID: "semantic", Label: "Semantic", Scope: schema.ScopeTenant,
		Fields: []schema.Field{
			{ID: "integer_value", Label: "Integer", Type: schema.FieldNumber, Rules: rule(schema.RuleInteger)},
			{ID: "decimal_value", Label: "Decimal", Type: schema.FieldNumber, Rules: rule(schema.RuleDecimal)},
			{ID: "money_value", Label: "Money", Type: schema.FieldNumber, Rules: rule(schema.RuleMoney)},
			{ID: "currency", Label: "Currency", Type: schema.FieldString},
			{ID: "date_value", Label: "Date", Type: schema.FieldDateTime, Rules: rule(schema.RuleDate)},
			{ID: "uri_value", Label: "URI", Type: schema.FieldString, Rules: rule(schema.RuleURI)},
			{ID: "uuid_value", Label: "UUID", Type: schema.FieldString, Rules: rule(schema.RuleUUID)},
			{ID: "nullable_value", Label: "Nullable", Type: schema.FieldString, Rules: rule(schema.RuleNullable)},
			{ID: "readonly_value", Label: "Read only", Type: schema.FieldString, Rules: rule(schema.RuleReadOnly)},
			{ID: "json_value", Label: "JSON", Type: schema.FieldJSON, Rules: rule(schema.RuleJSONAny)},
			{
				ID: "enum_value", Label: "Enum", Type: schema.FieldSelect,
				Options: []schema.Option{{Value: "alpha", Label: "Alpha"}, {Value: "beta", Label: "Beta"}},
			},
			{
				ID: "object_value", Label: "Object", Type: schema.FieldObject,
				Fields: []schema.Field{{ID: "name", Label: "Name", Type: schema.FieldString, Required: true}},
			},
			{
				ID: "collection_value", Label: "Collection", Type: schema.FieldCollection,
				Items: &schema.Field{ID: "item", Label: "Item", Type: schema.FieldString},
			},
		},
	}}
}

func testTime() time.Time {
	return time.Date(2026, time.September, 9, 12, 0, 0, 0, time.UTC)
}
