package content_test

import (
	"encoding/json"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/fastygo/codex/content"
)

func TestEntryValidateAndPublicProjection(t *testing.T) {
	now := time.Date(2026, time.September, 9, 12, 0, 0, 0, time.UTC)
	entry := validEntry(now)
	entry.Metadata = map[string]content.MetadataValue{
		"summary": {Value: map[string]any{"text": "visible"}},
		"token":   {Value: map[string]any{"value": "secret"}, Private: true},
	}

	if err := entry.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if !entry.IsPublicAt(now.Add(time.Minute)) {
		t.Fatal("published public entry is not public")
	}

	projected := entry.PublicProjection()
	if _, exists := projected.Metadata["token"]; exists {
		t.Fatal("private metadata survived projection")
	}
	projected.Slug["en"] = "changed"
	if entry.Slug["en"] == "changed" {
		t.Fatal("projection aliases localized text")
	}
	projected.Metadata["summary"].Value.(map[string]any)["text"] = "changed"
	if entry.Metadata["summary"].Value.(map[string]any)["text"] == "changed" {
		t.Fatal("projection aliases public metadata map")
	}
}

func TestEntryValidateRejectsLifecycleViolations(t *testing.T) {
	now := time.Date(2026, time.September, 9, 12, 0, 0, 0, time.UTC)
	tests := map[string]func(*content.Entry){
		"missing kind": func(entry *content.Entry) { entry.Kind = "" },
		"missing title": func(entry *content.Entry) {
			entry.Title = nil
		},
		"scheduled without time": func(entry *content.Entry) {
			entry.Status = content.StatusScheduled
			entry.PublishedAt = nil
		},
		"trashed without deleted time": func(entry *content.Entry) {
			entry.Status = content.StatusTrashed
			entry.DeletedAt = nil
		},
		"self parent": func(entry *content.Entry) { entry.ParentID = entry.ID },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			entry := validEntry(now)
			mutate(&entry)
			if err := entry.Validate(); err == nil {
				t.Fatal("Validate() error = nil")
			}
		})
	}
}

func TestResolveLocaleUsesDeterministicFallback(t *testing.T) {
	entry := content.Entry{Locales: map[string]content.LocaleDocument{
		"ru": {Data: map[string]any{"title": "RU"}, Status: content.StatusPublished},
		"de": {Data: map[string]any{"title": "DE"}, Status: content.StatusDraft},
	}}

	resolved := entry.ResolveLocale("fr", "es")
	if resolved.Served != "de" || !resolved.Fallback {
		t.Fatalf("ResolveLocale() = %#v, want deterministic de fallback", resolved)
	}
	resolved.Data["title"] = "changed"
	if reflect.DeepEqual(resolved.Data, entry.Locales["de"].Data) {
		t.Fatal("resolved data aliases entry locale data")
	}
}

func TestResolveLocaleAndMergeLocales(t *testing.T) {
	entry := content.Entry{Locales: map[string]content.LocaleDocument{
		"en": {
			Data:   map[string]any{"nested": map[string]any{"title": "EN"}},
			Status: content.StatusPublished,
		},
	}}
	resolved := entry.ResolveLocale("en", "")
	if resolved.Served != "en" || resolved.Fallback {
		t.Fatalf("ResolveLocale() = %#v", resolved)
	}
	nested := resolved.Data["nested"].(map[string]any)
	nested["title"] = "changed"
	original := entry.Locales["en"].Data["nested"].(map[string]any)
	if original["title"] == "changed" {
		t.Fatal("ResolveLocale() aliases nested data")
	}

	merged, err := content.MergeLocales(nil, map[string]content.LocaleDocument{
		" RU ": {Data: map[string]any{"title": "RU"}, Status: content.StatusDraft},
	})
	if err != nil {
		t.Fatalf("MergeLocales() error = %v", err)
	}
	if _, exists := merged["ru"]; !exists {
		t.Fatal("MergeLocales() did not normalize locale")
	}
	if index := (content.Entry{Locales: merged}).LocaleIndex(); len(index) != 1 || index[0].Locale != "ru" {
		t.Fatalf("LocaleIndex() = %#v", index)
	}
}

func TestMergeLocalesRejectsNormalizedCollision(t *testing.T) {
	target := map[string]content.LocaleDocument{
		"de": {Data: map[string]any{"title": "DE"}, Status: content.StatusDraft},
	}
	_, err := content.MergeLocales(target, map[string]content.LocaleDocument{
		" EN ": {Data: map[string]any{"title": "first"}, Status: content.StatusDraft},
		"en":   {Data: map[string]any{"title": "second"}, Status: content.StatusDraft},
	})
	if err == nil {
		t.Fatal("MergeLocales() error = nil")
	}
	if len(target) != 1 {
		t.Fatal("MergeLocales() mutated target before rejecting collision")
	}
}

func TestEntryValidateContainers(t *testing.T) {
	now := time.Date(2026, time.September, 9, 12, 0, 0, 0, time.UTC)
	tests := map[string]func(*content.Entry){
		"empty metadata key": func(entry *content.Entry) {
			entry.Metadata = map[string]content.MetadataValue{" ": {Value: "bad"}}
		},
		"invalid locale status": func(entry *content.Entry) {
			entry.Locales = map[string]content.LocaleDocument{"en": {Data: map[string]any{}, Status: "unknown"}}
		},
		"noncanonical locale": func(entry *content.Entry) {
			entry.Locales = map[string]content.LocaleDocument{" EN ": {Data: map[string]any{}, Status: content.StatusDraft}}
		},
		"noncanonical localized text": func(entry *content.Entry) {
			entry.Title = content.LocalizedText{" EN ": "Title"}
		},
		"incomplete term": func(entry *content.Entry) {
			entry.Terms = []content.TermRef{{Taxonomy: "topic"}}
		},
		"duplicate term": func(entry *content.Entry) {
			entry.Terms = []content.TermRef{
				{Taxonomy: "topic", TermID: "go"},
				{Taxonomy: "topic", TermID: "go"},
			}
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			entry := validEntry(now)
			mutate(&entry)
			if err := entry.Validate(); err == nil {
				t.Fatal("Validate() error = nil")
			}
		})
	}
}

func TestEntryPublicLifecycle(t *testing.T) {
	now := time.Date(2026, time.September, 9, 12, 0, 0, 0, time.UTC)
	entry := validEntry(now)
	future := now.Add(time.Hour)
	entry.PublishedAt = &future
	if entry.IsPublicAt(now) {
		t.Fatal("future publication is public")
	}
	entry.Visibility = content.VisibilityPrivate
	if entry.IsPublicAt(now.Add(2 * time.Hour)) {
		t.Fatal("private entry is public")
	}
}

func TestCloneDoesNotAliasJSONContainersOrTimes(t *testing.T) {
	now := time.Date(2026, time.September, 9, 12, 0, 0, 0, time.UTC)
	entry := validEntry(now)
	deletedAt := now.Add(time.Hour)
	entry.DeletedAt = &deletedAt
	entry.Metadata = map[string]content.MetadataValue{
		"nested": {Value: map[string][]int{"values": {1, 2}}},
	}
	cloned := entry.Clone()
	values := cloned.Metadata["nested"].Value.(map[string][]int)
	values["values"][0] = 9
	if original := entry.Metadata["nested"].Value.(map[string][]int)["values"][0]; original != 1 {
		t.Fatal("Clone() aliases typed JSON containers")
	}
	*cloned.PublishedAt = now.Add(2 * time.Hour)
	*cloned.DeletedAt = now.Add(3 * time.Hour)
	if entry.PublishedAt.Equal(*cloned.PublishedAt) || entry.DeletedAt.Equal(*cloned.DeletedAt) {
		t.Fatal("Clone() aliases timestamp pointers")
	}
}

func TestValidateJSONValueRejectsNonNativeValues(t *testing.T) {
	tests := []any{
		func() {},
		math.NaN(),
		map[int]string{1: "one"},
		struct{ Value string }{Value: "one"},
	}
	for _, value := range tests {
		if err := content.ValidateJSONValue(value); err == nil {
			t.Fatalf("ValidateJSONValue(%T) error = nil", value)
		}
	}
}

func TestCloneHandlesInvalidCyclicMetadataWithoutRecursing(t *testing.T) {
	now := time.Date(2026, time.September, 9, 12, 0, 0, 0, time.UTC)
	cyclic := map[string]any{}
	cyclic["self"] = cyclic
	entry := validEntry(now)
	entry.Metadata = map[string]content.MetadataValue{"cyclic": {Value: cyclic}}

	cloned := entry.Clone()
	clonedMap := cloned.Metadata["cyclic"].Value.(map[string]any)
	nested := clonedMap["self"].(map[string]any)
	nested["marker"] = true
	if _, exists := cyclic["marker"]; exists {
		t.Fatal("Clone() aliases cyclic metadata")
	}
}

func TestClonePreservesOverlappingSliceShapes(t *testing.T) {
	now := time.Date(2026, time.September, 9, 12, 0, 0, 0, time.UTC)
	for range 100 {
		base := []int{1, 2, 3}
		entry := validEntry(now)
		entry.Metadata = map[string]content.MetadataValue{
			"views": {Value: map[string]any{
				"short": base[:1],
				"long":  base[:3],
			}},
		}
		cloned := entry.Clone()
		views := cloned.Metadata["views"].Value.(map[string]any)
		if len(views["short"].([]int)) != 1 || len(views["long"].([]int)) != 3 {
			t.Fatalf("Clone() changed overlapping slice shapes: %#v", views)
		}
		views["long"].([]int)[1] = 9
		if base[1] != 2 {
			t.Fatal("Clone() aliases overlapping source slice")
		}
	}
}

func TestLocaleUpdatedAtOmitsZero(t *testing.T) {
	encoded, err := json.Marshal(content.LocaleDocument{
		Data:   map[string]any{},
		Status: content.StatusDraft,
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(encoded) != `{"data":{},"status":"draft"}` {
		t.Fatalf("Marshal() = %s", encoded)
	}
}

func TestDynamicJSONUnmarshalIsLosslessAndForwardCompatible(t *testing.T) {
	var metadata content.MetadataValue
	if err := json.Unmarshal([]byte(`{"value":9007199254740993,"future":true}`), &metadata); err != nil {
		t.Fatalf("MetadataValue Unmarshal() error = %v", err)
	}
	if got := metadata.Value.(json.Number).String(); got != "9007199254740993" {
		t.Fatalf("metadata number = %s", got)
	}
	if err := json.Unmarshal([]byte(`{"private":true}`), &metadata); err == nil {
		t.Fatal("MetadataValue without value error = nil")
	}

	var document content.LocaleDocument
	if err := json.Unmarshal([]byte(`{"data":{"sequence":9007199254740993},"status":"draft","future":true}`), &document); err != nil {
		t.Fatalf("LocaleDocument Unmarshal() error = %v", err)
	}
	if got := document.Data["sequence"].(json.Number).String(); got != "9007199254740993" {
		t.Fatalf("locale number = %s", got)
	}
	if err := json.Unmarshal([]byte(`{"status":"draft"}`), &document); err == nil {
		t.Fatal("LocaleDocument without data error = nil")
	}
}

func TestIdentifierGrammar(t *testing.T) {
	for _, value := range []string{"message", "message_type", "message-2", "a"} {
		if !content.ValidIdentifier(value) {
			t.Fatalf("ValidIdentifier(%q) = false", value)
		}
	}
	for _, value := range []string{"", " Message", "message ", "Message", "сообщение", "1message"} {
		if content.ValidIdentifier(value) {
			t.Fatalf("ValidIdentifier(%q) = true", value)
		}
	}
}

func TestNormalizeSlug(t *testing.T) {
	if got, want := content.NormalizeSlug("  Привет, Codex 2026!  "), "привет-codex-2026"; got != want {
		t.Fatalf("NormalizeSlug() = %q, want %q", got, want)
	}
}

func validEntry(now time.Time) content.Entry {
	return content.Entry{
		ID:         "post-1",
		Kind:       content.KindPost,
		Status:     content.StatusPublished,
		Visibility: content.VisibilityPublic,
		Slug:       content.LocalizedText{"en": "hello"},
		Title:      content.LocalizedText{"en": "Hello"},
		Content:    content.LocalizedText{"en": "World"},
		Version:    1,
		CreatedAt:  now,
		UpdatedAt:  now,
		PublishedAt: func() *time.Time {
			value := now
			return &value
		}(),
	}
}
