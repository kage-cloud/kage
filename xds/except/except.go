package except

import "fmt"

type ErrorReason string

const (
	ErrNotFound      ErrorReason = "NotFound"
	ErrConflict      ErrorReason = "Conflict"
	ErrInternalError ErrorReason = "InternalError"
	ErrUnsupported   ErrorReason = "Unsupported"
	ErrAlreadyExists ErrorReason = "AlreadyExists"
	ErrTimeout       ErrorReason = "Timeout"
)

type kageError struct {
	Reason  ErrorReason
	Message string
}

func (s *kageError) Error() string {
	return s.Message
}

func Reason(err error) ErrorReason {
	if err != nil {
		if v, ok := err.(*kageError); ok {
			return v.Reason
		}
	}
	return ErrInternalError
}

func NewError(msg string, reason ErrorReason, args ...interface{}) error {
	return &kageError{
		Reason:  reason,
		Message: fmt.Sprintf(msg, args...),
	}
}
