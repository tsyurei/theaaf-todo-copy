package app

import (
	"net/http"
	"todo-app/internal/model"
	"todo-app/internal/db"
	"github.com/sirupsen/logrus"
)

type Context struct {
	Logger        logrus.FieldLogger
	RemoteAddress string
	Database *db.Database
	User *model.User
}

func (ctx *Context) WithLogger(logger logrus.FieldLogger) *Context {
	ret := *ctx
	ret.Logger = logger
	return &ret
}

func (ctx *Context) WithRemoteAddress(address string) *Context {
	// TODO: experiment more with this
	ret := *ctx
	ret.RemoteAddress = address
	return &ret
}

func (ctx *Context) WithUser(user *model.User) *Context {
	ctx.User = user
	return ctx
}

func (ctx *Context) AuthorizationError() *UserError {
	return &UserError{Message: "unauthorized", StatusCode: http.StatusForbidden}
}