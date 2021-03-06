package except

import (
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	"net/http"
)

type ErrorReason string

const (
	ErrNotFound      ErrorReason = "NotFound"
	ErrConflict      ErrorReason = "Conflict"
	ErrInternalError ErrorReason = "InternalError"
	ErrUnsupported   ErrorReason = "Unsupported"
	ErrAlreadyExists ErrorReason = "AlreadyExists"
	ErrTimeout       ErrorReason = "Timeout"
	ErrInvalid       ErrorReason = "Invalid"
	ErrBatch         ErrorReason = "Batch"
)

type ReasonedError interface {
	error
	Reason() ErrorReason
}

type kageError struct {
	ErrorReason ErrorReason
	Message     string
}

func (s *kageError) Reason() ErrorReason {
	return s.ErrorReason
}

func (s *kageError) Error() string {
	return s.Message
}

func Reason(err error) ErrorReason {
	if err != nil {
		if v, ok := err.(ReasonedError); ok {
			return v.Reason()
		}
	}
	return ErrInternalError
}

func ToHttpStatus(err error) int {
	if errors.IsNotFound(err) {
		return http.StatusNotFound
	} else if errors.IsAlreadyExists(err) {
		return http.StatusBadRequest
	} else {
		switch Reason(err) {
		case ErrNotFound:
			return http.StatusNotFound
		case ErrAlreadyExists, ErrUnsupported, ErrConflict, ErrInvalid:
			return http.StatusBadRequest
		case ErrTimeout:
			return http.StatusRequestTimeout
		default:
			return http.StatusInternalServerError
		}
	}
}

func NewError(msg string, reason ErrorReason, args ...interface{}) error {
	return &kageError{
		ErrorReason: reason,
		Message:     fmt.Sprintf(msg, args...),
	}
}
