package content

import (
	"encoding/json"
	"slices"
	"strings"
	"time"

	"github.com/fastygo/codex/validation"
)

const DefaultLocale = "en"

type LocaleDocument struct {
	Data      map[string]any `json:"data"`
	Status    Status         `json:"status"`
	UpdatedAt time.Time      `json:"updated_at,omitzero"`
}

func (document *LocaleDocument) UnmarshalJSON(data []byte) error {
	type localeDocumentWire struct {
		Data      json.RawMessage `json:"data"`
		Status    Status          `json:"status"`
		UpdatedAt time.Time       `json:"updated_at,omitzero"`
	}
	var wire localeDocumentWire
	if err := decodeJSON(data, &wire, false); err != nil {
		return err
	}
	if len(wire.Data) == 0 {
		return validation.New("content.locale.data_required", "data", "locale data is required")
	}
	var values map[string]any
	if err := decodeJSON(wire.Data, &values, false); err != nil {
		return err
	}
	*document = LocaleDocument{
		Data:      values,
		Status:    wire.Status,
		UpdatedAt: wire.UpdatedAt,
	}
	return nil
}

type LocaleResolution struct {
	Requested string         `json:"requested"`
	Served    string         `json:"served"`
	Fallback  bool           `json:"fallback"`
	Data      map[string]any `json:"data"`
	Status    Status         `json:"status,omitempty"`
}

type LocaleIndexEntry struct {
	Locale    string    `json:"locale"`
	Status    Status    `json:"status"`
	UpdatedAt time.Time `json:"updated_at,omitzero"`
}

func NormalizeLocale(locale string) string {
	return strings.ToLower(strings.TrimSpace(locale))
}

func (text LocalizedText) Value(locale, fallback string) string {
	if value := strings.TrimSpace(text[NormalizeLocale(locale)]); value != "" {
		return value
	}
	return strings.TrimSpace(text[NormalizeLocale(fallback)])
}

// ResolveLocale returns one whole locale document and never mixes fields.
func (entry Entry) ResolveLocale(requested, fallback string) LocaleResolution {
	requested = NormalizeLocale(requested)
	if requested == "" {
		requested = DefaultLocale
	}
	fallback = NormalizeLocale(fallback)
	if fallback == "" {
		fallback = DefaultLocale
	}
	if document, ok := entry.Locales[requested]; ok && len(document.Data) > 0 {
		return localeResolution(requested, requested, false, document)
	}
	if document, ok := entry.Locales[fallback]; ok && len(document.Data) > 0 {
		return localeResolution(requested, fallback, true, document)
	}
	locales := make([]string, 0, len(entry.Locales))
	for locale, document := range entry.Locales {
		if len(document.Data) > 0 {
			locales = append(locales, locale)
		}
	}
	slices.Sort(locales)
	if len(locales) > 0 {
		served := locales[0]
		return localeResolution(requested, served, served != requested, entry.Locales[served])
	}
	return LocaleResolution{
		Requested: requested,
		Served:    requested,
		Data:      map[string]any{},
	}
}

func MergeLocales(target, patch map[string]LocaleDocument) (map[string]LocaleDocument, error) {
	if len(patch) == 0 {
		return target, nil
	}
	if target == nil {
		target = map[string]LocaleDocument{}
	}
	keys := make([]string, 0, len(patch))
	for key := range patch {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	normalizedKeys := make(map[string]string, len(keys))
	for _, key := range keys {
		locale := NormalizeLocale(key)
		if locale == "" {
			return target, validation.New("content.locale.required", key, "locale is required")
		}
		if previous, exists := normalizedKeys[locale]; exists && previous != key {
			return target, validation.New("content.locale.collision", locale, "locale patch contains a normalized collision")
		}
		normalizedKeys[locale] = key
	}
	for locale, key := range normalizedKeys {
		target[locale] = cloneLocaleDocument(patch[key])
	}
	return target, nil
}

func (entry Entry) LocaleIndex() []LocaleIndexEntry {
	locales := make([]string, 0, len(entry.Locales))
	for locale := range entry.Locales {
		locales = append(locales, locale)
	}
	slices.Sort(locales)
	index := make([]LocaleIndexEntry, 0, len(locales))
	for _, locale := range locales {
		document := entry.Locales[locale]
		index = append(index, LocaleIndexEntry{
			Locale:    locale,
			Status:    document.Status,
			UpdatedAt: document.UpdatedAt,
		})
	}
	return index
}

func localeResolution(requested, served string, fallback bool, document LocaleDocument) LocaleResolution {
	return LocaleResolution{
		Requested: requested,
		Served:    served,
		Fallback:  fallback,
		Data:      cloneDocument(document.Data),
		Status:    document.Status,
	}
}

func cloneLocales(source map[string]LocaleDocument) map[string]LocaleDocument {
	if source == nil {
		return nil
	}
	target := make(map[string]LocaleDocument, len(source))
	for locale, document := range source {
		target[locale] = cloneLocaleDocument(document)
	}
	return target
}

func cloneLocaleDocument(document LocaleDocument) LocaleDocument {
	document.Data = cloneDocument(document.Data)
	return document
}

func cloneDocument(source map[string]any) map[string]any {
	if source == nil {
		return nil
	}
	target := make(map[string]any, len(source))
	for key, value := range source {
		target[key] = cloneValue(value)
	}
	return target
}
