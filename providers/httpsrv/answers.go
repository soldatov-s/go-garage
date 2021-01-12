package httpsrv

import (
	"fmt"
	"net/http"
	"strings"
)

type ErrorAnswBody struct {
	Code       string `json:"code"`
	StatusCode int    `json:"statusCode"`
	Details    string `json:"details"`
}

type ErrorAnsw struct {
	Body ErrorAnswBody `json:"error"`
}

func (e ErrorAnsw) Error() string {
	return fmt.Sprintf("error %s: %s", e.Body.Code, e.Body.Details)
}

type ResultAnsw struct {
	Body interface{} `json:"result"`
}

// OkResult return OK answer
func OkResult() ResultAnsw {
	return ResultAnsw{Body: "OK"}
}

// NewErrorAnsw create new ErrorAnsw and fills it withcode, statuscode, details
func NewErrorAnsw(statusCode int, code string, err error) ErrorAnsw {
	return ErrorAnsw{
		Body: ErrorAnswBody{
			Code:       strings.ToUpper(strings.ReplaceAll(code, " ", "_")),
			StatusCode: statusCode,
			Details:    err.Error(),
		},
	}
}

// BadRequest return err 400
func BadRequest(err error) ErrorAnsw {
	return NewErrorAnsw(http.StatusBadRequest, "bad request", err)
}

// Unauthorized reurn err 401
func Unauthorized(err error) ErrorAnsw {
	return NewErrorAnsw(http.StatusUnauthorized, "unauthorized", err)
}

// Forbidden return err 403
func Forbidden(err error) ErrorAnsw {
	return NewErrorAnsw(http.StatusForbidden, "unauthorized", err)
}

// NotFound return err 404
func NotFound(err error) ErrorAnsw {
	return NewErrorAnsw(http.StatusNotFound, "not found data", err)
}

// NotDeleted return err 406
func NotDeleted(err error) ErrorAnsw {
	return NewErrorAnsw(http.StatusNotAcceptable, "data not deleted", err)
}

// HasExpired return err 408
func HasExpired(err error) ErrorAnsw {
	return NewErrorAnsw(http.StatusRequestTimeout, "has expired", err)
}

// CreateFailed return err 409
func CreateFailed(err error) ErrorAnsw {
	return NewErrorAnsw(http.StatusConflict, "create data failed", err)
}

// NotUpdated return err 409
func NotUpdated(err error) ErrorAnsw {
	return NewErrorAnsw(http.StatusConflict, "data not updated", err)
}

// InternalServerError return err 500
func InternalServerError(err error) ErrorAnsw {
	return NewErrorAnsw(http.StatusInternalServerError, "internal server error", err)
}
