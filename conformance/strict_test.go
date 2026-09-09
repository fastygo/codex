package conformance

import (
	"testing"

	"github.com/fastygo/codex/content"
)

func TestDecodeStrictRejectsUnknownAndTrailingValues(t *testing.T) {
	tests := [][]byte{
		[]byte(`{"id":"entry-1","unknown":true}`),
		[]byte(`{} {}`),
	}
	for _, data := range tests {
		var entry content.Entry
		if err := decodeStrict(data, &entry); err == nil {
			t.Fatalf("decodeStrict(%q) error = nil", data)
		}
	}
}
