package ginwrapper

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	// Message is the detail message of the error
	Error string `json:"error"`
	// Data is the response data
	Data interface{} `json:"data"`
}

// WriteResponse write an error or the response data into http response body.
// The Response.Error is empty if the err is null, or the Response.Error is the error message.
func WriteResponse(c *gin.Context, err error, data interface{}) {
	if err != nil {
		c.JSON(http.StatusOK, Response{
			Error: err.Error(),
			Data:  data,
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Data: data,
	})
}
