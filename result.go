package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type Result struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

const (
	CodeOK            = 0
	CodeBadRequest    = 400
	CodeInternalError = 500
)

func BuildResultOk(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, &Result{
		Code: CodeOK,
		Msg:  "ok",
		Data: data,
	})
}

func BuildResultError(c *gin.Context, httpCode int, code int, msg string) {
	c.JSON(httpCode, &Result{
		Code: code,
		Msg:  msg,
	})
}

func BuildResult(c *gin.Context, httpCode int, r *Result) {
	c.JSON(httpCode, r)
}
