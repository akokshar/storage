package modules

import (
	"fmt"
	"net/http"
)

// HandlerError error interface
type HandlerError interface {
	Code() int
}

type handlerError struct {
	errCode int
}

// NewHandlerErrorWithCode create new backend.Error struct
func NewHandlerErrorWithCode(code int) HandlerError {
	return &handlerError{
		errCode: code,
	}
}

func (e *handlerError) Code() int {
	return e.errCode
}

func (e *handlerError) Error() string {
	return fmt.Sprintf("HandlerError: '%s'", http.StatusText(e.Code()))
}
