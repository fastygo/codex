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
		Kind:       "message",
		Status:     content.StatusPublished,
		Visibility: content.VisibilityPrivate,
		Slug:       content.LocalizedText{"und": "message-1"},
		Title:      content.LocalizedText{"und": "Message 1"},
		Content:    content.LocalizedText{"und": "Hello"},
		Version:    2,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	value := revision.Revision{
		ID:        "revision-2",
		EntryID:   entry.ID,
		Version:   entry.Version,
		Snapshot:  entry,
		AuthorID:  "telegram-import",
		CreatedAt: now,
	}
	if err := value.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	cloned := value.Clone()
	cloned.Snapshot.Title["und"] = "Changed"
	if value.Snapshot.Title["und"] == "Changed" {
		t.Fatal("Clone() aliases snapshot")
	}
}

func TestRevisionRejectsMismatchedVersion(t *testing.T) {
	now := time.Date(2026, time.September, 9, 12, 0, 0, 0, time.UTC)
	value := revision.Revision{
		ID:       "revision-2",
		EntryID:  "message-1",
		Version:  1,
		AuthorID: "telegram-import",
		Snapshot: content.Entry{
			ID: "message-1", Version: 2,
		},
		CreatedAt: now,
	}
	if err := value.Validate(); err == nil {
		t.Fatal("Validate() error = nil")
	}
}
