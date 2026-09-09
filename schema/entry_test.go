package schema_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/fastygo/codex/content"
	"github.com/fastygo/codex/schema"
	"github.com/fastygo/formset"
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
		"chrome mismatch": func(entry *content.Entry) {
			document := entry.Locales["und"]
			document.Data["content"] = "different"
			entry.Locales["und"] = document
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
		ID: "conversation-1", Kind: "conversation",
		Status: content.StatusPublished, Visibility: content.VisibilityPrivate,
		Slug:  content.LocalizedText{"und": "conversation-1"},
		Title: content.LocalizedText{"und": "Conversation 1"},
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
	if _, exists := projected.Locales["und"].Data["content"]; exists {
		t.Fatal("sensitive localized field survived projection")
	}
	if projected.Content != nil {
		t.Fatal("sensitive content chrome survived projection")
	}
}

func TestResourcePublicProjectionRemovesNestedSensitiveFields(t *testing.T) {
	resource := schema.Resource{Record: formset.RecordType{
		ID: "profile", Label: "Profiles", Scope: formset.ScopeTenant,
		Fields: []formset.Field{{
			ID: "details", Label: "Details", Type: formset.FieldObject,
			Fields: []formset.Field{
				{ID: "display_name", Label: "Display name", Type: formset.FieldString},
				{ID: "secret", Label: "Secret", Type: formset.FieldString, Sensitive: true},
			},
		}, {
			ID: "tokens", Label: "Tokens", Type: formset.FieldCollection,
			Items: &formset.Field{ID: "token", Label: "Token", Type: formset.FieldString, Sensitive: true},
		}},
	}}
	now := testTime()
	entry := content.Entry{
		ID: "profile-1", Kind: "profile",
		Status: content.StatusPublished, Visibility: content.VisibilityPublic,
		Slug: content.LocalizedText{"en": "profile-1"}, Title: content.LocalizedText{"en": "Profile 1"},
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

func TestGoBackendFieldSemanticsHaveLosslessFormSetProfile(t *testing.T) {
	resource := semanticResource()
	now := testTime()
	entry := content.Entry{
		ID: "semantic-1", Kind: "semantic",
		Status: content.StatusPublished, Visibility: content.VisibilityPrivate,
		Slug:  content.LocalizedText{"en": "semantic-1"},
		Title: content.LocalizedText{"en": "Semantic 1"},
		Metadata: map[string]content.MetadataValue{
			"integer_value":  {Value: json.Number("42")},
			"decimal_value":  {Value: json.Number("10.25")},
			"money_value":    {Value: json.Number("99.95")},
			"date_value":     {Value: "2026-09-09"},
			"uri_value":      {Value: "https://example.test/item"},
			"uuid_value":     {Value: "018f0f4c-9b7a-7cc1-8d5f-5f4f44582000"},
			"nullable_value": {Value: nil},
			"readonly_value": {Value: "derived"},
			"json_value":     {Value: "scalar JSON is retained"},
			"enum_value":     {Value: "beta"},
			"object_value":   {Value: map[string]any{"name": "nested"}},
			"collection_value": {Value: []any{
				"first", "second",
			}},
		},
		Version: 1, CreatedAt: now, UpdatedAt: now,
	}
	if err := resource.ValidateEntry(entry); err != nil {
		t.Fatalf("ValidateEntry() semantic profile error = %v", err)
	}

	entry.Metadata["date_value"] = content.MetadataValue{Value: "09/09/2026"}
	if err := resource.ValidateEntry(entry); err == nil {
		t.Fatal("ValidateEntry() accepted invalid Codex date")
	}
}

func TestMediaUsesAFormSetRelation(t *testing.T) {
	manifest := schema.Manifest{
		Name: "media-example", Version: "1",
		Resources: []schema.Resource{
			{
				Record: formset.RecordType{
					ID: "article", Label: "Articles", Scope: formset.ScopeTenant,
					Fields: []formset.Field{{
						ID: "attachment", Label: "Attachment",
						Type: formset.FieldRelation, UIHint: schema.UIHintMedia,
					}},
					Relations: []formset.Relation{{
						ID: "attachment", Source: "article", Target: "media",
						Cardinality:    formset.RelationOneToOne,
						DeleteBehavior: formset.DeleteNullify,
					}},
				},
			},
			{Record: formset.RecordType{ID: "media", Label: "Media", Scope: formset.ScopeTenant}},
		},
	}
	if err := manifest.Validate(); err != nil {
		t.Fatalf("Validate() media relation error = %v", err)
	}
}

func validMessage() content.Entry {
	now := testTime()
	return content.Entry{
		ID: "telegram:group:42:1001", Kind: "message",
		Status: content.StatusPublished, Visibility: content.VisibilityPrivate,
		Slug:    content.LocalizedText{"und": "message-1001"},
		Title:   content.LocalizedText{"und": "Message 1001"},
		Content: content.LocalizedText{"und": "Hello"},
		Metadata: map[string]content.MetadataValue{
			"telegram_message_id": {Value: 1001},
		},
		Locales: map[string]content.LocaleDocument{
			"und": {
				Data:      map[string]any{"content": "Hello"},
				Status:    content.StatusPublished,
				UpdatedAt: now,
			},
		},
		Version: 1, CreatedAt: now, UpdatedAt: now,
	}
}

func semanticResource() schema.Resource {
	rule := func(name string) []formset.ValidationRule {
		return []formset.ValidationRule{{Name: name}}
	}
	return schema.Resource{Record: formset.RecordType{
		ID: "semantic", Label: "Semantic", Scope: formset.ScopeTenant,
		Fields: []formset.Field{
			{ID: "integer_value", Label: "Integer", Type: formset.FieldNumber, Rules: rule(schema.RuleInteger)},
			{ID: "decimal_value", Label: "Decimal", Type: formset.FieldNumber, Rules: rule(schema.RuleDecimal)},
			{ID: "money_value", Label: "Money", Type: formset.FieldNumber, Rules: rule(schema.RuleMoney)},
			{ID: "date_value", Label: "Date", Type: formset.FieldDateTime, Rules: rule(schema.RuleDate)},
			{ID: "uri_value", Label: "URI", Type: formset.FieldString, Rules: rule(schema.RuleURI)},
			{ID: "uuid_value", Label: "UUID", Type: formset.FieldString, Rules: rule(schema.RuleUUID)},
			{ID: "nullable_value", Label: "Nullable", Type: formset.FieldString, Rules: rule(schema.RuleNullable)},
			{ID: "readonly_value", Label: "Read only", Type: formset.FieldString, Rules: rule(schema.RuleReadOnly)},
			{ID: "json_value", Label: "JSON", Type: formset.FieldJSON, Rules: rule(schema.RuleJSONAny)},
			{
				ID: "enum_value", Label: "Enum", Type: formset.FieldSelect,
				Options: []formset.Option{{Value: "alpha", Label: "Alpha"}, {Value: "beta", Label: "Beta"}},
			},
			{
				ID: "object_value", Label: "Object", Type: formset.FieldObject,
				Fields: []formset.Field{{ID: "name", Label: "Name", Type: formset.FieldString, Required: true}},
			},
			{
				ID: "collection_value", Label: "Collection", Type: formset.FieldCollection,
				Items: &formset.Field{ID: "item", Label: "Item", Type: formset.FieldString},
			},
		},
	}}
}

func testTime() time.Time {
	return time.Date(2026, time.September, 9, 12, 0, 0, 0, time.UTC)
}
