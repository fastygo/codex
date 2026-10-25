// Package taxonomy defines Codex classification vocabularies and terms.
package taxonomy

import (
	"slices"
	"strings"
	"time"

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
	ID              string         `json:"id"`
	Locale          string         `json:"locale"`
	Label           string         `json:"label"`
	Mode            Mode           `json:"mode"`
	AssignedToKinds []content.Kind `json:"assigned_to_kinds"`
	Public          bool           `json:"public,omitempty"`
	Version         uint64         `json:"version"`
}

type Term struct {
	ID          ID                               `json:"id"`
	Locale      string                           `json:"locale"`
	TaxonomyID  string                           `json:"taxonomy_id"`
	Name        string                           `json:"name"`
	Slug        string                           `json:"slug"`
	Description string                           `json:"description,omitempty"`
	ParentID    ID                               `json:"parent_id,omitempty"`
	Origin      []content.Origin                 `json:"origin,omitempty"`
	Metadata    map[string]content.MetadataValue `json:"metadata,omitempty"`
	Version     uint64                           `json:"version"`
	CreatedAt   time.Time                        `json:"created_at"`
	UpdatedAt   time.Time                        `json:"updated_at"`
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
	if !content.ValidLocale(definition.Locale) {
		return validation.New(validation.CoreLocaleInvalid, "locale", "locale must be a BCP 47 tag like \"ru\" or \"en-US\"")
	}
	if definition.Version == 0 {
		return validation.New(validation.CoreVersionRequired, "version", "taxonomy version is required")
	}
	if definition.Mode != ModeFlat && definition.Mode != ModeHierarchical {
		return validation.New("taxonomy.definition.mode_invalid", "mode", "taxonomy mode is invalid")
	}
	if len(definition.AssignedToKinds) == 0 {
		return validation.New("taxonomy.definition.kinds_required", "assigned_to_kinds", "taxonomy requires at least one resource kind")
	}
	if strings.TrimSpace(definition.Label) == "" {
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
	return nil
}

func (term Term) Validate(definition Definition) error {
	switch {
	case strings.TrimSpace(string(term.ID)) == "":
		return validation.New(validation.CoreIDRequired, "id", "term id is required")
	case !content.ValidRecordID(string(term.ID)):
		return validation.New(validation.CoreIDInvalid, "id", "id must be opaque ASCII matching "+content.RecordIDPattern)
	case !content.ValidLocale(term.Locale):
		return validation.New(validation.CoreLocaleInvalid, "locale", "locale must be a BCP 47 tag like \"ru\" or \"en-US\"")
	case term.TaxonomyID != definition.ID:
		return validation.New("taxonomy.term.definition_mismatch", "taxonomy_id", "term taxonomy does not match definition")
	case !content.ValidIdentifier(term.TaxonomyID):
		return validation.New(validation.CoreTermTaxonomyInvalid, "taxonomy_id", "taxonomy_id is invalid")
	case term.Version == 0:
		return validation.New(validation.CoreVersionRequired, "version", "term version is required")
	case term.CreatedAt.IsZero() || term.UpdatedAt.IsZero():
		return validation.New(validation.CoreTimestampsRequired, "created_at", "created_at and updated_at must be RFC 3339")
	case term.UpdatedAt.Before(term.CreatedAt):
		return validation.New(validation.CoreTimestampsOrder, "updated_at", "updated_at precedes created_at")
	case term.ParentID == term.ID:
		return validation.New(validation.CoreTermParentSelf, "parent_id", "term cannot be its own parent")
	case definition.Mode == ModeFlat && term.ParentID != "":
		return validation.New("taxonomy.term.parent_flat", "parent_id", "flat taxonomy cannot contain parent terms")
	case strings.TrimSpace(term.Name) == "":
		return validation.New(validation.CoreTermNameRequired, "name", "term name is required")
	case strings.TrimSpace(term.Slug) == "":
		return validation.New(validation.CoreTermSlugRequired, "slug", "term slug is required")
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
	canonical.AssignedToKinds = append([]content.Kind(nil), definition.AssignedToKinds...)
	slices.Sort(canonical.AssignedToKinds)
	return canonical
}

func (term Term) Clone() Term {
	cloned := term
	cloned.Origin = append([]content.Origin(nil), term.Origin...)
	if term.Metadata != nil {
		cloned.Metadata = make(map[string]content.MetadataValue, len(term.Metadata))
		for key, value := range term.Metadata {
			cloned.Metadata[key] = value
		}
	}
	return cloned
}

// ValidateHierarchy rejects duplicate (id, locale) pairs, missing parents, and cycles.
func ValidateHierarchy(definition Definition, terms []Term) error {
	if err := definition.Validate(); err != nil {
		return err
	}
	byKey := make(map[string]Term, len(terms))
	byID := make(map[ID]Term, len(terms))
	for _, term := range terms {
		if err := term.Validate(definition); err != nil {
			return err
		}
		key := string(term.ID) + "\x00" + term.Locale
		if _, exists := byKey[key]; exists {
			return validation.New(validation.CoreTermKeyDuplicated, string(term.ID), "term key (id, locale) is duplicated")
		}
		byKey[key] = term
		byID[term.ID] = term
	}
	for _, term := range terms {
		if term.ParentID == "" {
			continue
		}
		if _, exists := byID[term.ParentID]; !exists {
			return validation.New(validation.CoreTermParentMissing, string(term.ParentID), "taxonomy parent does not exist")
		}
		visited := map[string]struct{}{string(term.ID) + "\x00" + term.Locale: {}}
		parentID := term.ParentID
		locale := term.Locale
		for parentID != "" {
			parentKey := string(parentID) + "\x00" + locale
			if _, cycle := visited[parentKey]; cycle {
				return validation.New(validation.CoreTermParentCycle, string(parentID), "taxonomy hierarchy contains a cycle")
			}
			parent, exists := byKey[parentKey]
			if !exists {
				break
			}
			visited[parentKey] = struct{}{}
			parentID = parent.ParentID
		}
	}
	return nil
}
