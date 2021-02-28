package echo

import (
	"reflect"
	"runtime"

	"github.com/labstack/echo/v4"
	echoSwagger "github.com/soldatov-s/go-swagger/echo-swagger"
)

type HandlerFunc func(Context) error

func handlerName(h HandlerFunc) string {
	t := reflect.ValueOf(h).Type()
	if t.Kind() == reflect.Func {
		return runtime.FuncForPC(reflect.ValueOf(h).Pointer()).Name()
	}
	return t.String()
}

func Handler(f HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Passing the real method name to swagger
		if echoSwagger.IsBuildingSwagger(c) {
			methodName := handlerName(f)
			c.Set("swMethodName", methodName)
		}
		return f(Context{c})
	}
}
