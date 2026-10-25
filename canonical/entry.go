package canonical

import (
	"fmt"
	"sort"
)

func handleTerms(v any, path string) (any, bool, error) {
	out, err := projectTerms(v, path)
	return out, false, err
}

func handleDiscussion(v any, path string) (any, bool, error) {
	if d, ok := isObj(v); ok {
		if err := closed(d, discussionKeys, path); err != nil {
			return nil, false, err
		}
	}
	return v, false, nil
}

func projectTerms(v any, path string) (any, error) {
	arr, ok := v.([]any)
	if !ok {
		return nil, fail(ErrInvalidType, path, "terms must be an array")
	}
	out := make([]any, len(arr))
	for n, t := range arr {
		item, ok := isObj(t)
		if !ok {
			return nil, fail(ErrInvalidType, fmt.Sprintf("%s[%d]", path, n), "term must be {taxonomy, term_id} strings")
		}
		_, taxOK := item["taxonomy"].(string)
		_, tidOK := item["term_id"].(string)
		if !taxOK || !tidOK {
			return nil, fail(ErrInvalidType, fmt.Sprintf("%s[%d]", path, n), "term must be {taxonomy, term_id} strings")
		}
		if err := closed(item, termRefKeys, fmt.Sprintf("%s[%d]", path, n)); err != nil {
			return nil, err
		}
		out[n] = item
	}
	sort.SliceStable(out, func(i, j int) bool {
		a := out[i].(map[string]any)
		b := out[j].(map[string]any)
		ta, _ := a["taxonomy"].(string)
		tb, _ := b["taxonomy"].(string)
		if ta != tb {
			return codePointLess(ta, tb)
		}
		ia, _ := a["term_id"].(string)
		ib, _ := b["term_id"].(string)
		return codePointLess(ia, ib)
	})
	return out, nil
}

func entrySpec() recordSpec {
	return recordSpec{
		kind:     "entry",
		keys:     entryKeys,
		required: requiredEntry,
		dates:    entryDates,
		special: map[string]fieldHandler{
			"metadata":   handleMetadata,
			"terms":      handleTerms,
			"origin":     handleOrigin,
			"discussion": handleDiscussion,
		},
	}
}

// Canonicalize returns the codex-entry/v1 JCS text of one Entry v0.4 JSON document.
func Canonicalize(inputText string) (string, error) {
	return canonicalizeRecord(inputText, entrySpec())
}

// Digest returns "codex-entry/v1:sha256:" plus hex(sha256(UTF-8 bytes of Canonicalize(inputText))).
func Digest(inputText string) (string, error) {
	text, err := Canonicalize(inputText)
	if err != nil {
		return "", err
	}
	return digestPrefixed(EntryDigestPrefix, text), nil
}
