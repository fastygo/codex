package validate_test

import (
	"testing"

	"github.com/fastygo/codex/schema"
	"github.com/fastygo/codex/validate"
	"github.com/fastygo/codex/validation"
)

func sampleEntry(id, locale string, extra map[string]any) map[string]any {
	o := map[string]any{
		"id": id, "locale": locale, "kind": "post", "status": "published",
		"visibility": "public", "slug": "hello-" + locale, "title": "Hello",
		"version": jsonNumber(1), "created_at": "2026-10-25T11:09:00Z", "updated_at": "2026-10-25T11:09:00Z",
	}
	for k, v := range extra {
		o[k] = v
	}
	return o
}

func jsonNumber(n int64) any {
	return float64(n)
}

func TestCollectionV04UniqueKey(t *testing.T) {
	issues := validate.CollectionV04(validate.Collection{
		Entries: []validate.RecordFile{
			{File: "a.json", Data: sampleEntry("post-1", "en", nil)},
			{File: "b.json", Data: sampleEntry("post-1", "en", nil)},
		},
	}, validate.Options{})
	if !hasCode(issues, validation.CoreEntryKeyDuplicated) {
		t.Fatalf("issues = %#v", issues)
	}
}

func TestCollectionV04UnknownFieldAndInvalidID(t *testing.T) {
	unknown := sampleEntry("post-1", "en", map[string]any{"password": "x"})
	issues := validate.CollectionV04(validate.Collection{
		Entries: []validate.RecordFile{{File: "a.json", Data: unknown}},
	}, validate.Options{})
	if !hasCode(issues, validation.DecodeUnknownField) {
		t.Fatalf("unknown field issues = %#v", issues)
	}

	badID := sampleEntry("пост-1", "en", nil)
	issues = validate.CollectionV04(validate.Collection{
		Entries: []validate.RecordFile{{File: "b.json", Data: badID}},
	}, validate.Options{})
	if !hasCode(issues, validation.CoreIDInvalid) {
		t.Fatalf("invalid id issues = %#v", issues)
	}
}

func TestCollectionV04LocalizedMismatch(t *testing.T) {
	manifest := schema.Manifest{
		Name: "x", Version: "1",
		Resources: []schema.Resource{{Record: schema.RecordType{
			ID: "post", Label: "Posts", Scope: schema.ScopeTenant,
			Fields: []schema.Field{
				{ID: "sku", Label: "SKU", Type: schema.FieldString},
				{ID: "blurb", Label: "Blurb", Type: schema.FieldString, Localized: true},
			},
		}}},
	}
	en := sampleEntry("post-1", "en", map[string]any{
		"metadata": map[string]any{
			"sku":   map[string]any{"value": "A"},
			"blurb": map[string]any{"value": "Hello"},
		},
	})
	ru := sampleEntry("post-1", "ru", map[string]any{
		"metadata": map[string]any{
			"sku":   map[string]any{"value": "B"},
			"blurb": map[string]any{"value": "Привет"},
		},
	})
	issues := validate.CollectionV04(validate.Collection{
		Entries: []validate.RecordFile{{File: "en.json", Data: en}, {File: "ru.json", Data: ru}},
	}, validate.Options{Manifest: &manifest})
	if !hasCode(issues, validation.EntryLocalizedMismatch) {
		t.Fatalf("expected localized_mismatch, got %#v", issues)
	}

	ru["metadata"].(map[string]any)["sku"] = map[string]any{"value": "A"}
	issues = validate.CollectionV04(validate.Collection{
		Entries: []validate.RecordFile{{File: "en.json", Data: en}, {File: "ru.json", Data: ru}},
	}, validate.Options{Manifest: &manifest})
	if hasCode(validation.ErrorsOnly(issues), validation.EntryLocalizedMismatch) {
		t.Fatalf("sku equal after canonicalization still mismatched: %#v", issues)
	}
}

func TestCollectionV04ParentIDNotSubjectToIDPattern(t *testing.T) {
	entry := sampleEntry("post-1", "en", map[string]any{"parent_id": "пост-parent"})
	issues := validate.CollectionV04(validate.Collection{
		Entries: []validate.RecordFile{{File: "a.json", Data: entry}},
	}, validate.Options{})
	if hasCode(issues, validation.CoreIDInvalid) {
		t.Fatalf("parent_id must not use the id grammar: %#v", issues)
	}
}

func hasCode(issues []validation.Issue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
