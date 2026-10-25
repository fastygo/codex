package migrate_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fastygo/codex/canonical"
	"github.com/fastygo/codex/migrate"
)

func TestEntryJSONMigratesV03Article(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "article.entry.v03.json"))
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	entry, err := migrate.EntryJSON(data, "en")
	if err != nil {
		t.Fatalf("EntryJSON() error = %v", err)
	}
	if entry.Locale != "en" || entry.Slug != "hello-from-codex" || entry.Title != "Hello from Codex" {
		t.Fatalf("chrome = %#v", entry)
	}
	if entry.Content != "One entry contract, many adapters." {
		t.Fatalf("content = %q", entry.Content)
	}
	seq, ok := entry.Metadata["source_sequence"].Value.(interface{ String() string })
	if !ok || seq.String() != "9007199254740991" {
		t.Fatalf("source_sequence = %#v", entry.Metadata["source_sequence"].Value)
	}
}

func TestEntryJSONRejectsUnsafeInteger(t *testing.T) {
	data := []byte(`{
		"id":"post-1","kind":"post","status":"published","visibility":"public",
		"slug":{"en":"hello"},"title":{"en":"Hello"},
		"metadata":{"n":{"value":9007199254740993}},
		"version":1,"created_at":"2026-09-09T12:00:00Z","updated_at":"2026-09-09T12:00:00Z"
	}`)
	_, err := migrate.EntryJSON(data, "en")
	if canonical.Code(err) != canonical.ErrUnsafeInteger {
		t.Fatalf("error = %v", err)
	}
}

func TestTaxonomyJSONMigratesV03Bundle(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "topic.taxonomy.v03.json"))
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	def, terms, err := migrate.TaxonomyJSON(data, "en")
	if err != nil {
		t.Fatalf("TaxonomyJSON() error = %v", err)
	}
	if def.Label != "Topic" || def.Locale != "en" {
		t.Fatalf("definition = %#v", def)
	}
	if len(terms) != 2 || terms[1].Name != "Architecture" || terms[1].ParentID != "engineering" {
		t.Fatalf("terms = %#v", terms)
	}
}
