// nolint:dupl // duplication
package base

import (
	"context"
	"net/http"

	"github.com/rs/zerolog"
	"github.com/soldatov-s/go-garage/x/httpx"
)

type AliveCheckStorage struct {
	aliveCheck *MapCheckOptions
}

func NewAliveCheckStorage() *AliveCheckStorage {
	return &AliveCheckStorage{
		aliveCheck: NewMapCheckOptions(),
	}
}

func (s *AliveCheckStorage) GetAliveHandlers() *MapCheckOptions {
	return s.aliveCheck
}

func (s *AliveCheckStorage) AliveCheckHandler(ctx context.Context, w http.ResponseWriter) {
	logger := zerolog.Ctx(ctx)
	for key, checkFunc := range s.aliveCheck.options {
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
