package service

import (
	"errors"
	"fmt"
	"net/http"
)

// AppError 是带 HTTP 状态码的业务错误，handler 直接据此回包。
type AppError struct {
	Status  int
	Message string
	Err     error
}

func (e *AppError) Error() string { return e.Message }
func (e *AppError) Unwrap() error { return e.Err }

func badRequest(format string, args ...any) error {
	return &AppError{Status: http.StatusBadRequest, Message: fmt.Sprintf(format, args...)}
}

func notFound(format string, args ...any) error {
	return &AppError{Status: http.StatusNotFound, Message: fmt.Sprintf(format, args...)}
}

func forbidden(format string, args ...any) error {
	return &AppError{Status: http.StatusForbidden, Message: fmt.Sprintf(format, args...)}
}

func conflict(format string, args ...any) error {
	return &AppError{Status: http.StatusConflict, Message: fmt.Sprintf(format, args...)}
}

// ErrNoRows 是内部「查不到」的哨兵，避免把 gorm 的错误泄露到接口。
var ErrNoRows = errors.New("记录不存在")
