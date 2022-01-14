package ginwrapper

import (
	"net/http"

	"k8s.io/klog/v2"

	"github.com/gin-gonic/gin"

	"github.com/gocrane/crane/pkg/server/errors"
)

type ErrorResponse struct {
	// Code defines the business error code.
	Code int `json:"code"`
	// Message is the detail message of the error
	Message string `json:"message"`
	// Reference is a http url which content is about the error
	Reference string `json:"reference,omitempty"`
}

// WriteResponse write an error or the response data into http response body.
func WriteResponse(c *gin.Context, err error, data interface{}) {
	if err != nil {
		klog.Errorf("%#+v, cause: %v", err, errors.Cause(err))
		coder := errors.ParseErrorCoder(err)
		c.JSON(coder.HTTPStatus(), ErrorResponse{
			Code:      coder.Code(),
			Message:   coder.String(),
			Reference: coder.Reference(),
		})

		return
	}

	c.JSON(http.StatusOK, data)
}
