// Package canonical implements the Codex v0.4 entry canonical form
// (profile codex-entry/v1, draft v0.4). It is a Go port of
// @ui8kit/contract src/canonical.ts at commit 8299fee.
//
// Serialization is RFC 8785 (JCS). encoding/json is not used for output:
// it HTML-escapes <>& and \u-escapes U+2028/U+2029, which JCS forbids.
package canonical

import "fmt"

const (
	EntryProfile           = "codex-entry/v1"
	EntryDigestPrefix      = EntryProfile + ":sha256:"
	ManifestDigestV1Prefix = "codex-manifest/v1:sha256:"
	ManifestDigestV2Prefix = "codex-manifest/v2:sha256:"
)

const (
	ErrInvalidJSON       = "canonical.invalid_json"
	ErrDuplicateKey      = "canonical.duplicate_key"
	ErrUnsafeInteger     = "canonical.unsafe_integer"
	ErrInvalidNumber     = "canonical.invalid_number"
	ErrInvalidString     = "canonical.invalid_string"
	ErrInvalidType       = "canonical.invalid_type"
	ErrUnknownField      = "canonical.unknown_field"
	ErrMissingField      = "canonical.missing_field"
	ErrInvalidID         = "canonical.invalid_id"
	ErrInvalidDate       = "canonical.invalid_date"
	ErrFractionalSeconds = "canonical.fractional_seconds"
)

// Error is CanonicalError from src/canonical.ts.
type Error struct {
	Code    string
	Path    string
	Message string
}

func (err *Error) Error() string {
	return fmt.Sprintf("%s: %s (at %s)", err.Code, err.Message, err.Path)
}

func fail(code, path, message string) error {
	return &Error{Code: code, Path: path, Message: message}
}

func Code(err error) string {
	if err == nil {
		return ""
	}
	if ce, ok := err.(*Error); ok {
		return ce.Code
	}
	return ""
}
