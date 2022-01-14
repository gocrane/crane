package errors

import "fmt"

type codeError struct {
	err   error
	code  int
	cause error
}

func WithCode(code int, format string, args ...interface{}) error {
	return &codeError{
		err:  fmt.Errorf(format, args...),
		code: code,
	}
}

func WrapC(err error, code int, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}

	return &codeError{
		err:   fmt.Errorf(format, args...),
		code:  code,
		cause: err,
	}
}

func (w *codeError) Error() string { return fmt.Sprintf("%v", w.err) }

func (w *codeError) Cause() error { return w.cause }

func (w *codeError) Unwrap() error { return w.cause }
