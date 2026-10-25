// Package migrate converts Codex v0.3 documents to v0.4 records.
package migrate

import (
	"encoding/json"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fastygo/codex/canonical"
	"github.com/fastygo/codex/content"
	"github.com/fastygo/codex/taxonomy"
	"github.com/fastygo/codex/validation"
)

// EntryJSON converts a v0.3 Entry JSON document into a v0.4 Entry for locale.
// LocalizedText maps become the string for that locale (or the first sorted
// key). The locales map is dropped. Integers outside ±(2^53-1) are rejected.
func EntryJSON(data []byte, locale string) (content.Entry, error) {
	var src v03Entry
	if err := unmarshal(data, &src); err != nil {
		return content.Entry{}, err
	}
	if locale == "" {
		locale = content.DefaultLocale
	}
	if !content.ValidLocale(locale) {
		return content.Entry{}, validation.New(validation.CoreLocaleInvalid, "locale", "locale must be a BCP 47 tag like \"ru\" or \"en-US\"")
	}
	entry := content.Entry{
		ID:              content.ID(src.ID),
		Locale:          locale,
		Kind:            content.Kind(src.Kind),
		Status:          content.Status(src.Status),
		Visibility:      content.Visibility(src.Visibility),
		Slug:            pickText(src.Slug, locale),
		Title:           pickText(src.Title, locale),
		Content:         pickText(src.Content, locale),
		Excerpt:         pickText(src.Excerpt, locale),
		AuthorID:        src.AuthorID,
		ParentID:        content.ID(src.ParentID),
		FeaturedMediaID: src.FeaturedMediaID,
		Template:        src.Template,
		Terms:           src.Terms,
		Version:         src.Version,
		CreatedAt:       src.CreatedAt,
		UpdatedAt:       src.UpdatedAt,
		PublishedAt:     src.PublishedAt,
		DeletedAt:       src.DeletedAt,
	}
	md, err := convertMetadata(src.Metadata)
	if err != nil {
		return content.Entry{}, err
	}
	entry.Metadata = md
	if src.Source != "" {
		entry.Origin = append(entry.Origin, content.Origin{System: "v0.3", ID: src.Source})
	}
	if err := entry.Validate(); err != nil {
		return content.Entry{}, err
	}
	return entry, nil
}

// TaxonomyJSON converts a v0.3 taxonomy bundle JSON document to v0.4.
func TaxonomyJSON(data []byte, locale string) (taxonomy.Definition, []taxonomy.Term, error) {
	var src v03Taxonomy
	if err := unmarshal(data, &src); err != nil {
		return taxonomy.Definition{}, nil, err
	}
	if locale == "" {
		locale = content.DefaultLocale
	}
	def := taxonomy.Definition{
		ID:              src.Definition.ID,
		Locale:          locale,
		Label:           pickText(src.Definition.Label, locale),
		Mode:            taxonomy.Mode(src.Definition.Mode),
		AssignedToKinds: src.Definition.AssignedToKinds,
		Public:          src.Definition.Public,
		Version:         src.Definition.Version,
	}
	now := src.Definition.UpdatedAt
	if now.IsZero() {
		now = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	}
	terms := make([]taxonomy.Term, 0, len(src.Terms))
	for _, item := range src.Terms {
		created, updated := item.CreatedAt, item.UpdatedAt
		if created.IsZero() {
			created = now
		}
		if updated.IsZero() {
			updated = now
		}
		term := taxonomy.Term{
			ID:          taxonomy.ID(item.ID),
			Locale:      locale,
			TaxonomyID:  item.TaxonomyID,
			Name:        pickText(item.Name, locale),
			Slug:        pickText(item.Slug, locale),
			Description: pickText(item.Description, locale),
			ParentID:    taxonomy.ID(item.ParentID),
			Version:     item.Version,
			CreatedAt:   created,
			UpdatedAt:   updated,
		}
		md, err := convertMetadata(item.Metadata)
		if err != nil {
			return taxonomy.Definition{}, nil, err
		}
		term.Metadata = md
		terms = append(terms, term)
	}
	if err := taxonomy.ValidateHierarchy(def, terms); err != nil {
		return taxonomy.Definition{}, nil, err
	}
	return def, terms, nil
}

type v03Entry struct {
	ID              string                           `json:"id"`
	Kind            string                           `json:"kind"`
	Status          string                           `json:"status"`
	Visibility      string                           `json:"visibility"`
	Slug            map[string]string                `json:"slug"`
	Title           map[string]string                `json:"title"`
	Content         map[string]string                `json:"content"`
	Excerpt         map[string]string                `json:"excerpt"`
	AuthorID        string                           `json:"author_id"`
	ParentID        string                           `json:"parent_id"`
	FeaturedMediaID string                           `json:"featured_media_id"`
	Template        string                           `json:"template"`
	Source          string                           `json:"source"`
	Metadata        map[string]content.MetadataValue `json:"metadata"`
	Terms           []content.TermRef                `json:"terms"`
	Version         uint64                           `json:"version"`
	CreatedAt       time.Time                        `json:"created_at"`
	UpdatedAt       time.Time                        `json:"updated_at"`
	PublishedAt     *time.Time                       `json:"published_at"`
	DeletedAt       *time.Time                       `json:"deleted_at"`
}

type v03Taxonomy struct {
	Definition struct {
		ID              string            `json:"id"`
		Label           map[string]string `json:"label"`
		Mode            string            `json:"mode"`
		AssignedToKinds []content.Kind    `json:"assigned_to_kinds"`
		Public          bool              `json:"public"`
		Version         uint64            `json:"version"`
		UpdatedAt       time.Time         `json:"updated_at"`
	} `json:"definition"`
	Terms []struct {
		ID          string                           `json:"id"`
		TaxonomyID  string                           `json:"taxonomy_id"`
		Name        map[string]string                `json:"name"`
		Slug        map[string]string                `json:"slug"`
		Description map[string]string                `json:"description"`
		ParentID    string                           `json:"parent_id"`
		Metadata    map[string]content.MetadataValue `json:"metadata"`
		Version     uint64                           `json:"version"`
		CreatedAt   time.Time                        `json:"created_at"`
		UpdatedAt   time.Time                        `json:"updated_at"`
	} `json:"terms"`
}

func pickText(values map[string]string, locale string) string {
	if values == nil {
		return ""
	}
	if text, ok := values[locale]; ok {
		return text
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return ""
	}
	return values[keys[0]]
}

func convertMetadata(src map[string]content.MetadataValue) (map[string]content.MetadataValue, error) {
	if src == nil {
		return nil, nil
	}
	out := make(map[string]content.MetadataValue, len(src))
	for key, value := range src {
		converted, err := convertJSONValue(value.Value)
		if err != nil {
			return nil, err
		}
		out[key] = content.MetadataValue{Value: converted, Private: value.Private}
	}
	return out, nil
}

func convertJSONValue(value any) (any, error) {
	switch typed := value.(type) {
	case json.Number:
		if err := rejectUnsafeNumber(typed); err != nil {
			return nil, err
		}
		return typed, nil
	case float64:
		if typed == math.Trunc(typed) && math.Abs(typed) > float64(content.MaxSafeInteger) {
			return nil, canonicalFail(typed)
		}
		return typed, nil
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			converted, err := convertJSONValue(item)
			if err != nil {
				return nil, err
			}
			out[key] = converted
		}
		return out, nil
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			converted, err := convertJSONValue(item)
			if err != nil {
				return nil, err
			}
			out[i] = converted
		}
		return out, nil
	default:
		return value, nil
	}
}

func rejectUnsafeNumber(n json.Number) error {
	s := n.String()
	if strings.ContainsAny(s, ".eE") {
		f, err := n.Float64()
		if err != nil {
			return err
		}
		if f == math.Trunc(f) && math.Abs(f) > float64(content.MaxSafeInteger) {
			return canonicalFail(f)
		}
		return nil
	}
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return &canonical.Error{Code: canonical.ErrUnsafeInteger, Path: "metadata", Message: "integer " + s + " is outside ±(2^53-1)"}
	}
	if i > content.MaxSafeInteger || i < -content.MaxSafeInteger {
		return &canonical.Error{Code: canonical.ErrUnsafeInteger, Path: "metadata", Message: "integer " + s + " is outside ±(2^53-1)"}
	}
	return nil
}

func canonicalFail(n float64) error {
	return &canonical.Error{
		Code:    canonical.ErrUnsafeInteger,
		Path:    "metadata",
		Message: "integer " + strconv.FormatFloat(n, 'f', -1, 64) + " is outside ±(2^53-1)",
	}
}

func unmarshal(data []byte, target any) error {
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.UseNumber()
	return dec.Decode(target)
}
