package content

import (
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

// Settings is one site-settings record per (id, locale). Currency is the default
// ISO 4217 code for money fields that do not carry their own currency.
type Settings struct {
	ID           ID                       `json:"id"`
	Locale       string                   `json:"locale"`
	Title        string                   `json:"title"`
	Description  string                   `json:"description,omitempty"`
	Language     string                   `json:"language"`
	Timezone     string                   `json:"timezone,omitempty"`
	LogoMediaID  string                   `json:"logo_media_id,omitempty"`
	IconMediaID  string                   `json:"icon_media_id,omitempty"`
	FrontPageID  string                   `json:"front_page_id,omitempty"`
	PostsPageID  string                   `json:"posts_page_id,omitempty"`
	PostsPerPage *int64                   `json:"posts_per_page,omitempty"`
	CommentsOpen *bool                    `json:"comments_open,omitempty"`
	Currency     string                   `json:"currency,omitempty"`
	Origin       []Origin                 `json:"origin,omitempty"`
	Metadata     map[string]MetadataValue `json:"metadata,omitempty"`
	Version      uint64                   `json:"version"`
	CreatedAt    time.Time                `json:"created_at"`
	UpdatedAt    time.Time                `json:"updated_at"`
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
	case settings.Currency != "" && !ValidCurrency(settings.Currency):
		return validation.New(validation.CoreSettingsCurrencyInvalid, "currency", "currency must be an ISO 4217 code like USD")
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
