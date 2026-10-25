package canonical_test

import (
	"strings"
	"testing"

	"github.com/fastygo/codex/canonical"
)

const termBase = `{"id":"term-1","locale":"en","taxonomy_id":"category","name":"News","slug":"news","version":1,"created_at":"2026-10-25T11:09:00+03:00","updated_at":"2026-10-25T11:09:00.000Z"}`

func TestCanonicalizeTermDropsCountAndOmitsEmptyOptionals(t *testing.T) {
	withCount := `{"count":42,"id":"term-1","locale":"en","taxonomy_id":"category","name":"News","slug":"news","description":"","parent_id":"","origin":[],"metadata":{},"version":1,"created_at":"2026-10-25T08:09:00Z","updated_at":"2026-10-25T08:09:00Z"}`
	out, err := canonical.CanonicalizeTerm(withCount)
	if err != nil {
		t.Fatalf("CanonicalizeTerm() error = %v", err)
	}
	if strings.Contains(out, `"count"`) {
		t.Fatalf("count must not appear in canonical form: %s", out)
	}
	if strings.Contains(out, `"description"`) || strings.Contains(out, `"parent_id"`) || strings.Contains(out, `"origin"`) || strings.Contains(out, `"metadata"`) {
		t.Fatalf("empty top-level optionals must be omitted: %s", out)
	}
	want := `{"created_at":"2026-10-25T08:09:00Z","id":"term-1","locale":"en","name":"News","slug":"news","taxonomy_id":"category","updated_at":"2026-10-25T08:09:00Z","version":1}`
	if out != want {
		t.Fatalf("got %s\nwant %s", out, want)
	}
	plain, err := canonical.CanonicalizeTerm(termBase)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(plain, `"created_at":"2026-10-25T08:09:00Z"`) {
		t.Fatalf("offset date must convert to Z: %s", plain)
	}
	if !strings.Contains(plain, `"updated_at":"2026-10-25T11:09:00Z"`) {
		t.Fatalf(".000 fractional zeros must be dropped: %s", plain)
	}
	digest, err := canonical.TermDigest(withCount)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(digest, canonical.TermDigestPrefix) {
		t.Fatalf("TermDigest prefix = %q", digest)
	}
	if len(digest) != len(canonical.TermDigestPrefix)+64 {
		t.Fatalf("TermDigest length = %d", len(digest))
	}
}

func TestCanonicalizeTermCountUnsafeIntegerIsParserError(t *testing.T) {
	_, err := canonical.CanonicalizeTerm(`{"count":9007199254740993,"id":"cat-news","locale":"en","taxonomy_id":"category","name":"News","slug":"news","version":1,"created_at":"2026-10-25T11:09:00Z","updated_at":"2026-10-25T11:09:00Z"}`)
	if canonical.Code(err) != canonical.ErrUnsafeInteger {
		t.Fatalf("unsafe count code = %q err = %v", canonical.Code(err), err)
	}
}

func TestCanonicalizeTermRejectsUnknownAndUnsafe(t *testing.T) {
	_, err := canonical.CanonicalizeTerm(`{"id":"term-1","locale":"en","taxonomy_id":"category","name":"News","slug":"news","version":1,"created_at":"2026-10-25T08:09:00Z","updated_at":"2026-10-25T08:09:00Z","extra":1}`)
	if canonical.Code(err) != canonical.ErrUnknownField {
		t.Fatalf("unknown field code = %q err = %v", canonical.Code(err), err)
	}
	_, err = canonical.CanonicalizeTerm(`{"id":"term-1","locale":"en","taxonomy_id":"category","name":"News","slug":"news","version":9007199254740993,"created_at":"2026-10-25T08:09:00Z","updated_at":"2026-10-25T08:09:00Z"}`)
	if canonical.Code(err) != canonical.ErrUnsafeInteger {
		t.Fatalf("unsafe integer code = %q err = %v", canonical.Code(err), err)
	}
	_, err = canonical.CanonicalizeTerm(`{"id":"term-1","locale":"en","taxonomy_id":"category","name":"News","slug":"news","version":1,"created_at":"2026-10-25T08:09:00.001Z","updated_at":"2026-10-25T08:09:00Z"}`)
	if canonical.Code(err) != canonical.ErrFractionalSeconds {
		t.Fatalf("fractional seconds code = %q err = %v", canonical.Code(err), err)
	}
}

func TestCanonicalizeTermKeepsNestedEmptyMetadataValue(t *testing.T) {
	input := `{"id":"term-1","locale":"en","taxonomy_id":"category","name":"News","slug":"news","version":1,"created_at":"2026-10-25T08:09:00Z","updated_at":"2026-10-25T08:09:00Z","metadata":{"note":{"value":"","private":false},"secret":{"value":"x","private":true}}}`
	out, err := canonical.CanonicalizeTerm(input)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"metadata":{"note":{"value":""}}`) {
		t.Fatalf("nested empty metadata value must be kept, private dropped: %s", out)
	}
	if strings.Contains(out, "secret") {
		t.Fatalf("private metadata must be dropped: %s", out)
	}
}

func TestCanonicalizeSettingsPrefixAndEmptyOptionals(t *testing.T) {
	input := `{"id":"site","locale":"en","title":"Shop","language":"en-US","description":"","comments_open":false,"version":1,"created_at":"2026-10-25T11:09:00+03:00","updated_at":"2026-10-25T08:09:00Z"}`
	out, err := canonical.CanonicalizeSettings(input)
	if err != nil {
		t.Fatalf("CanonicalizeSettings() error = %v", err)
	}
	if strings.Contains(out, `"description"`) {
		t.Fatalf("empty top-level description must be omitted: %s", out)
	}
	if !strings.Contains(out, `"comments_open":false`) {
		t.Fatalf("comments_open false must be kept: %s", out)
	}
	if !strings.Contains(out, `"created_at":"2026-10-25T08:09:00Z"`) {
		t.Fatalf("dates must be ...Z to seconds: %s", out)
	}
	digest, err := canonical.SettingsDigest(input)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(digest, canonical.SettingsDigestPrefix) {
		t.Fatalf("SettingsDigest prefix = %q", digest)
	}
	if len(digest) != len(canonical.SettingsDigestPrefix)+64 {
		t.Fatalf("SettingsDigest length = %d, digest = %q", len(digest), digest)
	}
}

func TestJCSDoesNotHTMLEscape(t *testing.T) {
	got := canonical.JCS(map[string]any{
		"error": "a < b & c > d",
		"line":  "\u2028sep\u2029",
	})
	if strings.Contains(got, `\u003c`) || strings.Contains(got, `\u0026`) || strings.Contains(got, `\u003e`) {
		t.Fatalf("JCS HTML-escaped: %s", got)
	}
	if strings.Contains(got, `\u2028`) || strings.Contains(got, `\u2029`) {
		t.Fatalf("JCS \\u-escaped line separators: %s", got)
	}
	if !strings.Contains(got, `a < b & c > d`) {
		t.Fatalf("JCS missing raw characters: %s", got)
	}
	text, err := canonical.CanonicalJSON(`{"b":1,"a":2}`)
	if err != nil {
		t.Fatal(err)
	}
	if text != `{"a":2,"b":1}` {
		t.Fatalf("CanonicalJSON = %s", text)
	}
}
