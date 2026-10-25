package content

import (
	"regexp"
	"slices"
	"strings"
)

const DefaultLocale = "en"

var localePattern = regexp.MustCompile(`^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$`)

type LocaleResolution struct {
	Requested string `json:"requested"`
	Served    string `json:"served"`
	Fallback  bool   `json:"fallback"`
}

// ValidLocale reports whether locale is a v0.4 BCP 47 tag (e.g. ru, en-US).
// Region case is preserved; this is not a lowercasing normalizer.
func ValidLocale(locale string) bool {
	return localePattern.MatchString(locale)
}

// Pick returns the entry with the given id and requested locale, falling back
// to fallback, then to the lexicographically first locale for that id.
func Pick(entries []Entry, id ID, requested, fallback string) (Entry, LocaleResolution, bool) {
	if requested == "" {
		requested = DefaultLocale
	}
	matches := make([]Entry, 0, 2)
	for _, entry := range entries {
		if entry.ID == id {
			matches = append(matches, entry)
		}
	}
	if len(matches) == 0 {
		return Entry{}, LocaleResolution{Requested: requested, Served: requested}, false
	}
	for _, entry := range matches {
		if entry.Locale == requested {
			return entry.Clone(), LocaleResolution{Requested: requested, Served: requested}, true
		}
	}
	if fallback != "" {
		for _, entry := range matches {
			if entry.Locale == fallback {
				return entry.Clone(), LocaleResolution{Requested: requested, Served: fallback, Fallback: true}, true
			}
		}
	}
	slices.SortFunc(matches, func(left, right Entry) int {
		return strings.Compare(left.Locale, right.Locale)
	})
	served := matches[0]
	return served.Clone(), LocaleResolution{
		Requested: requested,
		Served:    served.Locale,
		Fallback:  served.Locale != requested,
	}, true
}

// Locales returns the sorted locale tags of records that share id.
func Locales(entries []Entry, id ID) []string {
	set := map[string]struct{}{}
	for _, entry := range entries {
		if entry.ID == id && entry.Locale != "" {
			set[entry.Locale] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for locale := range set {
		out = append(out, locale)
	}
	slices.Sort(out)
	return out
}
