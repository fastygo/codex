// Package validation defines machine-readable contract validation errors.
package validation

import (
	"errors"
	"fmt"
)

type Error struct {
	Code     string
	Path     string
	Err      error
	Severity string
}

func (err *Error) Error() string {
	switch {
	case err.Path != "" && err.Err != nil:
		return fmt.Sprintf("%s at %s: %v", err.Code, err.Path, err.Err)
	case err.Err != nil:
		return fmt.Sprintf("%s: %v", err.Code, err.Err)
	default:
		return err.Code
	}
}

func (err *Error) Unwrap() error {
	return err.Err
}

func (err *Error) IsWarning() bool {
	return err != nil && err.Severity == "warning"
}

func New(code, path, message string) error {
	return &Error{Code: code, Path: path, Err: errors.New(message)}
}

func NewWarning(code, path, message string) error {
	return &Error{Code: code, Path: path, Err: errors.New(message), Severity: "warning"}
}

func Wrap(code, path string, err error) error {
	if err == nil {
		return nil
	}
	return &Error{Code: code, Path: path, Err: err}
}

func Code(err error) string {
	var contractError *Error
	if errors.As(err, &contractError) {
		return contractError.Code
	}
	return ""
}

func Path(err error) string {
	var contractError *Error
	if errors.As(err, &contractError) {
		return contractError.Path
	}
	return ""
}

func Severity(err error) string {
	var contractError *Error
	if errors.As(err, &contractError) {
		return contractError.Severity
	}
	return ""
}

// Issue is the collection-validator report row, matching @ui8kit/contract IssueV04.
type Issue struct {
	Code     string `json:"code"`
	Path     string `json:"path"`
	Message  string `json:"message"`
	Severity string `json:"severity,omitempty"`
}

func (issue Issue) Error() string {
	if issue.Path != "" {
		return fmt.Sprintf("%s at %s: %s", issue.Code, issue.Path, issue.Message)
	}
	if issue.Message != "" {
		return fmt.Sprintf("%s: %s", issue.Code, issue.Message)
	}
	return issue.Code
}

func (issue Issue) IsWarning() bool {
	return issue.Severity == "warning"
}

func (issue Issue) AsError() error {
	return &Error{
		Code:     issue.Code,
		Path:     issue.Path,
		Err:      errors.New(issue.Message),
		Severity: issue.Severity,
	}
}

func IssueOf(code, path, message string) Issue {
	issue := Issue{Code: code, Path: path, Message: message}
	if _, warn := WarningCodes[code]; warn {
		issue.Severity = "warning"
	}
	return issue
}

func Codes(issues []Issue) []string {
	codes := make([]string, 0, len(issues))
	for _, issue := range issues {
		codes = append(codes, issue.Code)
	}
	return codes
}

func ErrorsOnly(issues []Issue) []Issue {
	out := make([]Issue, 0, len(issues))
	for _, issue := range issues {
		if !issue.IsWarning() {
			out = append(out, issue)
		}
	}
	return out
}
