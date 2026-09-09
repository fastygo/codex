// Package taxonomy defines Codex classification vocabularies and terms.
package taxonomy

import (
	"errors"
	"slices"
	"strings"

	"github.com/fastygo/codex/content"
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
		return errors.New("taxonomy id is invalid")
	}
	if definition.Version == 0 {
		return errors.New("taxonomy version is required")
	}
	if definition.Mode != ModeFlat && definition.Mode != ModeHierarchical {
		return errors.New("taxonomy mode is invalid")
	}
	if len(definition.AssignedToKinds) == 0 {
		return errors.New("taxonomy requires at least one resource kind")
	}
	if !hasLocalizedValue(definition.Label) {
		return errors.New("taxonomy label is required")
	}
	seen := make(map[content.Kind]struct{}, len(definition.AssignedToKinds))
	for _, kind := range definition.AssignedToKinds {
		if !content.ValidKind(kind) {
			return errors.New("taxonomy resource kind is invalid")
		}
		if _, exists := seen[kind]; exists {
			return errors.New("taxonomy resource kind is duplicated")
		}
		seen[kind] = struct{}{}
	}
	return content.ValidateLocalizedText(definition.Label)
}

func (term Term) Validate(definition Definition) error {
	switch {
	case strings.TrimSpace(string(term.ID)) == "":
		return errors.New("term id is required")
	case term.TaxonomyID != definition.ID:
		return errors.New("term taxonomy does not match definition")
	case term.Version == 0:
		return errors.New("term version is required")
	case term.ParentID == term.ID:
		return errors.New("term cannot be its own parent")
	case definition.Mode == ModeFlat && term.ParentID != "":
		return errors.New("flat taxonomy cannot contain parent terms")
	case !hasLocalizedValue(term.Name):
		return errors.New("term name is required")
	case !hasLocalizedValue(term.Slug):
		return errors.New("term slug is required")
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
		return errors.New("assignment resource kind is invalid")
	case strings.TrimSpace(string(assignment.ResourceID)) == "":
		return errors.New("assignment resource id is required")
	case assignment.TaxonomyID != definition.ID:
		return errors.New("assignment taxonomy does not match definition")
	case assignment.TermID != term.ID:
		return errors.New("assignment term does not match")
	case !definition.Allows(assignment.ResourceKind):
		return errors.New("taxonomy does not allow resource kind")
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
			return errors.New("term id is duplicated")
		}
		byID[term.ID] = term
	}
	for _, term := range terms {
		visited := map[ID]struct{}{term.ID: {}}
		parentID := term.ParentID
		for parentID != "" {
			if _, cycle := visited[parentID]; cycle {
				return errors.New("taxonomy hierarchy contains a cycle")
			}
			visited[parentID] = struct{}{}
			parent, exists := byID[parentID]
			if !exists {
				return errors.New("taxonomy parent does not exist")
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
