package cerrors

import (
	"errors"
)

var (
	ServerBad = &UserError{Msg: "服务器错误"}
	NoLogin   = &UserError{Msg: "请先登录"}
)

type (
	UserError struct {
		Msg string
		Err error
	}
)

var _ error = (*UserError)(nil)

func NewUserError(msg string, err error) *UserError {
	return &UserError{
		Msg: msg,
		Err: err,
	}
}

func (u UserError) Error() string {
	return u.Msg
}

func AsUserError(err error) (*UserError, bool) {
	if e, ok := errors.AsType[*UserError](err); ok {
		return e, true
	}
	return nil, false
}
