package errors

import (
	"errors"
)

const (
	errCannotConnect       = "cannot connect to NATS"
	errEmptyConnectionURL  = "empty NATS connection URL"
	errEmptyConnectionName = "empty NATS connection name"
)

// ErrCannotConnect represents an error when NATS connection failed.
var ErrCannotConnect = errors.New(errCannotConnect)

// ErrEmptyConnectionURL represents an error when NATS connection URL is empty.
var ErrEmptyConnectionURL = errors.New(errEmptyConnectionURL)

// ErrEmptyConnectionName represents an error when NATS connection name is empty.
var ErrEmptyConnectionName = errors.New(errEmptyConnectionName)
