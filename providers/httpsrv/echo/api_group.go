package echo

import (
	"github.com/labstack/echo/v4"
)

type Group struct {
	echo.Group
}

func (gr *Group) ToEchoGroup() *echo.Group {
	return &gr.Group
}
