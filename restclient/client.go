package restclient

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

type Config struct {
	AliveURL string
}

type Client struct {
	httpClient *http.Client
	aliveURL   string
}

func DefaultHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 5 * time.Second,
	}
}

type ClientOption func(*Client)

func WithCustomHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

type ServiceHealthStatus struct {
	// Status always contains "ok".
	Status string `json:"status,omitempty"`
}

func NewClient(cfg *Config, opts ...ClientOption) *Client {
	client := &Client{
		aliveURL: cfg.AliveURL,
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// PingRemoteService ping aliveURL to checks that the requested service is active
func (c *Client) Ping(ctx context.Context) (*ServiceHealthStatus, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.aliveURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "build request")
	}
	response, err := c.httpClient.Do(request)
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
