package taxonomy_test

import (
	"testing"

	"github.com/fastygo/codex/content"
	"github.com/fastygo/codex/taxonomy"
)

func TestValidateHierarchy(t *testing.T) {
	definition := taxonomy.Definition{
		ID:              "topic",
		Label:           content.LocalizedText{"en": "Topic"},
		Mode:            taxonomy.ModeHierarchical,
		AssignedToKinds: []content.Kind{"message"},
		Version:         1,
	}
	terms := []taxonomy.Term{
		{ID: "go", TaxonomyID: "topic", Name: content.LocalizedText{"en": "Go"}, Slug: content.LocalizedText{"en": "go"}, Version: 1},
		{ID: "wails", TaxonomyID: "topic", Name: content.LocalizedText{"en": "Wails"}, Slug: content.LocalizedText{"en": "wails"}, ParentID: "go", Version: 1},
	}
	if err := taxonomy.ValidateHierarchy(definition, terms); err != nil {
		t.Fatalf("ValidateHierarchy() error = %v", err)
	}
	assignment := taxonomy.Assignment{
		ResourceKind: "message",
		ResourceID:   "telegram:group:42:1001",
		TaxonomyID:   "topic",
		TermID:       "wails",
	}
	if err := assignment.Validate(definition, terms[1]); err != nil {
		t.Fatalf("Assignment.Validate() error = %v", err)
	}
}

func TestValidateHierarchyRejectsCycle(t *testing.T) {
	definition := taxonomy.Definition{
		ID:              "topic",
		Label:           content.LocalizedText{"en": "Topic"},
		Mode:            taxonomy.ModeHierarchical,
		AssignedToKinds: []content.Kind{content.KindPost},
		Version:         1,
	}
	terms := []taxonomy.Term{
		{ID: "a", TaxonomyID: "topic", Name: content.LocalizedText{"en": "A"}, Slug: content.LocalizedText{"en": "a"}, ParentID: "b", Version: 1},
		{ID: "b", TaxonomyID: "topic", Name: content.LocalizedText{"en": "B"}, Slug: content.LocalizedText{"en": "b"}, ParentID: "a", Version: 1},
	}
	if err := taxonomy.ValidateHierarchy(definition, terms); err == nil {
		t.Fatal("ValidateHierarchy() error = nil")
	}
}

func TestDefinitionCanonicalDoesNotMutate(t *testing.T) {
	definition := taxonomy.Definition{
		ID:              "topic",
		Label:           content.LocalizedText{"en": "Topic"},
		Mode:            taxonomy.ModeFlat,
		AssignedToKinds: []content.Kind{"message", "conversation"},
		Version:         1,
	}
	canonical := definition.Canonical()
	if canonical.AssignedToKinds[0] != "conversation" {
		t.Fatalf("first canonical kind = %q", canonical.AssignedToKinds[0])
	}
	if definition.AssignedToKinds[0] != "message" {
		t.Fatal("Canonical() mutated input")
	}
}

func TestTermRejectsParentInFlatTaxonomy(t *testing.T) {
	definition := taxonomy.Definition{
		ID:              "topic",
		Label:           content.LocalizedText{"en": "Topic"},
		Mode:            taxonomy.ModeFlat,
		AssignedToKinds: []content.Kind{content.KindPost},
		Version:         1,
	}
	term := taxonomy.Term{
		ID: "child", TaxonomyID: "topic",
		Name: content.LocalizedText{"en": "Child"}, Slug: content.LocalizedText{"en": "child"},
		ParentID: "parent", Version: 1,
	}
	if err := term.Validate(definition); err == nil {
		t.Fatal("Validate() error = nil")
	}
}

func TestAssignmentRejectsDisallowedKind(t *testing.T) {
	definition := taxonomy.Definition{
		ID:              "topic",
		Label:           content.LocalizedText{"en": "Topic"},
		Mode:            taxonomy.ModeFlat,
		AssignedToKinds: []content.Kind{content.KindPost},
		Version:         1,
	}
	term := taxonomy.Term{
		ID: "go", TaxonomyID: "topic",
		Name: content.LocalizedText{"en": "Go"}, Slug: content.LocalizedText{"en": "go"},
		Version: 1,
	}
	assignment := taxonomy.Assignment{
		ResourceKind: "message", ResourceID: "message-1",
		TaxonomyID: "topic", TermID: "go",
	}
	if err := assignment.Validate(definition, term); err == nil {
		t.Fatal("Validate() error = nil")
	}
}

func TestDefinitionRejectsEmptyLabel(t *testing.T) {
	definition := taxonomy.Definition{
		ID:              "topic",
		Mode:            taxonomy.ModeFlat,
		AssignedToKinds: []content.Kind{content.KindPost},
		Version:         1,
	}
	if err := definition.Validate(); err == nil {
		t.Fatal("Validate() error = nil")
	}
}
