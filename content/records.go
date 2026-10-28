package content

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/fastygo/codex/validation"
)

type CommentStatus string

const (
	CommentApproved CommentStatus = "approved"
	CommentHold     CommentStatus = "hold"
	CommentSpam     CommentStatus = "spam"
	CommentTrash    CommentStatus = "trash"
)

func (status CommentStatus) Valid() bool {
	switch status {
	case CommentApproved, CommentHold, CommentSpam, CommentTrash:
		return true
	default:
		return false
	}
}

// Comment is a v0.4 comment record. Statuses match WordPress core names.
type Comment struct {
	ID         ID                       `json:"id"`
	Locale     string                   `json:"locale,omitempty"`
	EntryID    ID                       `json:"entry_id"`
	ParentID   ID                       `json:"parent_id,omitempty"`
	AuthorID   string                   `json:"author_id,omitempty"`
	AuthorName string                   `json:"author_name,omitempty"`
	Content    string                   `json:"content"`
	Status     CommentStatus            `json:"status"`
	Origin     []Origin                 `json:"origin,omitempty"`
	Metadata   map[string]MetadataValue `json:"metadata,omitempty"`
	Version    uint64                   `json:"version"`
	CreatedAt  time.Time                `json:"created_at"`
	UpdatedAt  time.Time                `json:"updated_at"`
}

func (comment Comment) Validate() error {
	switch {
	case strings.TrimSpace(string(comment.ID)) == "":
		return validation.New(validation.CoreIDRequired, "id", "id is required")
	case !ValidRecordID(string(comment.ID)):
		return validation.New(validation.CoreIDInvalid, "id", "id must be opaque ASCII matching "+RecordIDPattern)
	case comment.Locale != "" && !ValidLocale(comment.Locale):
		return validation.New(validation.CoreLocaleInvalid, "locale", "locale must be a BCP 47 tag")
	case strings.TrimSpace(string(comment.EntryID)) == "":
		return validation.New(validation.CoreCommentEntryRequired, "entry_id", "entry_id is required")
	case comment.Version == 0:
		return validation.New(validation.CoreVersionRequired, "version", "version must be an integer >= 1")
	case comment.CreatedAt.IsZero() || comment.UpdatedAt.IsZero():
		return validation.New(validation.CoreTimestampsRequired, "created_at", "created_at and updated_at must be RFC 3339")
	case !comment.Status.Valid():
		return validation.New(validation.CoreCommentStatusInvalid, "status", "comment status is invalid")
	}
	return validateMetadata(comment.Metadata)
}

type MenuTarget struct {
	Kind Kind `json:"kind"`
	ID   ID   `json:"id"`
}

type MenuItem struct {
	ID       ID          `json:"id"`
	Label    string      `json:"label"`
	URL      string      `json:"url"`
	ParentID ID          `json:"parent_id,omitempty"`
	Order    *int64      `json:"order,omitempty"`
	Target   *MenuTarget `json:"target,omitempty"`
}

// Menu is a v0.4 navigation menu. Labels are plain strings; translations live in dictionaries.
type Menu struct {
	ID       ID         `json:"id"`
	Location string     `json:"location"`
	Locale   string     `json:"locale"`
	Items    []MenuItem `json:"items"`
}

func (menu Menu) Validate() error {
	switch {
	case strings.TrimSpace(string(menu.ID)) == "":
		return validation.New(validation.CoreIDRequired, "id", "id is required")
	case strings.TrimSpace(menu.Location) == "":
		return validation.New(validation.CoreMenuLocationRequired, "location", "location is required")
	case !ValidLocale(menu.Locale):
		return validation.New(validation.CoreLocaleInvalid, "locale", "locale must be a BCP 47 tag")
	case menu.Items == nil:
		return validation.New(validation.CoreMenuItemsRequired, "items", "items must be an array")
	}
	ids := make(map[ID]struct{}, len(menu.Items))
	for _, item := range menu.Items {
		if strings.TrimSpace(string(item.ID)) == "" || strings.TrimSpace(item.Label) == "" || strings.TrimSpace(item.URL) == "" {
			return validation.New(validation.CoreMenuItemIncomplete, "items", "menu item needs id, label and url")
		}
		if _, exists := ids[item.ID]; exists {
			return validation.New(validation.CoreMenuItemDuplicated, "items", "menu item is duplicated")
		}
		ids[item.ID] = struct{}{}
		if item.Target != nil && (strings.TrimSpace(string(item.Target.Kind)) == "" || strings.TrimSpace(string(item.Target.ID)) == "") {
			return validation.New(validation.CoreMenuTargetInvalid, "items", "target must be {kind, id}")
		}
	}
	return nil
}

// Dictionary holds short UI strings. Long text does not belong here.
type Dictionary struct {
	Locale       string            `json:"locale"`
	SourceLocale string            `json:"source_locale,omitempty"`
	Messages     map[string]string `json:"messages"`
}

func (dictionary Dictionary) Validate() error {
	if !ValidLocale(dictionary.Locale) {
		return validation.New(validation.CoreLocaleInvalid, "locale", "locale must be a BCP 47 tag")
	}
	if dictionary.SourceLocale != "" && !ValidLocale(dictionary.SourceLocale) {
		return validation.New(validation.CoreLocaleInvalid, "source_locale", "source_locale must be a BCP 47 tag")
	}
	if dictionary.Messages == nil {
		return validation.New(validation.CoreDictionaryMessagesRequired, "messages", "messages must be an object")
	}
	return nil
}

// SettingsV04 is the contract name for Settings (@ui8kit/contract SettingsV04).
// It is a separate record type, not an entry of kind=setting. The key is (id, locale).
type SettingsV04 = Settings

// Weekdays of settings.hours[].days (ISO 8601 order, Monday first). Source order is kept.
const (
	WeekdayMon = "mon"
	WeekdayTue = "tue"
	WeekdayWed = "wed"
	WeekdayThu = "thu"
	WeekdayFri = "fri"
	WeekdaySat = "sat"
	WeekdaySun = "sun"
)

// LocaleNeutralSettings are the settings paths that must match across locales of
// one settings id. Only contact.address is localized.
var LocaleNeutralSettings = []string{
	"contact.phone", "contact.email", "contact.links", "hours", "hours_exceptions", "map",
}

// ContactLink is one messenger or social profile. network is a lowercase
// identifier; handle is the network id without spaces (not a URL).
type ContactLink struct {
	Network string `json:"network"`
	Handle  string `json:"handle"`
}

// Contact is platform-neutral contact data. Every field is optional; nested
// empty values are invalid (omit the key). address is localized per record.
type Contact struct {
	Phone   string        `json:"phone,omitempty"`
	Email   string        `json:"email,omitempty"`
	Address string        `json:"address,omitempty"`
	Links   []ContactLink `json:"links,omitempty"`
}

// HoursRule is weekly opening hours on days, from open to close ("HH:MM").
// close may be "24:00"; close earlier than open runs past midnight.
type HoursRule struct {
	Days  []string `json:"days"`
	Open  string   `json:"open"`
	Close string   `json:"close"`
}

// HoursException is one calendar date that replaces the weekly hours:
// {date, closed: true} or {date, open, close}, never both.
type HoursException struct {
	Date   string `json:"date"`
	Closed *bool  `json:"closed,omitempty"`
	Open   string `json:"open,omitempty"`
	Close  string `json:"close,omitempty"`
}

// MapPin is a map pin: a provider identifier and/or WGS 84 coordinates
// (lat and lng together).
type MapPin struct {
	Provider string   `json:"provider,omitempty"`
	Lat      *float64 `json:"lat,omitempty"`
	Lng      *float64 `json:"lng,omitempty"`
}

// Settings is one site-settings record per (id, locale). Currency is the default
// ISO 4217 code for money fields that do not carry their own currency.
type Settings struct {
	ID              ID                       `json:"id"`
	Locale          string                   `json:"locale"`
	Title           string                   `json:"title"`
	Description     string                   `json:"description,omitempty"`
	Language        string                   `json:"language"`
	Timezone        string                   `json:"timezone,omitempty"`
	LogoMediaID     string                   `json:"logo_media_id,omitempty"`
	IconMediaID     string                   `json:"icon_media_id,omitempty"`
	FrontPageID     string                   `json:"front_page_id,omitempty"`
	PostsPageID     string                   `json:"posts_page_id,omitempty"`
	PostsPerPage    *int64                   `json:"posts_per_page,omitempty"`
	CommentsOpen    *bool                    `json:"comments_open,omitempty"`
	Currency        string                   `json:"currency,omitempty"`
	Contact         *Contact                 `json:"contact,omitempty"`
	Hours           []HoursRule              `json:"hours,omitempty"`
	HoursExceptions []HoursException         `json:"hours_exceptions,omitempty"`
	Map             *MapPin                  `json:"map,omitempty"`
	Origin          []Origin                 `json:"origin,omitempty"`
	Metadata        map[string]MetadataValue `json:"metadata,omitempty"`
	Version         uint64                   `json:"version"`
	CreatedAt       time.Time                `json:"created_at"`
	UpdatedAt       time.Time                `json:"updated_at"`
}

func (settings Settings) Validate() error {
	switch {
	case strings.TrimSpace(string(settings.ID)) == "":
		return validation.New(validation.CoreIDRequired, "id", "id is required")
	case !ValidRecordID(string(settings.ID)):
		return validation.New(validation.CoreIDInvalid, "id", "id must be opaque ASCII matching "+RecordIDPattern)
	case !ValidLocale(settings.Locale):
		return validation.New(validation.CoreLocaleInvalid, "locale", "locale must be a BCP 47 tag")
	case strings.TrimSpace(settings.Title) == "":
		return validation.New(validation.CoreSettingsTitleRequired, "title", "title is required")
	case !ValidLocale(settings.Language):
		return validation.New(validation.CoreSettingsLanguageInvalid, "language", "language must be a BCP 47 tag like \"en-US\"")
	case settings.Version == 0:
		return validation.New(validation.CoreVersionRequired, "version", "version must be an integer >= 1")
	case settings.CreatedAt.IsZero() || settings.UpdatedAt.IsZero():
		return validation.New(validation.CoreTimestampsRequired, "created_at", "created_at and updated_at must be RFC 3339")
	case settings.Timezone != "" && !ValidTimezone(settings.Timezone):
		return validation.New(validation.CoreSettingsTimezoneInvalid, "timezone", "timezone must be an IANA name like Europe/Moscow")
	case settings.PostsPerPage != nil && *settings.PostsPerPage < 1:
		return validation.New(validation.CoreSettingsTypeInvalid, "posts_per_page", "posts_per_page must be an integer >= 1")
	case settings.Currency != "" && !ValidCurrency(settings.Currency):
		return validation.New(validation.CoreSettingsCurrencyInvalid, "currency", "currency must be an ISO 4217 code like USD")
	}
	if err := validateContact(settings.Contact); err != nil {
		return err
	}
	if err := validateHours(settings.Hours); err != nil {
		return err
	}
	if err := validateHoursExceptions(settings.HoursExceptions); err != nil {
		return err
	}
	if err := validateMapPin(settings.Map); err != nil {
		return err
	}
	if err := validateOrigin(settings.Origin); err != nil {
		return err
	}
	return validateMetadata(settings.Metadata)
}

// ValidCurrency reports whether value is ISO 4217 (three capital letters).
func ValidCurrency(value string) bool {
	if len(value) != 3 {
		return false
	}
	for _, c := range value {
		if c < 'A' || c > 'Z' {
			return false
		}
	}
	return true
}

var (
	timezonePattern  = regexp.MustCompile(`^[A-Za-z_]+(/[A-Za-z0-9_+-]+)*$|^UTC$`)
	e164Pattern      = regexp.MustCompile(`^\+[1-9][0-9]{1,14}$`)
	emailPattern     = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	handlePattern    = regexp.MustCompile(`^\S+$`)
	openTimePattern  = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)
	closeTimePattern = regexp.MustCompile(`^(([01][0-9]|2[0-3]):[0-5][0-9]|24:00)$`)
	calendarDate     = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)
	weekdays         = map[string]struct{}{
		WeekdayMon: {}, WeekdayTue: {}, WeekdayWed: {}, WeekdayThu: {},
		WeekdayFri: {}, WeekdaySat: {}, WeekdaySun: {},
	}
)

// ValidTimezone reports whether value is an IANA time zone name or UTC.
func ValidTimezone(value string) bool {
	return timezonePattern.MatchString(value)
}

// ValidE164 reports whether value is an E.164 phone ("+" and 2 to 15 digits).
func ValidE164(value string) bool {
	return e164Pattern.MatchString(value)
}

// ValidEmail reports whether value looks like name@example.com.
func ValidEmail(value string) bool {
	return emailPattern.MatchString(value)
}

// ValidHandle reports whether value is a non-empty string without whitespace.
func ValidHandle(value string) bool {
	return handlePattern.MatchString(value)
}

// ValidOpenTime reports whether value is HH:MM from 00:00 to 23:59.
func ValidOpenTime(value string) bool {
	return openTimePattern.MatchString(value)
}

// ValidCloseTime reports whether value is HH:MM from 00:00 to 24:00.
func ValidCloseTime(value string) bool {
	return closeTimePattern.MatchString(value)
}

// ValidCalendarDate reports whether value is an existing calendar day YYYY-MM-DD.
func ValidCalendarDate(value string) bool {
	if !calendarDate.MatchString(value) {
		return false
	}
	var y, m, d int
	if _, err := fmt.Sscanf(value, "%d-%d-%d", &y, &m, &d); err != nil {
		return false
	}
	t := time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
	return t.Year() == y && int(t.Month()) == m && t.Day() == d
}

func timeMinutes(hhmm string) int {
	return (int(hhmm[0]-'0')*10+int(hhmm[1]-'0'))*60 + int(hhmm[3]-'0')*10 + int(hhmm[4]-'0')
}

func validInterval(open, close string) bool {
	if !ValidOpenTime(open) || !ValidCloseTime(close) {
		return false
	}
	if close != "24:00" && timeMinutes(close) == timeMinutes(open) {
		return false
	}
	return true
}

func validateContact(c *Contact) error {
	if c == nil {
		return nil
	}
	if c.Phone != "" && !ValidE164(c.Phone) {
		return validation.New(validation.CoreSettingsPhoneInvalid, "contact.phone", "phone must be E.164: + and 2 to 15 digits without spaces, e.g. +12025550143")
	}
	if c.Email != "" && !ValidEmail(c.Email) {
		return validation.New(validation.CoreSettingsEmailInvalid, "contact.email", "email must look like name@example.com")
	}
	if c.Address != "" && strings.TrimSpace(c.Address) == "" {
		return validation.New(validation.CoreSettingsTypeInvalid, "contact.address", "address must be a non-blank string (omit it when there is none)")
	}
	if c.Links == nil {
		return nil
	}
	if len(c.Links) == 0 {
		return validation.New(validation.CoreSettingsTypeInvalid, "contact.links", "links must be a non-empty array of {network, handle} (omit it when there are none)")
	}
	for _, link := range c.Links {
		if !ValidIdentifier(link.Network) || !ValidHandle(link.Handle) {
			return validation.New(validation.CoreSettingsLinkInvalid, "contact.links", "a link must be {network, handle}")
		}
	}
	return nil
}

func validateHours(hours []HoursRule) error {
	for _, rule := range hours {
		if len(rule.Days) == 0 {
			return validation.New(validation.CoreSettingsHoursInvalid, "hours", "days must be a non-empty array of mon, tue, wed, thu, fri, sat, sun")
		}
		seen := make(map[string]struct{}, len(rule.Days))
		for _, day := range rule.Days {
			if _, ok := weekdays[day]; !ok {
				return validation.New(validation.CoreSettingsHoursInvalid, "hours", "day must be one of mon, tue, wed, thu, fri, sat, sun")
			}
			if _, dup := seen[day]; dup {
				return validation.New(validation.CoreSettingsHoursInvalid, "hours", "a day is listed twice in one rule")
			}
			seen[day] = struct{}{}
		}
		if !validInterval(rule.Open, rule.Close) {
			return validation.New(validation.CoreSettingsHoursInvalid, "hours", "an hours rule must be {days, open, close}")
		}
	}
	return nil
}

func validateHoursExceptions(exceptions []HoursException) error {
	seen := make(map[string]struct{}, len(exceptions))
	for _, ex := range exceptions {
		if !ValidCalendarDate(ex.Date) {
			return validation.New(validation.CoreSettingsHoursExceptionInvalid, "hours_exceptions", "date must be an existing calendar day YYYY-MM-DD")
		}
		if _, dup := seen[ex.Date]; dup {
			return validation.New(validation.CoreSettingsHoursExceptionDuplicated, "hours_exceptions", "date already has an exception")
		}
		seen[ex.Date] = struct{}{}
		timed := ex.Open != "" || ex.Close != ""
		if ex.Closed != nil {
			if !*ex.Closed {
				return validation.New(validation.CoreSettingsHoursExceptionInvalid, "hours_exceptions", "closed must be true (give open and close for a short day)")
			}
			if timed {
				return validation.New(validation.CoreSettingsHoursExceptionInvalid, "hours_exceptions", "an exception is either {date, closed: true} or {date, open, close}, not both")
			}
		} else if !timed {
			return validation.New(validation.CoreSettingsHoursExceptionInvalid, "hours_exceptions", "an exception needs closed: true or open and close")
		} else if !validInterval(ex.Open, ex.Close) {
			return validation.New(validation.CoreSettingsHoursExceptionInvalid, "hours_exceptions", "an exception must be {date, closed: true} or {date, open, close}")
		}
	}
	return nil
}

func validateMapPin(m *MapPin) error {
	if m == nil {
		return nil
	}
	if m.Provider == "" && m.Lat == nil && m.Lng == nil {
		return validation.New(validation.CoreSettingsMapInvalid, "map", "map needs a provider or lat and lng (omit it when there is none)")
	}
	if m.Provider != "" && !ValidIdentifier(m.Provider) {
		return validation.New(validation.CoreSettingsMapInvalid, "map.provider", "provider must be a lowercase identifier like osm, google or yandex")
	}
	if (m.Lat == nil) != (m.Lng == nil) {
		return validation.New(validation.CoreSettingsMapInvalid, "map", "lat and lng go together")
	}
	if m.Lat != nil && (*m.Lat < -90 || *m.Lat > 90) {
		return validation.New(validation.CoreSettingsMapInvalid, "map.lat", "lat must be a number from -90 to 90")
	}
	if m.Lng != nil && (*m.Lng < -180 || *m.Lng > 180) {
		return validation.New(validation.CoreSettingsMapInvalid, "map.lng", "lng must be a number from -180 to 180")
	}
	return nil
}
