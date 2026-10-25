package canonical

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"time"
)

// Closed-record keys from @ui8kit/contract KEYS_V04 / REQUIRED_V04 (commit 8299fee).
var (
	entryKeys = []string{
		"id", "locale", "kind", "status", "visibility", "slug", "title", "content", "excerpt",
		"author_id", "parent_id", "featured_media_id", "template", "terms", "order", "pinned",
		"format", "discussion", "origin", "metadata", "version", "created_at", "updated_at",
		"published_at", "deleted_at",
	}
	requiredEntry = []string{
		"id", "locale", "kind", "status", "visibility", "slug", "title", "version", "created_at", "updated_at",
	}
	metadataKeys   = []string{"value", "private"}
	termRefKeys    = []string{"taxonomy", "term_id"}
	originKeys     = []string{"system", "id"}
	discussionKeys = []string{"comments", "pings"}
	dateFields     = map[string]struct{}{
		"created_at": {}, "updated_at": {}, "published_at": {}, "deleted_at": {},
	}
	idPattern   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
	rfc3339Date = regexp.MustCompile(`^(\d{4})-(\d{2})-(\d{2})[Tt](\d{2}):(\d{2}):(\d{2})(\.\d+)?([Zz]|([+-])(\d{2}):(\d{2}))$`)
	dateOnly    = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

func isObj(v any) (map[string]any, bool) {
	m, ok := v.(map[string]any)
	return m, ok
}

func own(o map[string]any, k string) bool {
	_, ok := o[k]
	return ok
}

func closed(o map[string]any, keys []string, path string) error {
	allowed := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		allowed[k] = struct{}{}
	}
	for k := range o {
		if _, ok := allowed[k]; !ok {
			return fail(ErrUnknownField, path+"."+k, fmt.Sprintf("unknown field %s", strconv.Quote(k)))
		}
	}
	return nil
}

func isEmpty(v any) bool {
	if v == nil {
		return true
	}
	switch t := v.(type) {
	case string:
		return t == ""
	case []any:
		return len(t) == 0
	case map[string]any:
		return len(t) == 0
	default:
		return false
	}
}

func project(doc any) (map[string]any, error) {
	obj, ok := isObj(doc)
	if !ok {
		return nil, fail(ErrInvalidType, "$", "entry must be a JSON object")
	}
	if err := closed(obj, entryKeys, "$"); err != nil {
		return nil, err
	}
	for _, k := range requiredEntry {
		if !own(obj, k) || obj[k] == nil {
			return nil, fail(ErrMissingField, "$."+k, "required field "+k+" is missing")
		}
	}
	id, _ := obj["id"].(string)
	if _, isStr := obj["id"].(string); !isStr || !idPattern.MatchString(id) {
		return nil, fail(ErrInvalidID, "$.id", "id must match ^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$")
	}

	required := make(map[string]struct{}, len(requiredEntry))
	for _, k := range requiredEntry {
		required[k] = struct{}{}
	}
	out := map[string]any{}
	for k, v := range obj {
		if _, req := required[k]; !req && isEmpty(v) {
			continue
		}
		path := "$." + k
		var err error
		if _, isDate := dateFields[k]; isDate {
			v, err = normalizeDate(v, path)
			if err != nil {
				return nil, err
			}
		} else if k == "metadata" {
			v, err = projectMetadata(v, path)
			if err != nil {
				return nil, err
			}
			if v == nil {
				continue
			}
		} else if k == "terms" {
			v, err = projectTerms(v, path)
			if err != nil {
				return nil, err
			}
		} else if k == "origin" {
			v, err = projectOrigin(v, path)
			if err != nil {
				return nil, err
			}
		} else if k == "discussion" {
			if d, ok := isObj(v); ok {
				if err := closed(d, discussionKeys, path); err != nil {
					return nil, err
				}
			}
		}
		out[k] = v
	}
	return out, nil
}

func projectMetadata(v any, path string) (any, error) {
	md, ok := isObj(v)
	if !ok {
		return nil, fail(ErrInvalidType, path, "metadata must be an object")
	}
	out := map[string]any{}
	for mk, mv := range md {
		item, ok := isObj(mv)
		if !ok || !own(item, "value") {
			return nil, fail(ErrInvalidType, path+"."+mk, "metadata item must be {value, private?}")
		}
		if err := closed(item, metadataKeys, path+"."+mk); err != nil {
			return nil, err
		}
		if own(item, "private") {
			if _, isBool := item["private"].(bool); !isBool {
				return nil, fail(ErrInvalidType, path+"."+mk+".private", "private must be boolean")
			}
		}
		if priv, _ := item["private"].(bool); priv {
			continue
		}
		out[mk] = map[string]any{"value": item["value"]}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
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
		tax, taxOK := item["taxonomy"].(string)
		tid, tidOK := item["term_id"].(string)
		if !taxOK || !tidOK {
			return nil, fail(ErrInvalidType, fmt.Sprintf("%s[%d]", path, n), "term must be {taxonomy, term_id} strings")
		}
		if err := closed(item, termRefKeys, fmt.Sprintf("%s[%d]", path, n)); err != nil {
			return nil, err
		}
		_ = tax
		_ = tid
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

func projectOrigin(v any, path string) (any, error) {
	arr, ok := v.([]any)
	if !ok {
		return nil, fail(ErrInvalidType, path, "origin must be an array")
	}
	for n, o := range arr {
		item, ok := isObj(o)
		if !ok {
			return nil, fail(ErrInvalidType, fmt.Sprintf("%s[%d]", path, n), "origin item must be {system, id}")
		}
		if err := closed(item, originKeys, fmt.Sprintf("%s[%d]", path, n)); err != nil {
			return nil, err
		}
	}
	return arr, nil
}

func normalizeDate(v any, path string) (any, error) {
	s, ok := v.(string)
	if !ok {
		return nil, fail(ErrInvalidDate, path, "date-time must be a string")
	}
	if dateOnly.MatchString(s) {
		return nil, fail(ErrInvalidDate, path, "date without time: "+s)
	}
	m := rfc3339Date.FindStringSubmatch(s)
	if m == nil {
		return nil, fail(ErrInvalidDate, path, "not an RFC 3339 date-time: "+s)
	}
	if m[7] != "" {
		for _, c := range m[7] {
			if c >= '1' && c <= '9' {
				return nil, fail(ErrFractionalSeconds, path, "fractional seconds are not allowed: "+s)
			}
		}
	}
	Y, _ := strconv.Atoi(m[1])
	M, _ := strconv.Atoi(m[2])
	D, _ := strconv.Atoi(m[3])
	h, _ := strconv.Atoi(m[4])
	mi, _ := strconv.Atoi(m[5])
	sec, _ := strconv.Atoi(m[6])
	oh, om := 0, 0
	if m[10] != "" {
		oh, _ = strconv.Atoi(m[10])
		om, _ = strconv.Atoi(m[11])
	}
	dim := time.Date(2000, time.Month(M+1), 0, 0, 0, 0, 0, time.UTC).Day()
	leap := (Y%4 == 0 && Y%100 != 0) || Y%400 == 0
	maxD := dim
	if M == 2 && !leap {
		maxD = 28
	}
	if M < 1 || M > 12 || D < 1 || D > maxD || h > 23 || mi > 59 || sec > 59 || oh > 23 || om > 59 {
		return nil, fail(ErrInvalidDate, path, "date-time out of range: "+s)
	}
	t := time.Date(Y, time.Month(M), D, h, mi, sec, 0, time.UTC)
	off := time.Duration(oh*60+om) * time.Minute
	switch m[9] {
	case "+":
		t = t.Add(-off)
	case "-":
		t = t.Add(off)
	}
	y := t.UTC().Year()
	if y < 0 || y > 9999 {
		return nil, fail(ErrInvalidDate, path, "year out of range after UTC conversion: "+s)
	}
	u := t.UTC()
	return fmt.Sprintf("%04d-%02d-%02dT%02d:%02d:%02dZ", u.Year(), u.Month(), u.Day(), u.Hour(), u.Minute(), u.Second()), nil
}

// Canonicalize returns the codex-entry/v1 JCS text of one Entry v0.4 JSON document.
func Canonicalize(inputText string) (string, error) {
	doc, err := parse(inputText)
	if err != nil {
		return "", err
	}
	projected, err := project(doc)
	if err != nil {
		return "", err
	}
	return jcs(projected), nil
}

// Digest returns "codex-entry/v1:sha256:" plus hex(sha256(UTF-8 bytes of Canonicalize(inputText))).
func Digest(inputText string) (string, error) {
	text, err := Canonicalize(inputText)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(text))
	return EntryDigestPrefix + hex.EncodeToString(sum[:]), nil
}
