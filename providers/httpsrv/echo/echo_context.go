package echo

import (
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	echoSwagger "github.com/soldatov-s/go-swagger/echo-swagger"
	"github.com/soldatov-s/go-swagger/swagger"
)

type Context struct {
	echo.Context
}

func (ec Context) GetReqID() string {
	return ec.Request().Header.Get("x-request-id")
}

func (ec Context) GetLog() *zerolog.Logger {
	log := ec.Get(zerologWithReqID)
	if log == nil {
		return &zerolog.Logger{}
	}
	return log.(*zerolog.Logger)
}

func (ec Context) GetInt64Param(param string) (int64, error) {
	return strconv.ParseInt(ec.Param(param), 10, 64)
}

func (ec Context) IsBuildingSwagger() bool {
	return echoSwagger.IsBuildingSwagger(ec)
}

func (ec Context) AddToSwagger() swagger.IMethod {
	return echoSwagger.AddToSwagger(ec)
}
