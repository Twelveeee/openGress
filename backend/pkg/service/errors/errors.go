package errors

import "fmt"

type Kind string

const (
	KindBadRequest   Kind = "bad_request"
	KindUnauthorized Kind = "unauthorized"
	KindNotFound     Kind = "not_found"
	KindConflict     Kind = "conflict"
	KindInternal     Kind = "internal"
)

type ServiceError struct {
	Kind    Kind
	Message string
}

func (e *ServiceError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func New(kind Kind, message string) *ServiceError {
	return &ServiceError{Kind: kind, Message: message}
}

func Wrap(kind Kind, message string, cause error) *ServiceError {
	if cause == nil {
		return New(kind, message)
	}
	return &ServiceError{Kind: kind, Message: fmt.Sprintf("%s: %v", message, cause)}
}

func IsKind(err error, kind Kind) bool {
	se, ok := err.(*ServiceError)
	return ok && se.Kind == kind
}
