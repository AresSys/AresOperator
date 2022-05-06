package observer

import (
	"ares-operator/observer/server"

	"github.com/gin-gonic/gin"
)

var (
	InvalidParam = server.NewBadRequest("230400", "InvalidParam")
	NotFound     = server.NewNotFound("230404", "NotFound")
	UnknownError = server.NewInternalError("230500", "UnknownError")
)

func HandleError(c *gin.Context, err error) {
	if serviceError, ok := err.(server.ServiceError); ok {
		ServerLog.Error(err, serviceError.Error())
		server.Failure(c, serviceError.Code, serviceError.Message)
		return
	}
	ServerLog.Error(err, "unexpected error")
	server.Failure(c, UnknownError, err)
}
