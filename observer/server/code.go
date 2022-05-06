package server

import (
	"fmt"
	"net/http"
)

type ErrorCode struct {
	Code     string
	Reason   string
	HttpCode int
}

func NewErrorCode(code, reason string, httpCode int) *ErrorCode {
	return &ErrorCode{
		Code:     code,
		Reason:   reason,
		HttpCode: httpCode,
	}
}

func NewBadRequest(code, reason string) *ErrorCode {
	return &ErrorCode{
		Code:     code,
		Reason:   reason,
		HttpCode: http.StatusBadRequest,
	}
}

func NewNotFound(code, reason string) *ErrorCode {
	return &ErrorCode{
		Code:     code,
		Reason:   reason,
		HttpCode: http.StatusNotFound,
	}
}

func NewInternalError(code, reason string) *ErrorCode {
	return &ErrorCode{
		Code:     code,
		Reason:   reason,
		HttpCode: http.StatusInternalServerError,
	}
}

func (code *ErrorCode) GetCode() string   { return code.Code }
func (code *ErrorCode) GetReason() string { return code.Reason }
func (code *ErrorCode) GetHttpCode() int  { return code.HttpCode }
func (code *ErrorCode) String() string {
	return fmt.Sprintf("HTTPCode=%d, %s(%s)", code.HttpCode, code.Reason, code.Code)
}
