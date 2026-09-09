// Package taxonomy defines Codex classification vocabularies and terms.
package taxonomy

import (
	"slices"
	"strings"

	"github.com/fastygo/codex/content"
	"github.com/fastygo/codex/validation"
)

type ID string
type Mode string

const (
	ModeFlat         Mode = "flat"
	ModeHierarchical Mode = "hierarchical"
)

type Definition struct {
	ID              string                `json:"id"`
	Label           content.LocalizedText `json:"label"`
	Mode            Mode                  `json:"mode"`
	AssignedToKinds []content.Kind        `json:"assigned_to_kinds"`
	Public          bool                  `json:"public,omitempty"`
	Version         uint64                `json:"version"`
}

type Term struct {
	ID          ID                    `json:"id"`
	TaxonomyID  string                `json:"taxonomy_id"`
	Name        content.LocalizedText `json:"name"`
	Slug        content.LocalizedText `json:"slug"`
	Description content.LocalizedText `json:"description,omitempty"`
	ParentID    ID                    `json:"parent_id,omitempty"`
	Version     uint64                `json:"version"`
}

type Assignment struct {
	ResourceKind content.Kind `json:"resource_kind"`
	ResourceID   content.ID   `json:"resource_id"`
	TaxonomyID   string       `json:"taxonomy_id"`
	TermID       ID           `json:"term_id"`
}

func (definition Definition) Validate() error {
	if !content.ValidIdentifier(definition.ID) {
		return validation.New("taxonomy.definition.id_invalid", "id", "taxonomy id is invalid")
	}
	if definition.Version == 0 {
		return validation.New("taxonomy.definition.version_required", "version", "taxonomy version is required")
	}
	if definition.Mode != ModeFlat && definition.Mode != ModeHierarchical {
		return validation.New("taxonomy.definition.mode_invalid", "mode", "taxonomy mode is invalid")
	}
	if len(definition.AssignedToKinds) == 0 {
		return validation.New("taxonomy.definition.kinds_required", "assigned_to_kinds", "taxonomy requires at least one resource kind")
	}
	if !hasLocalizedValue(definition.Label) {
		return validation.New("taxonomy.definition.label_required", "label", "taxonomy label is required")
	}
	seen := make(map[content.Kind]struct{}, len(definition.AssignedToKinds))
	for _, kind := range definition.AssignedToKinds {
		if !content.ValidKind(kind) {
			return validation.New("taxonomy.definition.kind_invalid", string(kind), "taxonomy resource kind is invalid")
		}
		if _, exists := seen[kind]; exists {
			return validation.New("taxonomy.definition.kind_duplicated", string(kind), "taxonomy resource kind is duplicated")
		}
		seen[kind] = struct{}{}
	}
	return content.ValidateLocalizedText(definition.Label)
}

func (term Term) Validate(definition Definition) error {
	switch {
	case strings.TrimSpace(string(term.ID)) == "":
		return validation.New("taxonomy.term.id_required", "id", "term id is required")
	case term.TaxonomyID != definition.ID:
		return validation.New("taxonomy.term.definition_mismatch", "taxonomy_id", "term taxonomy does not match definition")
	case term.Version == 0:
		return validation.New("taxonomy.term.version_required", "version", "term version is required")
	case term.ParentID == term.ID:
		return validation.New("taxonomy.term.parent_self", "parent_id", "term cannot be its own parent")
	case definition.Mode == ModeFlat && term.ParentID != "":
		return validation.New("taxonomy.term.parent_flat", "parent_id", "flat taxonomy cannot contain parent terms")
	case !hasLocalizedValue(term.Name):
		return validation.New("taxonomy.term.name_required", "name", "term name is required")
	case !hasLocalizedValue(term.Slug):
		return validation.New("taxonomy.term.slug_required", "slug", "term slug is required")
	}
	for _, values := range []content.LocalizedText{term.Name, term.Slug, term.Description} {
		if err := content.ValidateLocalizedText(values); err != nil {
			return err
		}
	}
	return nil
}

func (assignment Assignment) Validate(definition Definition, term Term) error {
	if err := definition.Validate(); err != nil {
		return err
	}
	if err := term.Validate(definition); err != nil {
		return err
	}
	switch {
	case !content.ValidKind(assignment.ResourceKind):
		return validation.New("taxonomy.assignment.kind_invalid", "resource_kind", "assignment resource kind is invalid")
	case strings.TrimSpace(string(assignment.ResourceID)) == "":
		return validation.New("taxonomy.assignment.resource_required", "resource_id", "assignment resource id is required")
	case assignment.TaxonomyID != definition.ID:
		return validation.New("taxonomy.assignment.definition_mismatch", "taxonomy_id", "assignment taxonomy does not match definition")
	case assignment.TermID != term.ID:
		return validation.New("taxonomy.assignment.term_mismatch", "term_id", "assignment term does not match")
	case !definition.Allows(assignment.ResourceKind):
		return validation.New("taxonomy.assignment.kind_forbidden", "resource_kind", "taxonomy does not allow resource kind")
	default:
		return nil
	}
}

func (definition Definition) Allows(kind content.Kind) bool {
	return slices.Contains(definition.AssignedToKinds, kind)
}

// Canonical returns a clone with sorted kind assignments.
func (definition Definition) Canonical() Definition {
	canonical := definition
	canonical.Label = cloneLocalizedText(definition.Label)
	canonical.AssignedToKinds = append([]content.Kind(nil), definition.AssignedToKinds...)
	slices.Sort(canonical.AssignedToKinds)
	return canonical
}

// ValidateHierarchy rejects duplicate terms, missing parents, and cycles.
func ValidateHierarchy(definition Definition, terms []Term) error {
	if err := definition.Validate(); err != nil {
		return err
	}
	byID := make(map[ID]Term, len(terms))
	for _, term := range terms {
		if err := term.Validate(definition); err != nil {
			return err
		}
		if _, exists := byID[term.ID]; exists {
			return validation.New("taxonomy.hierarchy.term_duplicated", string(term.ID), "term id is duplicated")
		}
		byID[term.ID] = term
	}
	for _, term := range terms {
		visited := map[ID]struct{}{term.ID: {}}
		parentID := term.ParentID
		for parentID != "" {
			if _, cycle := visited[parentID]; cycle {
				return validation.New("taxonomy.hierarchy.cycle", string(parentID), "taxonomy hierarchy contains a cycle")
			}
			visited[parentID] = struct{}{}
			parent, exists := byID[parentID]
			if !exists {
				return validation.New("taxonomy.hierarchy.parent_missing", string(parentID), "taxonomy parent does not exist")
			}
			parentID = parent.ParentID
		}
	}
	return nil
}

func hasLocalizedValue(values content.LocalizedText) bool {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
}

func cloneLocalizedText(source content.LocalizedText) content.LocalizedText {
	if source == nil {
		return nil
	}
	target := make(content.LocalizedText, len(source))
	for key, value := range source {
		target[key] = value
	}
	return target
}
