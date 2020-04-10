package except

import (
	"fmt"
	"strings"
)

type BatchError interface {
	ReasonedError
	Add(err error)
	Len() int
}

func NewBatchError(msg string, args ...interface{}) BatchError {
	return &batchError{
		Message:      fmt.Sprintf(msg, args...),
		Errors:       []error{},
		ErrorStrings: []string{},
	}
}

type batchError struct {
	Message      string
	Errors       []error
	ErrorStrings []string
}

func (b *batchError) Len() int {
	return len(b.Errors)
}

func (b *batchError) Reason() ErrorReason {
	return ErrBatch
}

func (b *batchError) Error() string {
	return b.Message + ":\n" + strings.Join(b.ErrorStrings, "\n,")
}

func (b *batchError) Add(err error) {
	b.Errors = append(b.Errors, err)
	b.ErrorStrings = append(b.ErrorStrings, err.Error())
}
