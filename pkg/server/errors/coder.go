package errors

import (
	"fmt"
	"net/http"
	"sync"
)

var (
	unknownCoder defaultErrorCoder = defaultErrorCoder{1, http.StatusInternalServerError, "An internal server error occurred", ""}
)

// ErrorCoder define an interface for error code and http status details
type ErrorCoder interface {
	// HTTP status that should be used for the associated error code.
	HTTPStatus() int

	// External (user) facing error text.
	String() string

	// Reference returns the detail documents for user.
	Reference() string

	// Code returns the code of the coder
	Code() int
}

type defaultErrorCoder struct {
	// C refers to the integer code of the ErrCode.
	C int

	// HTTP status that should be used for the associated error code.
	HTTP int

	// External (user) facing error text.
	Ext string

	// Ref specify the reference document.
	Ref string
}

func (coder defaultErrorCoder) Code() int {
	return coder.C

}

func (coder defaultErrorCoder) String() string {
	return coder.Ext
}

func (coder defaultErrorCoder) HTTPStatus() int {
	if coder.HTTP == 0 {
		return 500
	}

	return coder.HTTP
}

// Reference returns the reference document.
func (coder defaultErrorCoder) Reference() string {
	return coder.Ref
}

var errCodes = map[int]ErrorCoder{}
var codeMux = &sync.Mutex{}

// MustRegister register a custom error code.
// It will panic when the same Code already exist.
func MustRegister(coder ErrorCoder) {
	codeMux.Lock()
	defer codeMux.Unlock()

	if _, ok := errCodes[coder.Code()]; ok {
		panic(fmt.Sprintf("code: %d already exist", coder.Code()))
	}

	errCodes[coder.Code()] = coder
}

// ParseErrorCoder parse any error into *withCode.
// nil error will return nil direct.
func ParseErrorCoder(err error) ErrorCoder {
	if err == nil {
		return nil
	}

	if v, ok := err.(*codeError); ok {
		if coder, ok := errCodes[v.code]; ok {
			return coder
		}
	}

	return unknownCoder
}

func Cause(err error) error {
	type causer interface {
		Cause() error
	}

	for err != nil {
		cause, ok := err.(causer)
		if !ok {
			break
		}

		if cause.Cause() == nil {
			break
		}

		err = cause.Cause()
	}
	return err
}

func init() {
	errCodes[unknownCoder.Code()] = unknownCoder
}
