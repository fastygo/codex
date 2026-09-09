// Package revision defines immutable Codex entry snapshots.
package revision

import (
	"strings"
	"time"

	"github.com/fastygo/codex/content"
	"github.com/fastygo/codex/validation"
)

type ID string

type Revision struct {
	ID        ID            `json:"id"`
	EntryID   content.ID    `json:"entry_id"`
	Version   uint64        `json:"version"`
	Snapshot  content.Entry `json:"snapshot"`
	AuthorID  string        `json:"author_id"`
	Reason    string        `json:"reason,omitempty"`
	CreatedAt time.Time     `json:"created_at"`
}

func (revision Revision) Validate() error {
	switch {
	case strings.TrimSpace(string(revision.ID)) == "":
		return validation.New("revision.id_required", "id", "revision id is required")
	case revision.EntryID == "":
		return validation.New("revision.entry_required", "entry_id", "revision entry id is required")
	case revision.EntryID != revision.Snapshot.ID:
		return validation.New("revision.entry_mismatch", "snapshot.id", "revision snapshot entry does not match")
	case revision.Version == 0 || revision.Version != revision.Snapshot.Version:
		return validation.New("revision.version_mismatch", "version", "revision version does not match snapshot")
	case strings.TrimSpace(revision.AuthorID) == "":
		return validation.New("revision.author_required", "author_id", "revision author is required")
	case revision.CreatedAt.IsZero():
		return validation.New("revision.created_at_required", "created_at", "revision created_at is required")
	}
	return revision.Snapshot.Validate()
}

func (revision Revision) Clone() Revision {
	cloned := revision
	cloned.Snapshot = revision.Snapshot.Clone()
	return cloned
}
