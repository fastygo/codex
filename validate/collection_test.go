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

func sampleSettings(locale string, extra map[string]any) map[string]any {
	o := map[string]any{
		"id": "site", "locale": locale, "title": "Studio", "language": locale,
		"version": jsonNumber(1), "created_at": "2026-10-25T11:09:00Z", "updated_at": "2026-10-25T11:09:00Z",
	}
	for k, v := range extra {
		o[k] = v
	}
	return o
}

func TestCollectionV04SettingsContact(t *testing.T) {
	full := sampleSettings("en", map[string]any{
		"contact": map[string]any{
			"phone": "+12025550143", "email": "hello@example.com", "address": "1 Example Street",
			"links": []any{map[string]any{"network": "telegram", "handle": "studio"}},
		},
		"hours": []any{map[string]any{"days": []any{"mon", "tue"}, "open": "09:00", "close": "18:00"}},
		"hours_exceptions": []any{
			map[string]any{"date": "2026-12-25", "closed": true},
			map[string]any{"date": "2026-12-24", "open": "09:00", "close": "13:00"},
		},
		"map": map[string]any{"provider": "osm", "lat": 38.8977, "lng": -77.0365},
	})
	issues := validate.CollectionV04(validate.Collection{
		Settings: []validate.RecordFile{{File: "s.json", Data: full}},
	}, validate.Options{})
	if len(issues) != 0 {
		t.Fatalf("full contact block issues = %#v", issues)
	}

	badPhone := sampleSettings("en", map[string]any{"contact": map[string]any{"phone": "+1 202 555 0143"}})
	issues = validate.CollectionV04(validate.Collection{
		Settings: []validate.RecordFile{{File: "s.json", Data: badPhone}},
	}, validate.Options{})
	if !hasCode(issues, validation.CoreSettingsPhoneInvalid) {
		t.Fatalf("phone issues = %#v", issues)
	}

	both := sampleSettings("en", map[string]any{
		"hours_exceptions": []any{map[string]any{"date": "2026-01-01", "closed": true, "open": "10:00", "close": "12:00"}},
	})
	issues = validate.CollectionV04(validate.Collection{
		Settings: []validate.RecordFile{{File: "s.json", Data: both}},
	}, validate.Options{})
	if !hasCode(issues, validation.CoreSettingsHoursExceptionInvalid) {
		t.Fatalf("union issues = %#v", issues)
	}

	dupDate := sampleSettings("en", map[string]any{
		"hours_exceptions": []any{
			map[string]any{"date": "2026-01-01", "closed": true},
			map[string]any{"date": "2026-01-01", "open": "10:00", "close": "12:00"},
		},
	})
	issues = validate.CollectionV04(validate.Collection{
		Settings: []validate.RecordFile{{File: "s.json", Data: dupDate}},
	}, validate.Options{})
	if !hasCode(issues, validation.CoreSettingsHoursExceptionDuplicated) {
		t.Fatalf("duplicated date issues = %#v", issues)
	}

	latOnly := sampleSettings("en", map[string]any{"map": map[string]any{"lat": 10.0}})
	issues = validate.CollectionV04(validate.Collection{
		Settings: []validate.RecordFile{{File: "s.json", Data: latOnly}},
	}, validate.Options{})
	if !hasCode(issues, validation.CoreSettingsMapInvalid) {
		t.Fatalf("lat without lng issues = %#v", issues)
	}

	unknown := sampleSettings("en", map[string]any{"contact": map[string]any{"fax": "+12"}})
	issues = validate.CollectionV04(validate.Collection{
		Settings: []validate.RecordFile{{File: "s.json", Data: unknown}},
	}, validate.Options{})
	if !hasCode(issues, validation.DecodeUnknownField) {
		t.Fatalf("unknown contact field issues = %#v", issues)
	}

	en := sampleSettings("en", map[string]any{
		"contact": map[string]any{"phone": "+12025550143", "address": "Springfield"},
		"map":     map[string]any{"provider": "osm"},
	})
	ru := sampleSettings("ru", map[string]any{
		"contact": map[string]any{"phone": "+12025550199", "address": "Москва"},
		"map":     map[string]any{"provider": "yandex"},
	})
	issues = validate.CollectionV04(validate.Collection{
		Settings: []validate.RecordFile{{File: "en.json", Data: en}, {File: "ru.json", Data: ru}},
	}, validate.Options{})
	if !hasCode(issues, validation.CoreSettingsLocaleNeutralMismatch) {
		t.Fatalf("locale-neutral issues = %#v", issues)
	}
	var warn int
	for _, issue := range issues {
		if issue.Code == validation.CoreSettingsLocaleNeutralMismatch {
			if issue.Severity != "warning" {
				t.Fatalf("locale-neutral severity = %q", issue.Severity)
			}
			warn++
		}
	}
	if warn != 2 {
		t.Fatalf("want 2 locale-neutral warnings, got %#v", issues)
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
