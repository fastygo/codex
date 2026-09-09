package conformance_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/fastygo/codex/conformance"
	"github.com/fastygo/codex/content"
	"github.com/fastygo/codex/revision"
	"github.com/fastygo/codex/schema"
)

func TestEntryFixture(t *testing.T) {
	entry, err := conformance.EntryFixture("article.entry.json")
	if err != nil {
		t.Fatalf("EntryFixture() error = %v", err)
	}
	if entry.Kind != "article" || entry.Visibility != content.VisibilityPublic {
		t.Fatalf("unexpected entry = %#v", entry)
	}
	sequence, ok := entry.Metadata["source_sequence"].Value.(json.Number)
	if !ok || sequence.String() != "9007199254740993" {
		t.Fatalf("source sequence = %#v, want lossless json.Number", entry.Metadata["source_sequence"].Value)
	}
}

func TestManifestFixture(t *testing.T) {
	manifest, err := conformance.ManifestFixture("example.manifest.json")
	if err != nil {
		t.Fatalf("ManifestFixture() error = %v", err)
	}
	resource, ok := manifest.Resource("article")
	if !ok || resource.Collection != "articles" {
		t.Fatalf("article resource = %#v, %v", resource, ok)
	}
}

func TestManifestCanonicalGolden(t *testing.T) {
	manifest, err := conformance.ManifestFixture("example.manifest.json")
	if err != nil {
		t.Fatalf("ManifestFixture() error = %v", err)
	}
	encoded, err := json.Marshal(manifest.Canonical())
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	golden, err := conformance.ReadFixture("example.manifest.canonical.json")
	if err != nil {
		t.Fatalf("ReadFixture() error = %v", err)
	}
	if !reflect.DeepEqual(decodeSemantic(t, encoded), decodeSemantic(t, golden)) {
		t.Fatalf("canonical manifest changed:\n got %s\nwant %s", encoded, golden)
	}
}

func TestTaxonomyAndRevisionFixtures(t *testing.T) {
	taxonomyFixture, err := conformance.TaxonomyFixture("topic.taxonomy.json")
	if err != nil {
		t.Fatalf("TaxonomyFixture() error = %v", err)
	}
	if len(taxonomyFixture.Terms) != 2 {
		t.Fatalf("term count = %d, want 2", len(taxonomyFixture.Terms))
	}
	revisionFixture, err := conformance.RevisionFixture("article.revision.json")
	if err != nil {
		t.Fatalf("RevisionFixture() error = %v", err)
	}
	if revisionFixture.EntryID != "article-1001" {
		t.Fatalf("revision entry = %q", revisionFixture.EntryID)
	}
}

func TestFixtureJSONRoundTrips(t *testing.T) {
	tests := []struct {
		name   string
		target any
	}{
		{name: "article.entry.json", target: &content.Entry{}},
		{name: "example.manifest.json", target: &schema.Manifest{}},
		{name: "example.manifest.canonical.json", target: &schema.Manifest{}},
		{name: "topic.taxonomy.json", target: &conformance.TaxonomyBundle{}},
		{name: "article.revision.json", target: &revision.Revision{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data, err := conformance.ReadFixture(test.name)
			if err != nil {
				t.Fatalf("ReadFixture() error = %v", err)
			}
			if err := json.Unmarshal(data, test.target); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}
			encoded, err := json.Marshal(test.target)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			originalJSON := decodeSemantic(t, data)
			emittedJSON := decodeSemantic(t, encoded)
			if !reflect.DeepEqual(originalJSON, emittedJSON) {
				t.Fatalf("fixture JSON contract changed:\n got %s\nwant %s", encoded, data)
			}
			second := reflect.New(reflect.TypeOf(test.target).Elem()).Interface()
			if err := json.Unmarshal(encoded, second); err != nil {
				t.Fatalf("second Unmarshal() error = %v", err)
			}
			if !reflect.DeepEqual(test.target, second) {
				t.Fatal("JSON round trip changed contract value")
			}
		})
	}
}

func decodeSemantic(t *testing.T, data []byte) any {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		t.Fatalf("semantic JSON decode error = %v", err)
	}
	return value
}

func TestReadFixtureRejectsTraversal(t *testing.T) {
	if _, err := conformance.ReadFixture("../article.entry.json"); err == nil {
		t.Fatal("ReadFixture() error = nil")
	}
}
