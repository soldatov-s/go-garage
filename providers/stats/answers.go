package stats

import (
	"encoding/json"
	"net/http"
)

// Body of error answer
type BaseAnswer struct {
	StatusCode int    `json:"statusCode"`
	Details    string `json:"details"`
}

// Body of error answer
type ErrorAnswerBody struct {
	Code string `json:"code"`
	BaseAnswer
}

// Send error answer.
type ErrorAnswer struct {
	Body ErrorAnswerBody `json:"error"`
}

func (answ ErrorAnswer) WriteJSON(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(answ.Body.StatusCode)
	res, err := json.Marshal(answ)
	if err != nil {
		return err
	}
	_, err = w.Write(res)

	return err
}

type ResultAnswer struct {
	Body interface{} `json:"result"`
}

func (answ ResultAnswer) WriteJSON(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	res, err := json.Marshal(answ)
	if err != nil {
		return err
	}
	_, err = w.Write(res)

	return err
}
