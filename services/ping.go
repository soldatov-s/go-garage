package services

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	// Default value of ping timeout
	defaultPingRemoteServiceTimeout = time.Second * 3
)

type ServiceHealthStatus struct {
	// Status always contains "ok".
	Status string `json:"status,omitempty"`
}

// PingRemoteService ping aliveURL to checks that the requested service is active
func PingRemoteService(aliveURL string) (*ServiceHealthStatus, error) {
	client := &http.Client{
		Timeout: defaultPingRemoteServiceTimeout,
	}

	response, err := client.Get(aliveURL)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	jData := &ServiceHealthStatus{}

	if len(contents) == 0 {
		jData.Status = "ok"
	} else {
		err = json.Unmarshal(contents, jData)
		if err != nil {
			return nil, err
		}
	}

	return jData, nil
}
