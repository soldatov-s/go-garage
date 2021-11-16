package httpx

import "net/http"

type MiddleWareFunc func(http.Handler) http.Handler
