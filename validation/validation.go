// Package validation defines machine-readable contract validation errors.
package validation

import (
	"errors"
	"fmt"
)

type Error struct {
	Code string
	Path string
	Err  error
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

func New(code, path, message string) error {
	return &Error{Code: code, Path: path, Err: errors.New(message)}
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
