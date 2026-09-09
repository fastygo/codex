package schema_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/fastygo/codex/content"
	"github.com/fastygo/codex/schema"
	"github.com/fastygo/formset"
)

func TestManifestWithProductResources(t *testing.T) {
	manifest := schema.WithCoreResources(schema.Manifest{
		Name:    "telegram-reader",
		Version: "1",
		Resources: []schema.Resource{
			conversationResource(),
			messageResource(),
		},
	})

	if err := manifest.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if len(manifest.Resources) != 6 {
		t.Fatalf("resource count = %d, want 6", len(manifest.Resources))
	}
	resource, ok := manifest.Resource("message")
	if !ok || resource.Collection != "messages" {
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

func TestCanonicalIsDeterministicAndNonMutating(t *testing.T) {
	conversation := conversationResource()
	conversation.Capabilities = []formset.CapabilityID{"write", "read"}
	conversation.Relations[0].Policy.AllowedTargets = []string{"workspace-b", "workspace-a"}
	manifest := schema.Manifest{
		Name:    "telegram-reader",
		Version: "1",
		Resources: []schema.Resource{
			messageResource(),
			conversation,
		},
	}
	original := manifest.Clone()
	canonical := manifest.Canonical()

	if got := canonical.Resources[0].ID; got != "conversation" {
		t.Fatalf("first resource = %q, want conversation", got)
	}
	if !reflect.DeepEqual(manifest, original) {
		t.Fatal("Canonical() mutated its input")
	}
	if got := canonical.Resources[1].Fields[0].ID; got != "telegram_message_id" {
		t.Fatalf("field order changed: first = %q", got)
	}
	if got := canonical.Resources[0].Capabilities[0]; got != "read" {
		t.Fatalf("capabilities not canonical: first = %q", got)
	}
	if got := canonical.Resources[0].Relations[0].Policy.AllowedTargets[0]; got != "workspace-a" {
		t.Fatalf("allowed targets not canonical: first = %q", got)
	}
}

func TestManifestJSONRoundTrip(t *testing.T) {
	manifest := schema.Manifest{
		Name:      "telegram-reader",
		Version:   "1",
		Resources: []schema.Resource{conversationResource(), messageResource()},
	}.Canonical()
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

func TestWithCoreResourcesPreservesProductCoreOverride(t *testing.T) {
	customPost := schema.CoreResources()[0]
	customPost.Label = "Articles"
	manifest := schema.WithCoreResources(schema.Manifest{
		Name:      "product",
		Version:   "1",
		Resources: []schema.Resource{customPost},
	})
	resource, ok := manifest.Resource(content.KindPost)
	if !ok || resource.Label != "Articles" {
		t.Fatalf("post resource = %#v, %v", resource, ok)
	}
	if len(manifest.Resources) != 4 {
		t.Fatalf("resource count = %d, want 4", len(manifest.Resources))
	}
}

func TestManifestRejectsDuplicateContractValues(t *testing.T) {
	tests := map[string]schema.Manifest{
		"resource": {
			Name: "x", Version: "1",
			Resources: []schema.Resource{conversationResource(), conversationResource()},
		},
		"collection": {
			Name: "x", Version: "1",
			Resources: []schema.Resource{
				conversationResource(),
				func() schema.Resource {
					value := messageResource()
					value.Collection = "conversations"
					return value
				}(),
			},
		},
		"taxonomy": {
			Name: "x", Version: "1",
			Resources: []schema.Resource{
				func() schema.Resource {
					value := messageResource()
					value.Taxonomies = []string{"topic", "topic"}
					return value
				}(),
			},
		},
		"capability": {
			Name: "x", Version: "1",
			Resources: []schema.Resource{
				func() schema.Resource {
					value := messageResource()
					value.Capabilities = []formset.CapabilityID{"read", "read"}
					return value
				}(),
			},
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

func TestResourceRejectsReservedCollection(t *testing.T) {
	resource := messageResource()
	resource.Collection = "settings"
	if err := resource.Validate(); err == nil {
		t.Fatal("Validate() error = nil")
	}
}

func TestManifestPreservesFormSetExtensionVocabulary(t *testing.T) {
	manifest := schema.Manifest{
		Name:    "extension",
		Version: "1",
		Resources: []schema.Resource{{
			RecordType: formset.RecordType{
				ID:         "custom",
				Label:      "Custom",
				Scope:      formset.Scope("product-scope"),
				Fields:     []formset.Field{{ID: "value", Label: "Value", Type: formset.FieldType("product-field")}},
				Visibility: "product-visibility",
			},
			Collection: "customs",
		}},
	}
	if err := manifest.Validate(); err != nil {
		t.Fatalf("Validate() extension error = %v", err)
	}
}

func conversationResource() schema.Resource {
	return schema.Resource{
		RecordType: formset.RecordType{
			ID:            "conversation",
			Label:         "Conversations",
			SchemaVersion: "1",
			OwnerModule:   "telegram-reader",
			Scope:         formset.ScopeUser,
			Fields: []formset.Field{
				{ID: "telegram_chat_id", Label: "Telegram chat ID", Type: formset.FieldString, Required: true, Indexed: true},
				{ID: "chat_type", Label: "Chat type", Type: formset.FieldSelect},
			},
			Relations: []formset.Relation{
				{
					ID: "conversation_messages", Source: "conversation", Target: "message",
					Cardinality:    formset.RelationOneToMany,
					DeleteBehavior: formset.DeleteRestrict,
				},
			},
		},
		Collection: "conversations",
	}
}

func messageResource() schema.Resource {
	return schema.Resource{
		RecordType: formset.RecordType{
			ID:            formset.RecordTypeID(content.Kind("message")),
			Label:         "Messages",
			SchemaVersion: "1",
			OwnerModule:   "telegram-reader",
			Scope:         formset.ScopeUser,
			Fields: []formset.Field{
				{ID: "telegram_message_id", Label: "Telegram message ID", Type: formset.FieldNumber, Required: true, Indexed: true},
				{ID: "content", Label: "Content", Type: formset.FieldText, Searchable: true},
			},
		},
		Collection: "messages",
		Taxonomies: []string{"message_type"},
	}
}
