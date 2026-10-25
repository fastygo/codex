// Package content defines the protocol-neutral Codex entry aggregate.
package content

import (
	"regexp"
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
	StatusPending   Status = "pending"
	StatusScheduled Status = "scheduled"
	StatusPublished Status = "published"
	StatusArchived  Status = "archived"
	StatusTrashed   Status = "trashed"
)

const (
	VisibilityPublic  Visibility = "public"
	VisibilityPrivate Visibility = "private"
)

// MaxSafeInteger is 2^53-1, the largest integer IEEE-754 float64 (and JSON.parse)
// can represent exactly. Integer and money rules, and the canonical profile,
// reject |n| greater than this (Q2).
const MaxSafeInteger = 1<<53 - 1

// RecordIDPattern is the Q8 opaque id grammar. The vendored JSON Schema stays
// looser (\S); Go conformance and Validate enforce this pattern.
const RecordIDPattern = `^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`

var recordID = regexp.MustCompile(RecordIDPattern)

type MetadataValue struct {
	Value   any  `json:"value"`
	Private bool `json:"private,omitempty"`
}

type TermRef struct {
	Taxonomy string `json:"taxonomy"`
	TermID   string `json:"term_id"`
}

// Origin records where a value came from during a migration or import.
type Origin struct {
	System string `json:"system"`
	ID     string `json:"id"`
}

// Discussion is whether comments and pings are open.
type Discussion struct {
	Comments bool `json:"comments"`
	Pings    bool `json:"pings"`
}

// ChromeField reports whether id is a v0.4 entry chrome field. Chrome is always
// localized (Q7); it does not use Field.Localized.
func ChromeField(id string) bool {
	switch id {
	case "slug", "title", "content", "excerpt":
		return true
	default:
		return false
	}
}

// Entry is the shared content aggregate. Kind is the manifest-backed post type.
// The record key is (id, locale). Translations are separate records.
type Entry struct {
	ID              ID                       `json:"id"`
	Locale          string                   `json:"locale"`
	Kind            Kind                     `json:"kind"`
	Status          Status                   `json:"status"`
	Visibility      Visibility               `json:"visibility"`
	Slug            string                   `json:"slug"`
	Title           string                   `json:"title"`
	Content         string                   `json:"content,omitempty"`
	Excerpt         string                   `json:"excerpt,omitempty"`
	AuthorID        string                   `json:"author_id,omitempty"`
	ParentID        ID                       `json:"parent_id,omitempty"`
	FeaturedMediaID string                   `json:"featured_media_id,omitempty"`
	Template        string                   `json:"template,omitempty"`
	Metadata        map[string]MetadataValue `json:"metadata,omitempty"`
	Terms           []TermRef                `json:"terms,omitempty"`
	Order           *int64                   `json:"order,omitempty"`
	Pinned          *bool                    `json:"pinned,omitempty"`
	Format          string                   `json:"format,omitempty"`
	Discussion      *Discussion              `json:"discussion,omitempty"`
	Origin          []Origin                 `json:"origin,omitempty"`
	Version         uint64                   `json:"version"`
	CreatedAt       time.Time                `json:"created_at"`
	UpdatedAt       time.Time                `json:"updated_at"`
	PublishedAt     *time.Time               `json:"published_at,omitempty"`
	DeletedAt       *time.Time               `json:"deleted_at,omitempty"`
}

// Validate verifies v0.4 core invariants independent from storage and delivery.
func (entry Entry) Validate() error {
	switch {
	case strings.TrimSpace(string(entry.ID)) == "":
		return validation.New(validation.CoreIDRequired, "id", "id is required")
	case !ValidRecordID(string(entry.ID)):
		return validation.New(validation.CoreIDInvalid, "id", "id must be opaque ASCII matching "+RecordIDPattern)
	case !ValidLocale(entry.Locale):
		return validation.New(validation.CoreLocaleInvalid, "locale", "locale must be a BCP 47 tag like \"ru\" or \"en-US\"")
	case !ValidKind(entry.Kind):
		return validation.New(validation.CoreEntryKindInvalid, "kind", "kind is invalid")
	case !entry.Status.Valid():
		return validation.New(validation.CoreEntryStatusInvalid, "status", "content status is invalid")
	case !entry.Visibility.Valid():
		return validation.New(validation.CoreEntryVisibilityInvalid, "visibility", "visibility must be public or private")
	case strings.TrimSpace(entry.Slug) == "":
		return validation.New(validation.CoreEntrySlugRequired, "slug", "slug is required")
	case entry.Version == 0:
		return validation.New(validation.CoreVersionRequired, "version", "version must be an integer >= 1")
	case entry.CreatedAt.IsZero() || entry.UpdatedAt.IsZero():
		return validation.New(validation.CoreTimestampsRequired, "created_at", "created_at and updated_at must be RFC 3339")
	case entry.UpdatedAt.Before(entry.CreatedAt):
		return validation.New(validation.CoreTimestampsOrder, "updated_at", "updated_at precedes created_at")
	case entry.ParentID != "" && entry.ParentID == entry.ID:
		return validation.New(validation.CoreEntryParentSelf, "parent_id", "entry cannot be its own parent")
	case entry.Status == StatusScheduled && entry.PublishedAt == nil:
		return validation.New(validation.CoreEntryPublishedAtRequired, "published_at", "scheduled entry requires published_at")
	case entry.Status == StatusTrashed && entry.DeletedAt == nil:
		return validation.New(validation.CoreEntryDeletedAtRequired, "deleted_at", "trashed entry requires deleted_at")
	}
	if err := validateOrigin(entry.Origin); err != nil {
		return err
	}
	if err := validateMetadata(entry.Metadata); err != nil {
		return err
	}
	if err := validateTermRefs(entry.Terms); err != nil {
		return err
	}
	if entry.Discussion != nil {
		// Struct fields are always bool; nothing else to check on a typed value.
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
// Sensitive fields declared on a resource are stripped by schema.Resource.PublicProjection.
func (entry Entry) PublicProjection() Entry {
	projected := entry.Clone()
	for key, value := range projected.Metadata {
		if value.Private {
			delete(projected.Metadata, key)
		}
	}
	if len(projected.Metadata) == 0 {
		projected.Metadata = nil
	}
	return projected
}

// Clone copies all contract-owned mutable containers.
func (entry Entry) Clone() Entry {
	cloned := entry
	cloned.Metadata = cloneMetadata(entry.Metadata)
	cloned.Terms = append([]TermRef(nil), entry.Terms...)
	cloned.Origin = append([]Origin(nil), entry.Origin...)
	if entry.Discussion != nil {
		value := *entry.Discussion
		cloned.Discussion = &value
	}
	if entry.Order != nil {
		value := *entry.Order
		cloned.Order = &value
	}
	if entry.Pinned != nil {
		value := *entry.Pinned
		cloned.Pinned = &value
	}
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
	case StatusDraft, StatusPending, StatusScheduled, StatusPublished, StatusArchived, StatusTrashed:
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

// ValidIdentifier reports whether value is a canonical Codex identifier (kinds, taxonomies).
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

// ValidRecordID reports whether value matches the Q8 opaque id grammar.
func ValidRecordID(value string) bool {
	return len(value) >= 1 && len(value) <= 128 && recordID.MatchString(value)
}

// IDFromKindInt is the Q8 rule for an integer platform primary key: <kind>-<n>.
// Canonicalize does not rewrite ids; adapters use this when creating records.
func IDFromKindInt(kind Kind, n int64) string {
	var b strings.Builder
	b.WriteString(string(kind))
	b.WriteByte('-')
	b.WriteString(itoa(n))
	return b.String()
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	negative := n < 0
	if negative {
		n = -n
	}
	var buf [32]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if negative {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
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
			return validation.New(validation.CoreMetadataKeyRequired, key, "metadata key is required")
		}
		if err := ValidateJSONValue(value.Value); err != nil {
			return validation.Wrap(validation.CoreMetadataInvalid, key, err)
		}
	}
	return nil
}

func validateOrigin(origin []Origin) error {
	for i, item := range origin {
		if strings.TrimSpace(item.System) == "" || strings.TrimSpace(item.ID) == "" {
			return validation.New(validation.CoreOriginInvalid, "origin", "origin needs a non-empty system and a string id")
		}
		_ = i
	}
	return nil
}

func validateTermRefs(terms []TermRef) error {
	seen := make(map[TermRef]struct{}, len(terms))
	for _, term := range terms {
		if !ValidIdentifier(term.Taxonomy) || strings.TrimSpace(term.TermID) == "" {
			return validation.New(validation.CoreTermRefIncomplete, term.Taxonomy, "term reference must be {taxonomy, term_id}")
		}
		if _, exists := seen[term]; exists {
			return validation.New(validation.CoreTermRefDuplicated, term.Taxonomy, "content term reference is duplicated")
		}
		seen[term] = struct{}{}
	}
	return nil
}
