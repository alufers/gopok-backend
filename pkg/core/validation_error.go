package core

import (
	"encoding/json"
)

/*
ValidationError is created when a model is invalid.
*/
type ValidationError interface {
	Error() string
	Field() string
	Model() string
}

type simpleValidationError struct {
	message string
	field   string
	model   string
}

func (e simpleValidationError) Error() string {
	return e.message
}

func (e simpleValidationError) Field() string {
	return e.field
}

func (e simpleValidationError) Model() string {
	return e.model
}

func (e simpleValidationError) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{
		"message": e.message,
		"field":   e.field,
		"model":   e.model,
	})
}

/*
NewValidationError Creates a new simpleValidationError with basic info.
*/
func NewValidationError(message, field, model string) ValidationError {
	return simpleValidationError{message, field, model}
}
