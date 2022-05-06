package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Error   string `json:"error"`
}

func Success(c *gin.Context, obj interface{}) {
	c.JSON(http.StatusOK, obj)
}

func Failure(c *gin.Context, errorCode *ErrorCode, msgAndArgs ...interface{}) {
	c.AbortWithStatusJSON(errorCode.GetHttpCode(),
		ErrorResponse{
			Code:    errorCode.GetCode(),
			Message: errorCode.GetReason(),
			Error:   BuildMessage(msgAndArgs...),
		},
	)
}
