package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPingRemoteService(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		res, err := json.Marshal(&ServiceHealthStatus{Status: "ok"})
		if err != nil {
			t.Fatal(err)
		}
		_, err = w.Write(res)
		if err != nil {
			t.Fatal(err)
		}
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	healthStatus, err := PingRemoteService(server.URL)

	require.Nil(t, err)
	require.NotNil(t, healthStatus)
}

func TestPingRemoteServiceEmptyAnswer(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {}

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	healthStatus, err := PingRemoteService(server.URL)

	require.Nil(t, err)
	require.NotNil(t, healthStatus)
}

func TestPingRemoteServiceNotExistServer(t *testing.T) {
	healthStatus, err := PingRemoteService("http://localhost:9999")

	require.NotNil(t, err)
	require.Nil(t, healthStatus)
}
