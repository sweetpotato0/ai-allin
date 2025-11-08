package errors

import "errors"

// Sentinel errors for common error conditions
var (
	// ErrNotFound indicates that a requested resource was not found
	ErrNotFound = errors.New("resource not found")

	// ErrAlreadyExists indicates that a resource already exists
	ErrAlreadyExists = errors.New("resource already exists")

	// ErrInvalidInput indicates that input validation failed
	ErrInvalidInput = errors.New("invalid input")

	// ErrUnauthorized indicates that the operation is not authorized
	ErrUnauthorized = errors.New("unauthorized")

	// ErrInternal indicates an internal server error
	ErrInternal = errors.New("internal error")
)
