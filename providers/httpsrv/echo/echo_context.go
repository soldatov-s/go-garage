package echo

import (
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
)

type Context interface {
	echo.Context
}

func GetReqID(ec Context) string {
	return ec.Request().Header.Get("x-request-id")
}

func GetLog(ec Context) *zerolog.Logger {
	return ec.Get(zerologWithReqID).(*zerolog.Logger)
}

func GetInt64Param(ec Context, param string) (int64, error) {
	return strconv.ParseInt(ec.Param(param), 10, 64)
}
