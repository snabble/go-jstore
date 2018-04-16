package http

import (
	"fmt"
)

type Error interface {
	IsClientError() bool
}

type internalError struct {
	msg string
}

func (err *internalError) IsClientError() bool {
	return false
}

func (err *internalError) Error() string {
	return err.msg
}

func InternalError(msg string, args ...interface{}) error {
	return &internalError{msg: fmt.Sprintf(msg, args...)}
}

type clientError struct {
	msg string
}

func (err *clientError) IsClientError() bool {
	return true
}

func (err *clientError) Error() string {
	return err.msg
}

func ClientError(msg string, args ...interface{}) error {
	return &clientError{msg: fmt.Sprintf(msg, args...)}
}

func WrapWithClientError(err error) error {
	return &clientError{msg: err.Error()}
}
