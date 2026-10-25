package taxonomy_test

import (
	"testing"
	"time"

	"github.com/fastygo/codex/content"
	"github.com/fastygo/codex/taxonomy"
)

func TestValidateHierarchy(t *testing.T) {
	now := testTime()
	definition := validDefinition()
	terms := []taxonomy.Term{
		validTerm("go", "Go", "go", "", now),
		validTerm("wails", "Wails", "wails", "go", now),
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
	now := testTime()
	definition := taxonomy.Definition{
		ID: "topic", Locale: "en", Label: "Topic",
		Mode: taxonomy.ModeHierarchical, AssignedToKinds: []content.Kind{content.KindPost}, Version: 1,
	}
	terms := []taxonomy.Term{
		validTerm("a", "A", "a", "b", now),
		validTerm("b", "B", "b", "a", now),
	}
	terms[0].TaxonomyID, terms[1].TaxonomyID = "topic", "topic"
	if err := taxonomy.ValidateHierarchy(definition, terms); err == nil {
		t.Fatal("ValidateHierarchy() error = nil")
	}
}

func TestDefinitionCanonicalDoesNotMutate(t *testing.T) {
	definition := taxonomy.Definition{
		ID: "topic", Locale: "en", Label: "Topic",
		Mode: taxonomy.ModeFlat, AssignedToKinds: []content.Kind{"message", "conversation"}, Version: 1,
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
	now := testTime()
	definition := taxonomy.Definition{
		ID: "topic", Locale: "en", Label: "Topic",
		Mode: taxonomy.ModeFlat, AssignedToKinds: []content.Kind{content.KindPost}, Version: 1,
	}
	term := validTerm("child", "Child", "child", "parent", now)
	if err := term.Validate(definition); err == nil {
		t.Fatal("Validate() error = nil")
	}
}

func TestAssignmentRejectsDisallowedKind(t *testing.T) {
	now := testTime()
	definition := taxonomy.Definition{
		ID: "topic", Locale: "en", Label: "Topic",
		Mode: taxonomy.ModeFlat, AssignedToKinds: []content.Kind{content.KindPost}, Version: 1,
	}
	term := validTerm("go", "Go", "go", "", now)
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
		ID: "topic", Locale: "en",
		Mode: taxonomy.ModeFlat, AssignedToKinds: []content.Kind{content.KindPost}, Version: 1,
	}
	if err := definition.Validate(); err == nil {
		t.Fatal("Validate() error = nil")
	}
}

func TestTermRejectsInvalidID(t *testing.T) {
	now := testTime()
	term := validTerm("пост-1", "X", "x", "", now)
	if err := term.Validate(validDefinition()); err == nil {
		t.Fatal("Validate() error = nil")
	}
}

func validDefinition() taxonomy.Definition {
	return taxonomy.Definition{
		ID: "topic", Locale: "en", Label: "Topic",
		Mode: taxonomy.ModeHierarchical, AssignedToKinds: []content.Kind{"message"}, Version: 1,
	}
}

func validTerm(id, name, slug string, parent taxonomy.ID, now time.Time) taxonomy.Term {
	return taxonomy.Term{
		ID: taxonomy.ID(id), Locale: "en", TaxonomyID: "topic",
		Name: name, Slug: slug, ParentID: parent, Version: 1,
		CreatedAt: now, UpdatedAt: now,
	}
}

func testTime() time.Time {
	return time.Date(2026, time.September, 9, 12, 0, 0, 0, time.UTC)
}
