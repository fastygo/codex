package schema_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/fastygo/codex/schema"
	"github.com/fastygo/formset"
)

func TestManifestWithProductResources(t *testing.T) {
	manifest := productManifest()
	if err := manifest.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	resource, ok := manifest.Resource("message")
	if !ok || resource.Record.ID != "message" {
		t.Fatalf("Resource(message) = %#v, %v", resource, ok)
	}
}

func TestManifestRejectsMissingRelationTarget(t *testing.T) {
	manifest := schema.Manifest{
		Name:      "telegram-reader",
		Version:   "1",
		Resources: []schema.Resource{conversationResource()},
	}
	if err := manifest.Validate(); err == nil {
		t.Fatal("Validate() error = nil")
	}
}

func TestCanonicalAndDigestAreDeterministicAndNonMutating(t *testing.T) {
	conversation := conversationResource()
	conversation.Record.Capabilities = []formset.CapabilityID{"write", "read"}
	conversation.Record.Relations[0].Policy.AllowedTargets = []string{"workspace-b", "workspace-a"}
	manifest := schema.Manifest{
		Name:      "telegram-reader",
		Version:   "1",
		Resources: []schema.Resource{messageResource(), conversation},
	}
	original := manifest.Clone()
	canonical := manifest.Canonical()

	if got := canonical.Resources[0].Record.ID; got != "conversation" {
		t.Fatalf("first resource = %q, want conversation", got)
	}
	if !reflect.DeepEqual(manifest, original) {
		t.Fatal("Canonical() mutated its input")
	}
	if got := canonical.Resources[1].Record.Fields[0].ID; got != "telegram_message_id" {
		t.Fatalf("field order changed: first = %q", got)
	}
	if got := canonical.Resources[0].Record.Capabilities[0]; got != "read" {
		t.Fatalf("capabilities not canonical: first = %q", got)
	}
	if got := canonical.Resources[0].Record.Relations[0].Policy.AllowedTargets[0]; got != "workspace-a" {
		t.Fatalf("allowed targets not canonical: first = %q", got)
	}
	first, err := manifest.Digest()
	if err != nil {
		t.Fatalf("Digest() error = %v", err)
	}
	second, err := canonical.Digest()
	if err != nil {
		t.Fatalf("canonical Digest() error = %v", err)
	}
	if first != second {
		t.Fatalf("digest differs by set ordering: %q != %q", first, second)
	}
}

func TestManifestJSONRoundTrip(t *testing.T) {
	manifest := productManifest().Canonical()
	encoded, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var decoded schema.Manifest
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if !reflect.DeepEqual(decoded, manifest) {
		t.Fatalf("round trip mismatch:\n got %#v\nwant %#v", decoded, manifest)
	}
}

func TestManifestRejectsDuplicateContractValues(t *testing.T) {
	tests := map[string]schema.Manifest{
		"resource": {
			Name: "x", Version: "1",
			Resources: []schema.Resource{messageResource(), messageResource()},
		},
		"taxonomy": {
			Name: "x", Version: "1",
			Resources: []schema.Resource{func() schema.Resource {
				value := messageResource()
				value.Taxonomies = []string{"topic", "topic"}
				return value
			}()},
		},
		"capability": {
			Name: "x", Version: "1",
			Resources: []schema.Resource{func() schema.Resource {
				value := messageResource()
				value.Record.Capabilities = []formset.CapabilityID{"read", "read"}
				return value
			}()},
		},
	}
	for name, manifest := range tests {
		t.Run(name, func(t *testing.T) {
			if err := manifest.Validate(); err == nil {
				t.Fatal("Validate() error = nil")
			}
		})
	}
}

func TestManifestPreservesFormSetExtensionVocabulary(t *testing.T) {
	manifest := schema.Manifest{
		Name:    "extension",
		Version: "1",
		Resources: []schema.Resource{{Record: formset.RecordType{
			ID:         "custom",
			Label:      "Custom",
			Scope:      formset.Scope("product-scope"),
			Fields:     []formset.Field{{ID: "value", Label: "Value", Type: formset.FieldType("product-field")}},
			Visibility: "product-visibility",
		}}},
	}
	if err := manifest.Validate(); err != nil {
		t.Fatalf("Validate() extension error = %v", err)
	}
}

func TestRelationRequiresMatchingNonLocalizedField(t *testing.T) {
	resource := conversationResource()
	resource.Record.Fields[2].ID = "other"
	if err := resource.Validate(); err == nil {
		t.Fatal("Validate() error = nil")
	}
}

func TestEntryChromeFieldMustBeLocalized(t *testing.T) {
	resource := messageResource()
	resource.Record.Fields[1].Localized = false
	if err := resource.Validate(); err == nil {
		t.Fatal("Validate() accepted non-localized content field")
	}
}

func TestFieldProfileRules(t *testing.T) {
	valid := formset.Field{
		ID: "count", Label: "Count", Type: formset.FieldNumber,
		Rules: []formset.ValidationRule{{Name: schema.RuleInteger}},
	}
	if err := schema.ValidateFieldProfile(valid); err != nil {
		t.Fatalf("ValidateFieldProfile() error = %v", err)
	}
	invalid := valid
	invalid.Type = formset.FieldString
	if err := schema.ValidateFieldProfile(invalid); err == nil {
		t.Fatal("ValidateFieldProfile() error = nil")
	}
}

func productManifest() schema.Manifest {
	return schema.Manifest{
		Name:      "telegram-reader",
		Version:   "1",
		Resources: []schema.Resource{conversationResource(), messageResource()},
	}
}

func conversationResource() schema.Resource {
	return schema.Resource{
		Record: formset.RecordType{
			ID:            "conversation",
			Label:         "Conversations",
			SchemaVersion: "1",
			OwnerModule:   "telegram-reader",
			Scope:         formset.ScopeUser,
			Fields: []formset.Field{
				{ID: "telegram_chat_id", Label: "Telegram chat ID", Type: formset.FieldString, Required: true, Indexed: true},
				{ID: "chat_type", Label: "Chat type", Type: formset.FieldSelect},
				{ID: "messages", Label: "Messages", Type: formset.FieldRelation},
			},
			Relations: []formset.Relation{{
				ID: "messages", Source: "conversation", Target: "message",
				Cardinality:    formset.RelationOneToMany,
				DeleteBehavior: formset.DeleteRestrict,
			}},
		},
	}
}

func messageResource() schema.Resource {
	return schema.Resource{
		Record: formset.RecordType{
			ID:            "message",
			Label:         "Messages",
			SchemaVersion: "1",
			OwnerModule:   "telegram-reader",
			Scope:         formset.ScopeUser,
			Fields: []formset.Field{
				{
					ID: "telegram_message_id", Label: "Telegram message ID",
					Type: formset.FieldNumber, Required: true, Indexed: true,
					Rules: []formset.ValidationRule{{Name: schema.RuleInteger}},
				},
				{ID: "content", Label: "Content", Type: formset.FieldText, Localized: true, Searchable: true},
			},
		},
		Taxonomies: []string{"message_type"},
	}
}
