package echo

import (
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
