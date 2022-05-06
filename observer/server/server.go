package server

import (
	"fmt"
)

type ServiceError struct {
	Code    *ErrorCode
	Message string
}

func NewServiceError(code *ErrorCode, msgAndArgs ...interface{}) ServiceError {
	message := BuildMessage(msgAndArgs...)
	return ServiceError{
		Code:    code,
		Message: message,
	}
}

func (e ServiceError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code.String(), e.Message)
}

func BuildMessage(msgAndArgs ...interface{}) string {
	if msgAndArgs == nil || len(msgAndArgs) == 0 {
		return ""
	}
	if len(msgAndArgs) == 1 {
		msg := msgAndArgs[0]
		if msgAsStr, ok := msg.(string); ok {
			return msgAsStr
		}
		return fmt.Sprintf("%+v", msg)
	}
	if len(msgAndArgs) > 1 {
		return fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...)
	}
	return ""
}
