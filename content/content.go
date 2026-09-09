// Package content defines the protocol-neutral Codex entry aggregate.
package content

import (
	"strings"
	"time"

	"github.com/fastygo/codex/validation"
)

type ID string
type Kind string
type Status string
type Visibility string

const (
	KindPost    Kind = "post"
	KindPage    Kind = "page"
	KindMenu    Kind = "menu"
	KindSetting Kind = "setting"
)

const (
	StatusDraft     Status = "draft"
	StatusScheduled Status = "scheduled"
	StatusPublished Status = "published"
	StatusArchived  Status = "archived"
	StatusTrashed   Status = "trashed"
)

const (
	VisibilityPublic  Visibility = "public"
	VisibilityPrivate Visibility = "private"
)

type LocalizedText map[string]string

type MetadataValue struct {
	Value   any  `json:"value"`
	Private bool `json:"private,omitempty"`
}

type TermRef struct {
	Taxonomy string `json:"taxonomy"`
	TermID   string `json:"term_id"`
}

// Entry is the shared content aggregate. Kind is the manifest-backed post type.
type Entry struct {
	ID              ID                        `json:"id"`
	Kind            Kind                      `json:"kind"`
	Status          Status                    `json:"status"`
	Visibility      Visibility                `json:"visibility"`
	Slug            LocalizedText             `json:"slug"`
	Title           LocalizedText             `json:"title"`
	Content         LocalizedText             `json:"content,omitempty"`
	Excerpt         LocalizedText             `json:"excerpt,omitempty"`
	AuthorID        string                    `json:"author_id,omitempty"`
	ParentID        ID                        `json:"parent_id,omitempty"`
	FeaturedMediaID string                    `json:"featured_media_id,omitempty"`
	Template        string                    `json:"template,omitempty"`
	Metadata        map[string]MetadataValue  `json:"metadata,omitempty"`
	Locales         map[string]LocaleDocument `json:"locales,omitempty"`
	Terms           []TermRef                 `json:"terms,omitempty"`
	Version         uint64                    `json:"version"`
	CreatedAt       time.Time                 `json:"created_at"`
	UpdatedAt       time.Time                 `json:"updated_at"`
	PublishedAt     *time.Time                `json:"published_at,omitempty"`
	DeletedAt       *time.Time                `json:"deleted_at,omitempty"`
}

// Validate verifies invariants independent from storage and delivery.
func (entry Entry) Validate() error {
	switch {
	case strings.TrimSpace(string(entry.ID)) == "":
		return validation.New("content.entry.id_required", "id", "content id is required")
	case !ValidKind(entry.Kind):
		return validation.New("content.entry.kind_invalid", "kind", "content kind is invalid")
	case !entry.Status.Valid():
		return validation.New("content.entry.status_invalid", "status", "content status is invalid")
	case !entry.Visibility.Valid():
		return validation.New("content.entry.visibility_invalid", "visibility", "content visibility is invalid")
	case entry.Version == 0:
		return validation.New("content.entry.version_required", "version", "content version is required")
	case entry.CreatedAt.IsZero() || entry.UpdatedAt.IsZero():
		return validation.New("content.entry.timestamps_required", "created_at", "content timestamps are required")
	case entry.UpdatedAt.Before(entry.CreatedAt):
		return validation.New("content.entry.timestamps_order", "updated_at", "content updated_at precedes created_at")
	case entry.ParentID != "" && entry.ParentID == entry.ID:
		return validation.New("content.entry.parent_self", "parent_id", "content cannot be its own parent")
	case entry.Status == StatusScheduled && entry.PublishedAt == nil:
		return validation.New("content.entry.published_at_required", "published_at", "scheduled content requires published_at")
	case entry.Status == StatusScheduled && !entry.PublishedAt.After(entry.UpdatedAt):
		return validation.New("content.entry.published_at_future", "published_at", "scheduled content requires a future published_at")
	case entry.Status == StatusTrashed && entry.DeletedAt == nil:
		return validation.New("content.entry.deleted_at_required", "deleted_at", "trashed content requires deleted_at")
	case !hasLocalizedValue(entry.Slug):
		return validation.New("content.entry.slug_required", "slug", "content slug is required")
	case !hasLocalizedValue(entry.Title):
		return validation.New("content.entry.title_required", "title", "content title is required")
	}
	for _, values := range []LocalizedText{entry.Slug, entry.Title, entry.Content, entry.Excerpt} {
		if err := ValidateLocalizedText(values); err != nil {
			return validation.Wrap("content.entry.localized_text_invalid", "", err)
		}
	}
	if err := validateMetadata(entry.Metadata); err != nil {
		return err
	}
	if err := validateLocales(entry.Locales); err != nil {
		return err
	}
	if err := validateTermRefs(entry.Terms); err != nil {
		return err
	}
	return nil
}

func (entry Entry) IsPublicAt(now time.Time) bool {
	if entry.Visibility != VisibilityPublic || entry.DeletedAt != nil {
		return false
	}
	if entry.Status != StatusPublished {
		return false
	}
	return entry.PublishedAt == nil || !entry.PublishedAt.After(now)
}

// PublicProjection removes private metadata and clones mutable containers.
func (entry Entry) PublicProjection() Entry {
	projected := entry.Clone()
	for key, value := range projected.Metadata {
		if value.Private {
			delete(projected.Metadata, key)
		}
	}
	return projected
}

// Clone copies all contract-owned mutable containers.
func (entry Entry) Clone() Entry {
	cloned := entry
	cloned.Slug = cloneLocalizedText(entry.Slug)
	cloned.Title = cloneLocalizedText(entry.Title)
	cloned.Content = cloneLocalizedText(entry.Content)
	cloned.Excerpt = cloneLocalizedText(entry.Excerpt)
	cloned.Metadata = cloneMetadata(entry.Metadata)
	cloned.Locales = cloneLocales(entry.Locales)
	cloned.Terms = append([]TermRef(nil), entry.Terms...)
	if entry.PublishedAt != nil {
		value := *entry.PublishedAt
		cloned.PublishedAt = &value
	}
	if entry.DeletedAt != nil {
		value := *entry.DeletedAt
		cloned.DeletedAt = &value
	}
	return cloned
}

func (status Status) Valid() bool {
	switch status {
	case StatusDraft, StatusScheduled, StatusPublished, StatusArchived, StatusTrashed:
		return true
	default:
		return false
	}
}

func (visibility Visibility) Valid() bool {
	return visibility == VisibilityPublic || visibility == VisibilityPrivate
}

func ValidKind(kind Kind) bool {
	return ValidIdentifier(string(kind))
}

// ValidIdentifier reports whether value is a canonical Codex identifier.
func ValidIdentifier(value string) bool {
	if len(value) == 0 || len(value) > 63 || value[0] < 'a' || value[0] > 'z' {
		return false
	}
	for _, character := range value[1:] {
		if (character >= 'a' && character <= 'z') ||
			(character >= '0' && character <= '9') ||
			character == '_' || character == '-' {
			continue
		}
		return false
	}
	return true
}

func hasLocalizedValue(values LocalizedText) bool {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
}

func ValidateLocalizedText(values LocalizedText) error {
	for locale := range values {
		if NormalizeLocale(locale) == "" || NormalizeLocale(locale) != locale {
			return validation.New("content.locale.noncanonical", locale, "localized text locale is not canonical")
		}
	}
	return nil
}

func cloneLocalizedText(source LocalizedText) LocalizedText {
	if source == nil {
		return nil
	}
	target := make(LocalizedText, len(source))
	for key, value := range source {
		target[key] = value
	}
	return target
}

func cloneMetadata(source map[string]MetadataValue) map[string]MetadataValue {
	if source == nil {
		return nil
	}
	target := make(map[string]MetadataValue, len(source))
	for key, value := range source {
		value.Value = cloneValue(value.Value)
		target[key] = value
	}
	return target
}

func validateMetadata(metadata map[string]MetadataValue) error {
	for key, value := range metadata {
		if strings.TrimSpace(key) == "" {
			return validation.New("content.metadata.key_required", key, "content metadata key is required")
		}
		if err := ValidateJSONValue(value.Value); err != nil {
			return validation.Wrap("content.metadata.value_invalid", key, err)
		}
	}
	return nil
}

func validateLocales(locales map[string]LocaleDocument) error {
	for locale, document := range locales {
		if NormalizeLocale(locale) == "" || NormalizeLocale(locale) != locale {
			return validation.New("content.locale.noncanonical", locale, "content locale is not canonical")
		}
		if !document.Status.Valid() {
			return validation.New("content.locale.status_invalid", locale+".status", "content locale status is invalid")
		}
		if err := ValidateJSONValue(document.Data); err != nil {
			return validation.Wrap("content.locale.data_invalid", locale+".data", err)
		}
	}
	return nil
}

func validateTermRefs(terms []TermRef) error {
	seen := make(map[TermRef]struct{}, len(terms))
	for _, term := range terms {
		if !ValidIdentifier(term.Taxonomy) || strings.TrimSpace(term.TermID) == "" {
			return validation.New("content.term_ref.incomplete", term.Taxonomy, "content term reference is incomplete")
		}
		if _, exists := seen[term]; exists {
			return validation.New("content.term_ref.duplicated", term.Taxonomy, "content term reference is duplicated")
		}
		seen[term] = struct{}{}
	}
	return nil
}
