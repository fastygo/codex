package validation_test

import (
	"errors"
	"testing"

	"github.com/fastygo/codex/validation"
)

func TestMachineReadableValidationError(t *testing.T) {
	err := validation.Wrap(
		"schema.resource.invalid",
		"resources[0]",
		validation.New("schema.resource.id", "record.id", "identifier is invalid"),
	)
	if got := validation.Code(err); got != "schema.resource.invalid" {
		t.Fatalf("Code() = %q", got)
	}
	if got := validation.Path(err); got != "resources[0]" {
		t.Fatalf("Path() = %q", got)
	}
	var contractError *validation.Error
	if !errors.As(err, &contractError) {
		t.Fatal("errors.As() did not find validation.Error")
	}
}
