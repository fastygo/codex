package revision_test

import (
	"testing"
	"time"

	"github.com/fastygo/codex/content"
	"github.com/fastygo/codex/revision"
)

func TestRevisionValidateAndClone(t *testing.T) {
	now := time.Date(2026, time.September, 9, 12, 0, 0, 0, time.UTC)
	entry := content.Entry{
		ID:         "message-1",
		Locale:     "und",
		Kind:       "message",
		Status:     content.StatusPublished,
		Visibility: content.VisibilityPrivate,
		Slug:       "message-1",
		Title:      "Message 1",
		Content:    "Hello",
		Version:    2,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	value := revision.Revision{
		ID:        "revision-2",
		EntryID:   entry.ID,
		Locale:    entry.Locale,
		Version:   entry.Version,
		Snapshot:  entry,
		AuthorID:  "telegram-import",
		CreatedAt: now,
	}
	if err := value.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	cloned := value.Clone()
	cloned.Snapshot.Title = "Changed"
	if value.Snapshot.Title == "Changed" {
		t.Fatal("Clone() aliases snapshot")
	}
}

func TestRevisionRejectsMismatchedVersion(t *testing.T) {
	now := time.Date(2026, time.September, 9, 12, 0, 0, 0, time.UTC)
	value := revision.Revision{
		ID:       "revision-2",
		EntryID:  "message-1",
		Locale:   "en",
		Version:  1,
		AuthorID: "telegram-import",
		Snapshot: content.Entry{
			ID: "message-1", Locale: "en", Version: 2,
		},
		CreatedAt: now,
	}
	if err := value.Validate(); err == nil {
		t.Fatal("Validate() error = nil")
	}
}

func TestRevisionRejectsLocaleMismatch(t *testing.T) {
	now := time.Date(2026, time.September, 9, 12, 0, 0, 0, time.UTC)
	value := revision.Revision{
		ID: "revision-2", EntryID: "message-1", Locale: "en", Version: 1,
		AuthorID: "telegram-import", CreatedAt: now,
		Snapshot: content.Entry{
			ID: "message-1", Locale: "ru", Kind: "message",
			Status: content.StatusDraft, Visibility: content.VisibilityPrivate,
			Slug: "x", Title: "x", Version: 1, CreatedAt: now, UpdatedAt: now,
		},
	}
	if err := value.Validate(); err == nil {
		t.Fatal("Validate() error = nil")
	}
}
