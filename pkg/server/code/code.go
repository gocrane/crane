package code

import (
	"fmt"
	"net/http"

	"github.com/gocrane/crane/pkg/server/errors"
)

/// ErrCode implements ErrorCoder
type ErrCode struct {
	C        int
	HTTPCode int
	Msg      string
	Ref      string
}

var _ errors.ErrorCoder = &ErrCode{}

// Code returns the integer code of ErrCode.
func (coder ErrCode) Code() int {
	return coder.C
}

// String implements stringer. String returns the external error message,
// if any.
func (coder ErrCode) String() string {
	return coder.Msg
}

// Reference returns the reference document.
func (coder ErrCode) Reference() string {
	return coder.Ref
}

// HTTPStatus returns the associated HTTP status code, if any. Otherwise,
// returns 200.
func (coder ErrCode) HTTPStatus() int {
	if coder.HTTPCode == 0 {
		return http.StatusInternalServerError
	}

	return coder.HTTPCode
}

func ValidateHttpCode(httpCode int) bool {
	validHttpCodes := []int{200, 400, 401, 403, 404, 500}
	for _, code := range validHttpCodes {
		if code == httpCode {
			return true
		}
	}
	return false
}

// nolint: unparam,deadcode
func register(code int, httpCode int, message string, refs ...string) {
	if !ValidateHttpCode(httpCode) {
		panic(fmt.Sprintf("unknown http code %v", httpCode))
	}

	var reference string
	if len(refs) > 0 {
		reference = refs[0]
	}

	coder := &ErrCode{
		C:        code,
		HTTPCode: httpCode,
		Msg:      message,
		Ref:      reference,
	}

	errors.MustRegister(coder)
}
