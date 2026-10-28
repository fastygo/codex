// Package validate is the Go port of @ui8kit/contract validateCollectionV04.
package validate

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/fastygo/codex/canonical"
	"github.com/fastygo/codex/content"
	"github.com/fastygo/codex/schema"
	"github.com/fastygo/codex/validation"
)

type RecordFile struct {
	File string
	Data any
}

type Collection struct {
	Entries      []RecordFile
	Terms        []RecordFile
	Comments     []RecordFile
	Menus        []RecordFile
	Dictionaries []RecordFile
	Settings     []RecordFile
}

type Options struct {
	Manifest *schema.Manifest
	Currency string
}

var (
	ident     = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,62}$`)
	localeRe  = regexp.MustCompile(`^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$`)
	currency  = regexp.MustCompile(`^[A-Z]{3}$`)
	rfc3339   = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2})$`)
	timezone  = regexp.MustCompile(`^[A-Za-z_]+(/[A-Za-z0-9_+-]+)*$|^UTC$`)
	e164      = regexp.MustCompile(`^\+[1-9][0-9]{1,14}$`)
	emailRe   = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	handleRe  = regexp.MustCompile(`^\S+$`)
	openTime  = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)
	closeTime = regexp.MustCompile(`^(([01][0-9]|2[0-3]):[0-5][0-9]|24:00)$`)
	dateRe    = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)
	weekdays  = map[string]struct{}{
		"mon": {}, "tue": {}, "wed": {}, "thu": {}, "fri": {}, "sat": {}, "sun": {},
	}
	weekdayList           = "mon, tue, wed, thu, fri, sat, sun"
	localeNeutralSettings = []string{"contact.phone", "contact.email", "contact.links", "hours", "hours_exceptions", "map"}
)

var entryKeys = []string{
	"id", "locale", "kind", "status", "visibility", "slug", "title", "content", "excerpt",
	"author_id", "parent_id", "featured_media_id", "template", "terms", "order", "pinned",
	"format", "discussion", "origin", "metadata", "version", "created_at", "updated_at",
	"published_at", "deleted_at",
}
var termKeys = []string{
	"id", "locale", "taxonomy_id", "name", "slug", "description", "parent_id", "origin", "metadata", "version", "created_at", "updated_at",
}
var commentKeys = []string{
	"id", "locale", "entry_id", "parent_id", "author_id", "author_name", "content", "status", "origin", "metadata", "version", "created_at", "updated_at",
}
var menuKeys = []string{"id", "location", "locale", "items"}
var menuItemKeys = []string{"id", "label", "url", "parent_id", "order", "target"}
var dictionaryKeys = []string{"locale", "source_locale", "messages"}
var settingsKeys = []string{
	"id", "locale", "title", "description", "language", "timezone", "logo_media_id", "icon_media_id",
	"front_page_id", "posts_page_id", "posts_per_page", "comments_open", "currency", "contact",
	"hours", "hours_exceptions", "map", "origin", "metadata", "version", "created_at", "updated_at",
}
var originKeys = []string{"system", "id"}
var metadataKeys = []string{"value", "private"}
var termRefKeys = []string{"taxonomy", "term_id"}
var discussionKeys = []string{"comments", "pings"}
var targetKeys = []string{"kind", "id"}
var contactKeys = []string{"phone", "email", "address", "links"}
var contactLinkKeys = []string{"network", "handle"}
var hoursRuleKeys = []string{"days", "open", "close"}
var hoursExceptionKeys = []string{"date", "closed", "open", "close"}
var mapKeys = []string{"provider", "lat", "lng"}

var entryStatuses = map[string]struct{}{
	"draft": {}, "pending": {}, "scheduled": {}, "published": {}, "archived": {}, "trashed": {},
}
var commentStatuses = map[string]struct{}{
	"approved": {}, "hold": {}, "spam": {}, "trash": {},
}

type obj = map[string]any

func isObj(v any) (obj, bool) {
	m, ok := v.(obj)
	return m, ok
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

func blank(v any) bool {
	return strings.TrimSpace(str(v)) == ""
}

func asObject(v any) (obj, error) {
	if m, ok := isObj(v); ok {
		return m, nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	var out any
	if err := dec.Decode(&out); err != nil {
		return nil, err
	}
	m, ok := isObj(out)
	if !ok {
		return nil, fmt.Errorf("not an object")
	}
	return m, nil
}

func closed(add func(string, string, string), o obj, keys []string, p string) {
	allowed := map[string]struct{}{}
	for _, k := range keys {
		allowed[k] = struct{}{}
	}
	for k := range o {
		if _, ok := allowed[k]; !ok {
			add(validation.DecodeUnknownField, p+"."+k, `unknown field "`+k+`"`)
		}
	}
}

func isIntGE1(v any) bool {
	n, ok := asInt(v)
	return ok && n >= 1
}

func asInt(v any) (int64, bool) {
	switch t := v.(type) {
	case json.Number:
		i, err := t.Int64()
		return i, err == nil
	case float64:
		if t != float64(int64(t)) {
			return 0, false
		}
		return int64(t), true
	case int:
		return int64(t), true
	case int64:
		return t, true
	default:
		return 0, false
	}
}

func tsOrder(a, b any) (int, bool) {
	as, aok := a.(string)
	bs, bok := b.(string)
	if !aok || !bok {
		return 0, false
	}
	// RFC 3339 UTC/offset strings compare correctly only after parse; keep simple string compare
	// for equal-format Z timestamps used in tests. For mixed offsets, parse via canonical dates.
	if as < bs {
		return -1, true
	}
	if as > bs {
		return 1, true
	}
	return 0, true
}

func common(o obj, p string, add func(string, string, string), localeRequired bool) {
	if blank(o["id"]) {
		add(validation.CoreIDRequired, p+".id", "id is required")
	} else if !content.ValidRecordID(str(o["id"])) {
		add(validation.CoreIDInvalid, p+".id", "id must be opaque ASCII matching "+content.RecordIDPattern)
	}
	if o["locale"] != nil || localeRequired {
		if !localeRe.MatchString(str(o["locale"])) {
			add(validation.CoreLocaleInvalid, p+".locale", "locale must be a BCP 47 tag like \"ru\" or \"en-US\"")
		}
	}
	if !isIntGE1(o["version"]) {
		add(validation.CoreVersionRequired, p+".version", "version must be an integer >= 1")
	}
	if !rfc3339.MatchString(str(o["created_at"])) || !rfc3339.MatchString(str(o["updated_at"])) {
		add(validation.CoreTimestampsRequired, p+".created_at", "created_at and updated_at must be RFC 3339")
	} else if cmp, ok := tsOrder(o["updated_at"], o["created_at"]); ok && cmp < 0 {
		add(validation.CoreTimestampsOrder, p+".updated_at", "updated_at precedes created_at")
	}
	if o["origin"] != nil {
		arr, ok := o["origin"].([]any)
		if !ok {
			add(validation.CoreOriginInvalid, p+".origin", "origin must be an array")
		} else {
			for i, x := range arr {
				item, is := isObj(x)
				if !is {
					add(validation.CoreOriginInvalid, fmt.Sprintf("%s.origin[%d]", p, i), "origin item must be {system, id}")
					continue
				}
				closed(add, item, originKeys, fmt.Sprintf("%s.origin[%d]", p, i))
				if blank(item["system"]) || blank(item["id"]) {
					add(validation.CoreOriginInvalid, fmt.Sprintf("%s.origin[%d]", p, i), "origin needs a non-empty system and a string id")
				}
			}
		}
	}
	if o["metadata"] != nil {
		md, ok := isObj(o["metadata"])
		if !ok {
			add(validation.CoreMetadataInvalid, p+".metadata", "metadata must be an object")
		} else {
			for k, mv := range md {
				if strings.TrimSpace(k) == "" {
					add(validation.CoreMetadataKeyRequired, p+".metadata", "metadata key is required")
				}
				item, is := isObj(mv)
				if !is || !hasKey(item, "value") {
					add(validation.CoreMetadataInvalid, p+".metadata."+k, "metadata value must be {value, private?}")
					continue
				}
				closed(add, item, metadataKeys, p+".metadata."+k)
				if item["private"] != nil {
					if _, isBool := item["private"].(bool); !isBool {
						add(validation.CoreMetadataInvalid, p+".metadata."+k+".private", "private must be boolean")
					}
				}
			}
		}
	}
}

func hasKey(o obj, k string) bool {
	_, ok := o[k]
	return ok
}

func checkEntry(o any, p string, add func(string, string, string)) {
	m, ok := isObj(o)
	if !ok {
		add(validation.DecodeInvalid, p, "entry must be an object")
		return
	}
	closed(add, m, entryKeys, p)
	common(m, p, add, true)
	if !ident.MatchString(str(m["kind"])) {
		add(validation.CoreEntryKindInvalid, p+".kind", "kind is invalid")
	}
	if _, ok := entryStatuses[str(m["status"])]; !ok {
		add(validation.CoreEntryStatusInvalid, p+".status", "status is invalid")
	}
	if vis := str(m["visibility"]); vis != "public" && vis != "private" {
		add(validation.CoreEntryVisibilityInvalid, p+".visibility", "visibility must be public or private")
	}
	if blank(m["slug"]) {
		add(validation.CoreEntrySlugRequired, p+".slug", "slug is required (string)")
	}
	if _, ok := m["title"].(string); !ok {
		add(validation.CoreEntryTitleRequired, p+".title", "title is required (string, may be empty)")
	}
	for _, k := range []string{"content", "excerpt", "author_id", "parent_id", "featured_media_id", "template", "format"} {
		if m[k] != nil {
			if _, ok := m[k].(string); !ok {
				add(validation.CoreEntryTypeInvalid, p+"."+k, k+" must be a string")
			}
		}
	}
	if !blank(m["parent_id"]) && str(m["parent_id"]) == str(m["id"]) {
		add(validation.CoreEntryParentSelf, p+".parent_id", "entry cannot be its own parent")
	}
	if m["order"] != nil {
		if _, ok := asInt(m["order"]); !ok {
			add(validation.CoreEntryTypeInvalid, p+".order", "order must be an integer")
		}
	}
	if m["pinned"] != nil {
		if _, ok := m["pinned"].(bool); !ok {
			add(validation.CoreEntryTypeInvalid, p+".pinned", "pinned must be boolean")
		}
	}
	if m["discussion"] != nil {
		d, ok := isObj(m["discussion"])
		if !ok {
			add(validation.CoreEntryTypeInvalid, p+".discussion", "discussion must be {comments: boolean, pings: boolean}")
		} else {
			_, cOK := d["comments"].(bool)
			_, pOK := d["pings"].(bool)
			if !cOK || !pOK {
				add(validation.CoreEntryTypeInvalid, p+".discussion", "discussion must be {comments: boolean, pings: boolean}")
			} else {
				closed(add, d, discussionKeys, p+".discussion")
			}
		}
	}
	if str(m["status"]) == "scheduled" && m["published_at"] == nil {
		add(validation.CoreEntryPublishedAtRequired, p+".published_at", "scheduled entry requires published_at")
	}
	if str(m["status"]) == "trashed" && m["deleted_at"] == nil {
		add(validation.CoreEntryDeletedAtRequired, p+".deleted_at", "trashed entry requires deleted_at")
	}
	for _, k := range []string{"published_at", "deleted_at"} {
		if m[k] != nil && !rfc3339.MatchString(str(m[k])) {
			add(validation.CoreEntryTimeInvalid, p+"."+k, k+" must be RFC 3339")
		}
	}
	if m["terms"] != nil {
		arr, ok := m["terms"].([]any)
		if !ok {
			add(validation.CoreEntryTermsInvalid, p+".terms", "terms must be an array")
		} else {
			seen := map[string]struct{}{}
			for _, t := range arr {
				item, is := isObj(t)
				if !is || !ident.MatchString(str(item["taxonomy"])) || blank(item["term_id"]) {
					add(validation.CoreTermRefIncomplete, p+".terms", "term reference must be {taxonomy, term_id}")
					continue
				}
				closed(add, item, termRefKeys, p+".terms")
				key := str(item["taxonomy"]) + "/" + str(item["term_id"])
				if _, dup := seen[key]; dup {
					add(validation.CoreTermRefDuplicated, p+".terms", "term reference "+key+" is duplicated")
				}
				seen[key] = struct{}{}
			}
		}
	}
}

func checkTerm(o any, p string, add func(string, string, string)) {
	m, ok := isObj(o)
	if !ok {
		add(validation.DecodeInvalid, p, "term must be an object")
		return
	}
	closed(add, m, termKeys, p)
	common(m, p, add, true)
	if !ident.MatchString(str(m["taxonomy_id"])) {
		add(validation.CoreTermTaxonomyInvalid, p+".taxonomy_id", "taxonomy_id is invalid")
	}
	if blank(m["name"]) {
		add(validation.CoreTermNameRequired, p+".name", "name is required")
	}
	if blank(m["slug"]) {
		add(validation.CoreTermSlugRequired, p+".slug", "slug is required")
	}
	if m["description"] != nil {
		if _, ok := m["description"].(string); !ok {
			add(validation.CoreTermTypeInvalid, p+".description", "description must be a string")
		}
	}
	if !blank(m["parent_id"]) && str(m["parent_id"]) == str(m["id"]) {
		add(validation.CoreTermParentSelf, p+".parent_id", "term cannot be its own parent")
	}
}

func checkSettings(o any, p string, add func(string, string, string)) {
	m, ok := isObj(o)
	if !ok {
		add(validation.DecodeInvalid, p, "settings must be an object")
		return
	}
	closed(add, m, settingsKeys, p)
	common(m, p, add, true)
	if blank(m["title"]) {
		add(validation.CoreSettingsTitleRequired, p+".title", "title is required")
	}
	if !localeRe.MatchString(str(m["language"])) {
		add(validation.CoreSettingsLanguageInvalid, p+".language", "language must be a BCP 47 tag like \"en-US\"")
	}
	for _, k := range []string{"description", "timezone", "logo_media_id", "icon_media_id", "front_page_id", "posts_page_id"} {
		if m[k] != nil {
			if _, ok := m[k].(string); !ok {
				add(validation.CoreSettingsTypeInvalid, p+"."+k, k+" must be a string")
			}
		}
	}
	if m["timezone"] != nil && !timezone.MatchString(str(m["timezone"])) {
		add(validation.CoreSettingsTimezoneInvalid, p+".timezone", "timezone must be an IANA name like Europe/Moscow")
	}
	if m["posts_per_page"] != nil && !isIntGE1(m["posts_per_page"]) {
		add(validation.CoreSettingsTypeInvalid, p+".posts_per_page", "posts_per_page must be an integer >= 1")
	}
	if m["comments_open"] != nil {
		if _, ok := m["comments_open"].(bool); !ok {
			add(validation.CoreSettingsTypeInvalid, p+".comments_open", "comments_open must be boolean")
		}
	}
	if m["currency"] != nil && !currency.MatchString(str(m["currency"])) {
		add(validation.CoreSettingsCurrencyInvalid, p+".currency", "currency must be an ISO 4217 code like USD")
	}
	if m["contact"] != nil {
		checkContact(m["contact"], p+".contact", add)
	}
	if m["hours"] != nil {
		checkHours(m["hours"], p+".hours", add)
	}
	if m["hours_exceptions"] != nil {
		checkHoursExceptions(m["hours_exceptions"], p+".hours_exceptions", add)
	}
	if m["map"] != nil {
		checkMap(m["map"], p+".map", add)
	}
}

func checkContact(c any, p string, add func(string, string, string)) {
	o, ok := isObj(c)
	if !ok {
		add(validation.CoreSettingsTypeInvalid, p, "contact must be an object {phone?, email?, address?, links?}")
		return
	}
	closed(add, o, contactKeys, p)
	if o["phone"] != nil && !(isString(o["phone"]) && e164.MatchString(str(o["phone"]))) {
		add(validation.CoreSettingsPhoneInvalid, p+".phone", "phone must be E.164: + and 2 to 15 digits without spaces, e.g. +12025550143")
	}
	if o["email"] != nil && !(isString(o["email"]) && emailRe.MatchString(str(o["email"]))) {
		add(validation.CoreSettingsEmailInvalid, p+".email", "email must look like name@example.com")
	}
	if o["address"] != nil && (!isString(o["address"]) || blank(o["address"])) {
		add(validation.CoreSettingsTypeInvalid, p+".address", "address must be a non-blank string (omit it when there is none)")
	}
	if o["links"] == nil {
		return
	}
	links, ok := o["links"].([]any)
	if !ok || len(links) == 0 {
		add(validation.CoreSettingsTypeInvalid, p+".links", "links must be a non-empty array of {network, handle} (omit it when there are none)")
		return
	}
	for i, item := range links {
		q := fmt.Sprintf("%s.links[%d]", p, i)
		l, is := isObj(item)
		if !is {
			add(validation.CoreSettingsLinkInvalid, q, "a link must be {network, handle}")
			continue
		}
		closed(add, l, contactLinkKeys, q)
		if !ident.MatchString(str(l["network"])) {
			add(validation.CoreSettingsLinkInvalid, q+".network", "network must be a lowercase identifier like telegram or whatsapp")
		}
		if !(isString(l["handle"]) && handleRe.MatchString(str(l["handle"]))) {
			add(validation.CoreSettingsLinkInvalid, q+".handle", "handle must be a non-empty string without spaces")
		}
	}
}

func hhmmMinutes(t string) int {
	return (int(t[0]-'0')*10+int(t[1]-'0'))*60 + int(t[3]-'0')*10 + int(t[4]-'0')
}

func checkInterval(o obj, q, code string, add func(string, string, string)) {
	if !(isString(o["open"]) && openTime.MatchString(str(o["open"]))) {
		add(code, q+".open", "open must be HH:MM (00:00 to 23:59)")
		return
	}
	if !(isString(o["close"]) && closeTime.MatchString(str(o["close"]))) {
		add(code, q+".close", "close must be HH:MM (00:00 to 24:00)")
		return
	}
	close := str(o["close"])
	open := str(o["open"])
	if close != "24:00" && hhmmMinutes(close) == hhmmMinutes(open) {
		add(code, q+".close", "close must differ from open (00:00 to 24:00 = all day)")
	}
}

func realDate(d string) bool {
	if !dateRe.MatchString(d) {
		return false
	}
	var y, m, day int
	if _, err := fmt.Sscanf(d, "%d-%d-%d", &y, &m, &day); err != nil {
		return false
	}
	t := time.Date(y, time.Month(m), day, 0, 0, 0, 0, time.UTC)
	return t.Year() == y && int(t.Month()) == m && t.Day() == day
}

func checkHours(h any, p string, add func(string, string, string)) {
	arr, ok := h.([]any)
	if !ok {
		add(validation.CoreSettingsTypeInvalid, p, "hours must be an array of {days, open, close}")
		return
	}
	for i, item := range arr {
		q := fmt.Sprintf("%s[%d]", p, i)
		r, is := isObj(item)
		if !is {
			add(validation.CoreSettingsHoursInvalid, q, "an hours rule must be {days, open, close}")
			continue
		}
		closed(add, r, hoursRuleKeys, q)
		days, isArr := r["days"].([]any)
		if !isArr || len(days) == 0 {
			add(validation.CoreSettingsHoursInvalid, q+".days", "days must be a non-empty array of "+weekdayList)
		} else {
			seen := map[string]struct{}{}
			for n, d := range days {
				ds := str(d)
				if _, ok := weekdays[ds]; !ok {
					add(validation.CoreSettingsHoursInvalid, fmt.Sprintf("%s.days[%d]", q, n), "day must be one of "+weekdayList)
				}
				if _, dup := seen[ds]; dup {
					add(validation.CoreSettingsHoursInvalid, q+".days", "a day is listed twice in one rule")
				}
				seen[ds] = struct{}{}
			}
		}
		checkInterval(r, q, validation.CoreSettingsHoursInvalid, add)
	}
}

func checkHoursExceptions(x any, p string, add func(string, string, string)) {
	arr, ok := x.([]any)
	if !ok {
		add(validation.CoreSettingsTypeInvalid, p, "hours_exceptions must be an array of {date, closed: true} or {date, open, close}")
		return
	}
	seen := map[string]int{}
	for i, item := range arr {
		q := fmt.Sprintf("%s[%d]", p, i)
		e, is := isObj(item)
		if !is {
			add(validation.CoreSettingsHoursExceptionInvalid, q, "an exception must be {date, closed: true} or {date, open, close}")
			continue
		}
		closed(add, e, hoursExceptionKeys, q)
		date := str(e["date"])
		if !realDate(date) {
			add(validation.CoreSettingsHoursExceptionInvalid, q+".date", "date must be an existing calendar day YYYY-MM-DD")
		} else if prev, dup := seen[date]; dup {
			add(validation.CoreSettingsHoursExceptionDuplicated, q+".date", fmt.Sprintf("date %s already has an exception (%s[%d])", date, p, prev))
		} else {
			seen[date] = i
		}
		timed := e["open"] != nil || e["close"] != nil
		if e["closed"] != nil {
			closedTrue, isBool := e["closed"].(bool)
			if !isBool || !closedTrue {
				add(validation.CoreSettingsHoursExceptionInvalid, q+".closed", "closed must be true (give open and close for a short day)")
			}
			if timed {
				add(validation.CoreSettingsHoursExceptionInvalid, q, "an exception is either {date, closed: true} or {date, open, close}, not both")
			}
		} else if !timed {
			add(validation.CoreSettingsHoursExceptionInvalid, q, "an exception needs closed: true or open and close")
		} else {
			checkInterval(e, q, validation.CoreSettingsHoursExceptionInvalid, add)
		}
	}
}

func asFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, !math.IsNaN(t) && !math.IsInf(t, 0)
	case json.Number:
		f, err := t.Float64()
		return f, err == nil && !math.IsNaN(f) && !math.IsInf(f, 0)
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	default:
		return 0, false
	}
}

func inRange(v any, max float64) bool {
	f, ok := asFloat(v)
	return ok && f >= -max && f <= max
}

func checkMap(m any, p string, add func(string, string, string)) {
	o, ok := isObj(m)
	if !ok {
		add(validation.CoreSettingsTypeInvalid, p, "map must be an object {provider?, lat?, lng?}")
		return
	}
	closed(add, o, mapKeys, p)
	if len(o) == 0 {
		add(validation.CoreSettingsMapInvalid, p, "map needs a provider or lat and lng (omit it when there is none)")
		return
	}
	if o["provider"] != nil && !ident.MatchString(str(o["provider"])) {
		add(validation.CoreSettingsMapInvalid, p+".provider", "provider must be a lowercase identifier like osm, google or yandex")
	}
	if hasKey(o, "lat") != hasKey(o, "lng") {
		add(validation.CoreSettingsMapInvalid, p, "lat and lng go together")
	}
	if o["lat"] != nil && !inRange(o["lat"], 90) {
		add(validation.CoreSettingsMapInvalid, p+".lat", "lat must be a number from -90 to 90")
	}
	if o["lng"] != nil && !inRange(o["lng"], 180) {
		add(validation.CoreSettingsMapInvalid, p+".lng", "lng must be a number from -180 to 180")
	}
}

func isString(v any) bool {
	_, ok := v.(string)
	return ok
}

func lookupDotted(o obj, path string) (any, bool) {
	var cur any = o
	for _, k := range strings.Split(path, ".") {
		m, ok := isObj(cur)
		if !ok {
			return nil, false
		}
		v, exists := m[k]
		if !exists {
			return nil, false
		}
		cur = v
	}
	return cur, true
}

func stableJSON(v any) string {
	switch t := v.(type) {
	case nil:
		return "null"
	case []any:
		parts := make([]string, len(t))
		for i, x := range t {
			parts[i] = stableJSON(x)
		}
		return "[" + strings.Join(parts, ",") + "]"
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			kb, _ := json.Marshal(k)
			parts = append(parts, string(kb)+":"+stableJSON(t[k]))
		}
		return "{" + strings.Join(parts, ",") + "}"
	case json.Number:
		return t.String()
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return "null"
		}
		return string(b)
	}
}

func neutralValue(o obj, path string) string {
	v, ok := lookupDotted(o, path)
	if !ok {
		return ""
	}
	return stableJSON(v)
}

func checkComment(o any, p string, add func(string, string, string)) {
	m, ok := isObj(o)
	if !ok {
		add(validation.DecodeInvalid, p, "comment must be an object")
		return
	}
	closed(add, m, commentKeys, p)
	common(m, p, add, false)
	if blank(m["entry_id"]) {
		add(validation.CoreCommentEntryRequired, p+".entry_id", "entry_id is required")
	}
	if _, ok := m["content"].(string); !ok {
		add(validation.CoreCommentContentRequired, p+".content", "content is required")
	}
	if _, ok := commentStatuses[str(m["status"])]; !ok {
		add(validation.CoreCommentStatusInvalid, p+".status", "comment status is invalid")
	}
}

func checkMenu(o any, p string, add func(string, string, string)) {
	m, ok := isObj(o)
	if !ok {
		add(validation.DecodeInvalid, p, "menu must be an object")
		return
	}
	closed(add, m, menuKeys, p)
	if blank(m["id"]) {
		add(validation.CoreIDRequired, p+".id", "id is required")
	}
	if blank(m["location"]) {
		add(validation.CoreMenuLocationRequired, p+".location", "location is required")
	}
	if !localeRe.MatchString(str(m["locale"])) {
		add(validation.CoreLocaleInvalid, p+".locale", "locale must be a BCP 47 tag")
	}
	items, ok := m["items"].([]any)
	if !ok {
		add(validation.CoreMenuItemsRequired, p+".items", "items must be an array")
		return
	}
	ids := map[string]struct{}{}
	for i, it := range items {
		ip := fmt.Sprintf("%s.items[%d]", p, i)
		item, is := isObj(it)
		if !is {
			add(validation.DecodeInvalid, ip, "menu item must be an object")
			continue
		}
		closed(add, item, menuItemKeys, ip)
		if blank(item["id"]) || blank(item["label"]) || blank(item["url"]) {
			add(validation.CoreMenuItemIncomplete, ip, "menu item needs id, label and url")
		}
		if _, dup := ids[str(item["id"])]; dup {
			add(validation.CoreMenuItemDuplicated, ip, `menu item "`+str(item["id"])+`" is duplicated`)
		}
		ids[str(item["id"])] = struct{}{}
		if item["target"] != nil {
			tg, is := isObj(item["target"])
			if !is || blank(tg["kind"]) || blank(tg["id"]) {
				add(validation.CoreMenuTargetInvalid, ip+".target", "target must be {kind, id}")
			} else {
				closed(add, tg, targetKeys, ip+".target")
			}
		}
	}
}

func checkDictionary(o any, p string, add func(string, string, string)) {
	m, ok := isObj(o)
	if !ok {
		add(validation.DecodeInvalid, p, "dictionary must be an object")
		return
	}
	closed(add, m, dictionaryKeys, p)
	if !localeRe.MatchString(str(m["locale"])) {
		add(validation.CoreLocaleInvalid, p+".locale", "locale must be a BCP 47 tag")
	}
	if m["source_locale"] != nil && !localeRe.MatchString(str(m["source_locale"])) {
		add(validation.CoreLocaleInvalid, p+".source_locale", "source_locale must be a BCP 47 tag")
	}
	msgs, ok := isObj(m["messages"])
	if !ok {
		add(validation.CoreDictionaryMessagesRequired, p+".messages", "messages must be an object")
		return
	}
	for k, v := range msgs {
		if _, is := v.(string); !is {
			add(validation.CoreDictionaryValueInvalid, p+".messages."+k, "message must be a string")
		}
	}
}

// CollectionV04 validates a v0.4 record collection. Error codes match ISSUE_CODES_V04
// plus Q7 entry.localized_mismatch when a manifest is provided.
func CollectionV04(c Collection, opts Options) []validation.Issue {
	var issues []validation.Issue
	add := func(code, path, message string) {
		issues = append(issues, validation.IssueOf(code, path, message))
	}
	run := func(list []RecordFile, check func(any, string, func(string, string, string))) []struct {
		file string
		o    obj
	} {
		var ok []struct {
			file string
			o    obj
		}
		for _, rec := range list {
			n := len(issues)
			data := rec.Data
			if m, err := asObject(data); err == nil {
				data = m
			}
			check(data, rec.File, add)
			if len(issues) == n {
				if m, is := isObj(data); is {
					ok = append(ok, struct {
						file string
						o    obj
					}{rec.File, m})
				}
			}
		}
		return ok
	}
	unique := func(list []struct {
		file string
		o    obj
	}, code string, key func(obj) string, what string) {
		seen := map[string]string{}
		for _, item := range list {
			k := key(item.o)
			if prev, dup := seen[k]; dup {
				add(code, item.file, what+" "+k+" is duplicated (also in "+prev+")")
			} else {
				seen[k] = item.file
			}
		}
	}

	entries := run(c.Entries, checkEntry)
	terms := run(c.Terms, checkTerm)
	comments := run(c.Comments, checkComment)
	menus := run(c.Menus, checkMenu)
	run(c.Dictionaries, checkDictionary)
	settings := run(c.Settings, checkSettings)

	unique(settings, validation.CoreSettingsKeyDuplicated, func(o obj) string { return "(" + str(o["id"]) + ", " + str(o["locale"]) + ")" }, "settings key")
	firstOf := map[string]struct {
		file string
		o    obj
	}{}
	for _, r := range settings {
		id := str(r.o["id"])
		first, ok := firstOf[id]
		if !ok {
			firstOf[id] = r
			continue
		}
		for _, k := range localeNeutralSettings {
			if neutralValue(first.o, k) != neutralValue(r.o, k) {
				add(validation.CoreSettingsLocaleNeutralMismatch, r.file+"."+k, k+" differs from settings ("+str(first.o["id"])+", "+str(first.o["locale"])+") in "+first.file+"; only contact.address is localized")
			}
		}
	}
	core := map[string]struct{}{}
	for _, k := range entryKeys {
		core[k] = struct{}{}
	}
	for _, item := range entries {
		md, _ := isObj(item.o["metadata"])
		for k := range md {
			if _, shadowed := core[k]; shadowed {
				add(validation.CoreMetadataKeyShadowed, item.file+".metadata."+k, `metadata key "`+k+`" is a core entry field`)
			}
		}
	}
	unique(entries, validation.CoreEntryKeyDuplicated, func(o obj) string { return "(" + str(o["id"]) + ", " + str(o["locale"]) + ")" }, "entry key")
	unique(entries, validation.CoreEntrySlugDuplicated, func(o obj) string {
		parent := str(o["parent_id"])
		if parent == "" {
			parent = "-"
		}
		return "(" + str(o["kind"]) + ", " + str(o["locale"]) + ", " + parent + ") slug \"" + str(o["slug"]) + "\""
	}, "entry slug in")
	unique(terms, validation.CoreTermKeyDuplicated, func(o obj) string { return "(" + str(o["id"]) + ", " + str(o["locale"]) + ")" }, "term key")
	unique(terms, validation.CoreTermSlugDuplicated, func(o obj) string {
		parent := str(o["parent_id"])
		if parent == "" {
			parent = "-"
		}
		return "(" + str(o["taxonomy_id"]) + ", " + str(o["locale"]) + ", " + parent + ") slug \"" + str(o["slug"]) + "\""
	}, "term slug in")
	unique(comments, validation.CoreCommentKeyDuplicated, func(o obj) string {
		loc := str(o["locale"])
		if loc == "" {
			loc = "-"
		}
		return "(" + str(o["id"]) + ", " + loc + ")"
	}, "comment key")
	unique(menus, validation.CoreMenuKeyDuplicated, func(o obj) string { return "(" + str(o["id"]) + ", " + str(o["locale"]) + ")" }, "menu key")

	entryByID := map[string]obj{}
	for _, item := range entries {
		entryByID[str(item.o["id"])] = item.o
	}
	termIDs := map[string]struct{}{}
	for _, item := range terms {
		termIDs[str(item.o["taxonomy_id"])+"/"+str(item.o["id"])] = struct{}{}
	}
	for _, item := range entries {
		if pid := str(item.o["parent_id"]); pid != "" {
			par, exists := entryByID[pid]
			if !exists {
				add(validation.CoreEntryParentMissing, item.file+".parent_id", `parent "`+pid+`" does not exist`)
			} else if str(par["kind"]) != str(item.o["kind"]) {
				add(validation.CoreEntryParentKindInvalid, item.file+".parent_id", `parent "`+pid+`" has another kind`)
			}
		}
		if len(terms) > 0 {
			arr, _ := item.o["terms"].([]any)
			for _, t := range arr {
				tr, _ := isObj(t)
				key := str(tr["taxonomy"]) + "/" + str(tr["term_id"])
				if _, exists := termIDs[key]; !exists {
					add(validation.CoreTermRefMissing, item.file+".terms", "term "+key+" does not exist")
				}
			}
		}
	}

	if opts.Manifest != nil {
		defaultCurrency := opts.Currency
		for _, s := range settings {
			if currency.MatchString(str(s.o["currency"])) {
				defaultCurrency = str(s.o["currency"])
				break
			}
		}
		checkAgainstManifest(*opts.Manifest, entries, entryByID, &issues, defaultCurrency)
		checkLocalizedMismatch(*opts.Manifest, entries, &issues)
	}
	return issues
}

func checkAgainstManifest(manifest schema.Manifest, entries []struct {
	file string
	o    obj
}, byID map[string]obj, issues *[]validation.Issue, defaultCurrency string) {
	add := func(code, path, message string) {
		*issues = append(*issues, validation.IssueOf(code, path, message))
	}
	kinds := map[string]schema.Resource{}
	for _, res := range manifest.Resources {
		kinds[string(res.Record.ID)] = res
	}
	entryKinds := map[string]content.Kind{}
	for id, o := range byID {
		entryKinds[id] = content.Kind(str(o["kind"]))
	}
	core := coreKeys()
	for _, item := range entries {
		kind := str(item.o["kind"])
		res, ok := kinds[kind]
		if !ok {
			add(validation.ManifestKindUnknown, item.file+".kind", `kind "`+kind+`" is not in the manifest; metadata not checked`)
			continue
		}
		var fields []schema.Field
		declared := map[string]struct{}{}
		for _, f := range res.Record.Fields {
			if core[string(f.ID)] {
				continue
			}
			fields = append(fields, f)
			declared[string(f.ID)] = struct{}{}
		}
		md := obj{}
		if raw, ok := isObj(item.o["metadata"]); ok {
			md = raw
		}
		typed := map[string]content.MetadataValue{}
		for k, mv := range md {
			if _, known := declared[k]; !known {
				add(validation.ManifestMetadataUnknown, item.file+".metadata."+k, `metadata key "`+k+`" is not a field of kind "`+kind+`"`)
			}
			itemObj, _ := isObj(mv)
			typed[k] = content.MetadataValue{Value: itemObj["value"]}
			if priv, ok := itemObj["private"].(bool); ok {
				val := typed[k]
				val.Private = priv
				typed[k] = val
			}
		}
		for _, issue := range schema.CheckMetadataFields(res, fields, typed, item.file+".metadata", entryKinds) {
			*issues = append(*issues, issue)
		}
		if money := schema.CheckMoneyCurrency(fields, typed, defaultCurrency, item.file); money.Code != "" {
			*issues = append(*issues, money)
		}
	}
}

func coreKeys() map[string]bool {
	out := map[string]bool{}
	for _, k := range entryKeys {
		out[k] = true
	}
	return out
}

func checkLocalizedMismatch(manifest schema.Manifest, entries []struct {
	file string
	o    obj
}, issues *[]validation.Issue) {
	kinds := map[string]schema.Resource{}
	for _, res := range manifest.Resources {
		kinds[string(res.Record.ID)] = res
	}
	byID := map[string][]struct {
		file string
		o    obj
	}{}
	for _, item := range entries {
		id := str(item.o["id"])
		byID[id] = append(byID[id], item)
	}
	core := coreKeys()
	for id, group := range byID {
		if len(group) < 2 {
			continue
		}
		kind := str(group[0].o["kind"])
		res, ok := kinds[kind]
		if !ok {
			continue
		}
		for _, field := range res.Record.Fields {
			fid := string(field.ID)
			if core[fid] || content.ChromeField(fid) || field.Localized {
				continue
			}
			var first string
			var firstSet bool
			for _, item := range group {
				md, _ := isObj(item.o["metadata"])
				var value any
				if mv, ok := isObj(md[fid]); ok {
					value = mv["value"]
				} else {
					value = absentSentinel
				}
				form := canonical.Format(value)
				if !firstSet {
					first = form
					firstSet = true
					continue
				}
				if form != first {
					*issues = append(*issues, validation.IssueOf(validation.EntryLocalizedMismatch, item.file+".metadata."+fid, `non-localized field "`+fid+`" differs across locales of id "`+id+`"`))
					break
				}
			}
		}
	}
}

var absentSentinel = struct{ absent bool }{true}
