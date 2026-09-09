// Package conformance exposes stable fixtures for Codex adapter tests.
package conformance

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"reflect"

	"github.com/fastygo/codex/content"
	"github.com/fastygo/codex/revision"
	"github.com/fastygo/codex/schema"
	"github.com/fastygo/codex/taxonomy"
)

// Fixtures contains versioned protocol-neutral contract examples.
//
//go:embed fixtures/*.json
var Fixtures embed.FS

func ReadFixture(name string) ([]byte, error) {
	if path.Base(name) != name {
		return nil, fmt.Errorf("fixture name must be a base name")
	}
	return Fixtures.ReadFile("fixtures/" + name)
}

func EntryFixture(name string) (content.Entry, error) {
	data, err := ReadFixture(name)
	if err != nil {
		return content.Entry{}, err
	}
	var entry content.Entry
	if err := decodeStrict(data, &entry); err != nil {
		return content.Entry{}, err
	}
	if err := entry.Validate(); err != nil {
		return content.Entry{}, err
	}
	return entry, nil
}

func ManifestFixture(name string) (schema.Manifest, error) {
	data, err := ReadFixture(name)
	if err != nil {
		return schema.Manifest{}, err
	}
	var manifest schema.Manifest
	if err := decodeStrict(data, &manifest); err != nil {
		return schema.Manifest{}, err
	}
	if err := manifest.Validate(); err != nil {
		return schema.Manifest{}, err
	}
	return manifest, nil
}

func ResourceEntryFixture(manifestName, entryName string) (schema.Manifest, content.Entry, error) {
	manifest, err := ManifestFixture(manifestName)
	if err != nil {
		return schema.Manifest{}, content.Entry{}, err
	}
	entry, err := EntryFixture(entryName)
	if err != nil {
		return schema.Manifest{}, content.Entry{}, err
	}
	resource, exists := manifest.Resource(entry.Kind)
	if !exists {
		return schema.Manifest{}, content.Entry{}, fmt.Errorf("entry resource %q is not declared", entry.Kind)
	}
	if err := resource.ValidateEntry(entry); err != nil {
		return schema.Manifest{}, content.Entry{}, err
	}
	return manifest, entry, nil
}

type TaxonomyBundle struct {
	Definition taxonomy.Definition `json:"definition"`
	Terms      []taxonomy.Term     `json:"terms"`
}

func TaxonomyFixture(name string) (TaxonomyBundle, error) {
	data, err := ReadFixture(name)
	if err != nil {
		return TaxonomyBundle{}, err
	}
	var fixture TaxonomyBundle
	if err := decodeStrict(data, &fixture); err != nil {
		return TaxonomyBundle{}, err
	}
	if err := taxonomy.ValidateHierarchy(fixture.Definition, fixture.Terms); err != nil {
		return TaxonomyBundle{}, err
	}
	return fixture, nil
}

func RevisionFixture(name string) (revision.Revision, error) {
	data, err := ReadFixture(name)
	if err != nil {
		return revision.Revision{}, err
	}
	var value revision.Revision
	if err := decodeStrict(data, &value); err != nil {
		return revision.Revision{}, err
	}
	if err := value.Validate(); err != nil {
		return revision.Revision{}, err
	}
	return value, nil
}

func decodeStrict(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("fixture contains multiple JSON values")
		}
		return err
	}
	encoded, err := json.Marshal(target)
	if err != nil {
		return err
	}
	original, err := decodeDynamic(data)
	if err != nil {
		return err
	}
	roundTrip, err := decodeDynamic(encoded)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(original, roundTrip) {
		return errors.New("fixture contains unknown or unstable JSON fields")
	}
	return nil
}

func decodeDynamic(data []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	return value, nil
}
