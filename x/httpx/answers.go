package httpx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
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

func (e *ErrorAnsw) WriteJSON(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.Body.StatusCode)
	res, err := json.Marshal(e)
	if err != nil {
		return errors.Wrap(err, "marshal answer")
	}
	if _, err := w.Write(res); err != nil {
		return errors.Wrap(err, "write data to connection")
	}

	return nil
}

type ResultAnsw struct {
	Body interface{} `json:"result"`
}

func (answ *ResultAnsw) WriteJSON(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	res, err := json.Marshal(answ)
	if err != nil {
		return errors.Wrap(err, "marshal answer")
	}
	if _, err := w.Write(res); err != nil {
		return errors.Wrap(err, "write data to connection")
	}

	return nil
}

// OkResult return OK answer
func OkResult() ResultAnsw {
	return ResultAnsw{Body: "OK"}
}

func WriteErrAnswer(ctx context.Context, w http.ResponseWriter, err error, code string) {
	logger := zerolog.Ctx(ctx)
	answ := ErrorAnsw{
		Body: ErrorAnswBody{
			Code:       code,
			StatusCode: http.StatusServiceUnavailable,
			Details:    err.Error(),
		},
	}

	w.WriteHeader(http.StatusFailedDependency)
	if errWriteJSON := answ.WriteJSON(w); errWriteJSON != nil {
		logger.Err(errWriteJSON).Msg("write json")
	}
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

func NewStandartErrorAnsw(statusCode int, err error) ErrorAnsw {
	return NewErrorAnsw(statusCode, http.StatusText(statusCode), err)
}

func BadRequest(err error) ErrorAnsw {
	return NewStandartErrorAnsw(http.StatusBadRequest, err)
}

func Unauthorized(err error) ErrorAnsw {
	return NewStandartErrorAnsw(http.StatusUnauthorized, err)
}

func Forbidden(err error) ErrorAnsw {
	return NewStandartErrorAnsw(http.StatusForbidden, err)
}

func NotFound(err error) ErrorAnsw {
	return NewStandartErrorAnsw(http.StatusNotFound, err)
}

func NotAcceptable(err error) ErrorAnsw {
	return NewStandartErrorAnsw(http.StatusNotAcceptable, err)
}

func RequestTimeout(err error) ErrorAnsw {
	return NewStandartErrorAnsw(http.StatusRequestTimeout, err)
}

func Conflict(err error) ErrorAnsw {
	return NewStandartErrorAnsw(http.StatusConflict, err)
}

func InternalServerError(err error) ErrorAnsw {
	return NewStandartErrorAnsw(http.StatusInternalServerError, err)
}
