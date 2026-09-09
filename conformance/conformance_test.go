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
	"github.com/fastygo/codex/validation"
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
	if !ok || resource.Record.ID != "article" {
		t.Fatalf("article resource = %#v, %v", resource, ok)
	}
}

func TestResourceEntryFixture(t *testing.T) {
	manifest, entry, err := conformance.ResourceEntryFixture(
		"example.manifest.json",
		"article.entry.json",
	)
	if err != nil {
		t.Fatalf("ResourceEntryFixture() error = %v", err)
	}
	if entry.ID != "article-1001" {
		t.Fatalf("entry id = %q", entry.ID)
	}
	resource, _ := manifest.Resource(entry.Kind)
	projected, err := resource.PublicProjection(entry)
	if err != nil {
		t.Fatalf("PublicProjection() error = %v", err)
	}
	if _, exists := projected.Metadata["operator_note"]; exists {
		t.Fatal("schema-sensitive metadata survived public projection")
	}
}

func TestResourceEntryFixtureFailureCode(t *testing.T) {
	_, _, err := conformance.ResourceEntryFixture(
		"example.manifest.json",
		"article.entry.invalid-location.json",
	)
	if got := validation.Code(err); got != "schema.entry.field_location" {
		t.Fatalf("validation code = %q, error = %v", got, err)
	}
	if got := validation.Path(err); got != "en.source_id" {
		t.Fatalf("validation path = %q", got)
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

func TestManifestDigestGolden(t *testing.T) {
	manifest, err := conformance.ManifestFixture("example.manifest.json")
	if err != nil {
		t.Fatalf("ManifestFixture() error = %v", err)
	}
	digest, err := manifest.Digest()
	if err != nil {
		t.Fatalf("Digest() error = %v", err)
	}
	fixtureData, err := conformance.ReadFixture("example.manifest.digest.json")
	if err != nil {
		t.Fatalf("ReadFixture() error = %v", err)
	}
	var fixture struct {
		Manifest string `json:"manifest"`
		Digest   string `json:"digest"`
	}
	if err := json.Unmarshal(fixtureData, &fixture); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if fixture.Manifest != "example.manifest.json" || digest != fixture.Digest {
		t.Fatalf("manifest digest = %q", digest)
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
		{name: "article.entry.invalid-location.json", target: &content.Entry{}},
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
