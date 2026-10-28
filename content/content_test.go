package content_test

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/fastygo/codex/content"
	"github.com/fastygo/codex/validation"
)

func TestEntryValidateAndPublicProjection(t *testing.T) {
	now := testTime()
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
	projected.Slug = "changed"
	if entry.Slug == "changed" {
		t.Fatal("projection aliases slug")
	}
	projected.Metadata["summary"].Value.(map[string]any)["text"] = "changed"
	if entry.Metadata["summary"].Value.(map[string]any)["text"] == "changed" {
		t.Fatal("projection aliases public metadata map")
	}
}

func TestEntryValidationHasStableCodeAndPath(t *testing.T) {
	err := (content.Entry{}).Validate()
	if got := validation.Code(err); got != validation.CoreIDRequired {
		t.Fatalf("validation code = %q, error = %v", got, err)
	}
	if got := validation.Path(err); got != "id" {
		t.Fatalf("validation path = %q", got)
	}
}

func TestEntryValidateRejectsLifecycleViolations(t *testing.T) {
	now := testTime()
	tests := map[string]func(*content.Entry){
		"missing kind":   func(entry *content.Entry) { entry.Kind = "" },
		"missing slug":   func(entry *content.Entry) { entry.Slug = "" },
		"invalid locale": func(entry *content.Entry) { entry.Locale = "EN" },
		"scheduled without time": func(entry *content.Entry) {
			entry.Status = content.StatusScheduled
			entry.PublishedAt = nil
		},
		"trashed without deleted time": func(entry *content.Entry) {
			entry.Status = content.StatusTrashed
			entry.DeletedAt = nil
		},
		"self parent": func(entry *content.Entry) { entry.ParentID = entry.ID },
		"invalid id":  func(entry *content.Entry) { entry.ID = "пост-1" },
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

func TestStatusPendingIsValid(t *testing.T) {
	now := testTime()
	entry := validEntry(now)
	entry.Status = content.StatusPending
	if err := entry.Validate(); err != nil {
		t.Fatalf("pending status error = %v", err)
	}
}

func TestPickUsesDeterministicFallback(t *testing.T) {
	now := testTime()
	ru := validEntry(now)
	ru.Locale = "ru"
	ru.Title = "RU"
	de := validEntry(now)
	de.Locale = "de"
	de.Title = "DE"
	got, resolution, ok := content.Pick([]content.Entry{ru, de}, "post-1", "fr", "es")
	if !ok || got.Locale != "de" || !resolution.Fallback {
		t.Fatalf("Pick() = %#v %#v %v, want deterministic de fallback", got, resolution, ok)
	}
	got.Title = "changed"
	if ru.Title == "changed" || de.Title == "changed" {
		t.Fatal("Pick() aliases source entry")
	}
}

func TestPickExactAndLocales(t *testing.T) {
	now := testTime()
	en := validEntry(now)
	en.Locale = "en"
	ru := validEntry(now)
	ru.Locale = "ru"
	got, resolution, ok := content.Pick([]content.Entry{en, ru}, "post-1", "ru", "en")
	if !ok || got.Locale != "ru" || resolution.Fallback {
		t.Fatalf("Pick() = %#v %#v", got, resolution)
	}
	locales := content.Locales([]content.Entry{en, ru}, "post-1")
	if len(locales) != 2 || locales[0] != "en" || locales[1] != "ru" {
		t.Fatalf("Locales() = %#v", locales)
	}
}

func TestEntryValidateContainers(t *testing.T) {
	now := testTime()
	tests := map[string]func(*content.Entry){
		"empty metadata key": func(entry *content.Entry) {
			entry.Metadata = map[string]content.MetadataValue{" ": {Value: "bad"}}
		},
		"incomplete origin": func(entry *content.Entry) {
			entry.Origin = []content.Origin{{System: "wp"}}
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
	now := testTime()
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
	now := testTime()
	entry := validEntry(now)
	deletedAt := now.Add(time.Hour)
	entry.DeletedAt = &deletedAt
	zero := int64(0)
	pinned := false
	entry.Order = &zero
	entry.Pinned = &pinned
	entry.Origin = []content.Origin{{System: "wp", ID: "1"}}
	entry.Discussion = &content.Discussion{Comments: true, Pings: false}
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
	*cloned.Order = 4
	*cloned.Pinned = true
	cloned.Discussion.Comments = false
	cloned.Origin[0].ID = "changed"
	if entry.PublishedAt.Equal(*cloned.PublishedAt) || entry.DeletedAt.Equal(*cloned.DeletedAt) {
		t.Fatal("Clone() aliases timestamp pointers")
	}
	if *entry.Order != 0 || *entry.Pinned || !entry.Discussion.Comments || entry.Origin[0].ID != "1" {
		t.Fatal("Clone() aliases order, pinned, discussion, or origin")
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
	now := testTime()
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
	now := testTime()
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

func TestRecordIDGrammar(t *testing.T) {
	for _, value := range []string{"post-1", "telegram:group:42:1001", "A", "a.b_c:d-e"} {
		if !content.ValidRecordID(value) {
			t.Fatalf("ValidRecordID(%q) = false", value)
		}
	}
	for _, value := range []string{"", "пост-1", "-post", ".post", ":post", " post", "post "} {
		if content.ValidRecordID(value) {
			t.Fatalf("ValidRecordID(%q) = true", value)
		}
	}
	if got, want := content.IDFromKindInt(content.KindPost, 42), "post-42"; got != want {
		t.Fatalf("IDFromKindInt() = %q, want %q", got, want)
	}
}

func TestValidCurrency(t *testing.T) {
	if !content.ValidCurrency("USD") || content.ValidCurrency("usd") || content.ValidCurrency("US") {
		t.Fatal("ValidCurrency() grammar")
	}
}

func TestNormalizeSlug(t *testing.T) {
	if got, want := content.NormalizeSlug("  Привет, Codex 2026!  "), "привет-codex-2026"; got != want {
		t.Fatalf("NormalizeSlug() = %q, want %q", got, want)
	}
}

func TestCommentMenuDictionarySettings(t *testing.T) {
	now := testTime()
	comment := content.Comment{
		ID: "c-1", EntryID: "post-1", Content: "Hi", Status: content.CommentApproved,
		Version: 1, CreatedAt: now, UpdatedAt: now,
	}
	if err := comment.Validate(); err != nil {
		t.Fatalf("Comment.Validate() error = %v", err)
	}
	menu := content.Menu{
		ID: "m-1", Location: "header", Locale: "en",
		Items: []content.MenuItem{{ID: "i-1", Label: "Home", URL: "/"}},
	}
	if err := menu.Validate(); err != nil {
		t.Fatalf("Menu.Validate() error = %v", err)
	}
	dict := content.Dictionary{Locale: "en", Messages: map[string]string{"ok": "OK"}}
	if err := dict.Validate(); err != nil {
		t.Fatalf("Dictionary.Validate() error = %v", err)
	}
	settings := content.Settings{
		ID: "site", Locale: "en", Title: "Site", Language: "en-US", Currency: "USD",
		Version: 1, CreatedAt: now, UpdatedAt: now,
	}
	if err := settings.Validate(); err != nil {
		t.Fatalf("Settings.Validate() error = %v", err)
	}
	closed := true
	lat, lng := 55.7558, 37.6173
	settings.Contact = &content.Contact{
		Phone: "+12025550143", Email: "hello@example.com", Address: "Moscow",
		Links: []content.ContactLink{{Network: "telegram", Handle: "studio"}},
	}
	settings.Hours = []content.HoursRule{{Days: []string{content.WeekdayMon}, Open: "09:00", Close: "18:00"}}
	settings.HoursExceptions = []content.HoursException{{Date: "2026-12-25", Closed: &closed}}
	settings.Map = &content.MapPin{Provider: "osm", Lat: &lat, Lng: &lng}
	if err := settings.Validate(); err != nil {
		t.Fatalf("Settings.Validate() contact block error = %v", err)
	}
	settings.Contact.Phone = "+1 202"
	if err := settings.Validate(); err == nil || validation.Code(err) != validation.CoreSettingsPhoneInvalid {
		t.Fatalf("invalid phone = %v", err)
	}
}

func TestSiteLocale(t *testing.T) {
	if got := content.SiteLocale(nil); got != content.DefaultLocale {
		t.Fatalf("no settings = %q, want %q", got, content.DefaultLocale)
	}
	now := testTime()
	settings := []content.Settings{
		{ID: "other", Locale: "de", Title: "X", Language: "de", Version: 1, CreatedAt: now, UpdatedAt: now},
		{ID: "site", Locale: "ru", Title: "S", Language: "ru", Version: 1, CreatedAt: now, UpdatedAt: now},
		{ID: "site", Locale: "en", Title: "S", Language: "en-US", Version: 1, CreatedAt: now, UpdatedAt: now},
	}
	if got := content.SiteLocale(settings); got != "en-US" {
		t.Fatalf("SiteLocale() = %q, want en-US (site, locale-sorted)", got)
	}
	var typed content.SettingsV04 = settings[0]
	if typed.Language != "de" {
		t.Fatalf("SettingsV04 alias = %#v", typed)
	}
}

func validEntry(now time.Time) content.Entry {
	published := now
	return content.Entry{
		ID:          "post-1",
		Locale:      "en",
		Kind:        content.KindPost,
		Status:      content.StatusPublished,
		Visibility:  content.VisibilityPublic,
		Slug:        "hello",
		Title:       "Hello",
		Content:     "World",
		Version:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
		PublishedAt: &published,
	}
}

func testTime() time.Time {
	return time.Date(2026, time.September, 9, 12, 0, 0, 0, time.UTC)
}
