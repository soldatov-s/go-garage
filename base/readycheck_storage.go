// nolint:dupl // duplication
package base

import (
	"context"
	"net/http"

	"github.com/rs/zerolog"
	"github.com/soldatov-s/go-garage/x/httpx"
)

type ReadyCheckStorage struct {
	readyCheck *MapCheckOptions
}

func NewReadyCheckStorage() *ReadyCheckStorage {
	return &ReadyCheckStorage{
		readyCheck: NewMapCheckOptions(),
	}
}

func (s *ReadyCheckStorage) GetReadyHandlers() *MapCheckOptions {
	return s.readyCheck
}

func (s *ReadyCheckStorage) ReadyCheckHandler(ctx context.Context, w http.ResponseWriter) {
	logger := zerolog.Ctx(ctx)
	for key, checkFunc := range s.readyCheck.options {
		if err := checkFunc.CheckFunc(ctx); err != nil {
			httpx.WriteErrAnswer(ctx, w, err, key)
			return
		}
	}

	answ := httpx.ResultAnsw{Body: "ok"}
	if err := answ.WriteJSON(w); err != nil {
		logger.Err(err).Msg("write json")
	}
}
