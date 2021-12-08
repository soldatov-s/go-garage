// Based on KromDaniel/rejonson
package rejson

import (
	"github.com/go-redis/redis/v8"
)

func NewEmptyClient() *Client {
	return &Client{}
}

func (cl *Client) SetConn(client *redis.Client) {
	cl.Client = client
	cl.redisProcessor = &redisProcessor{
		Process: client.Process,
	}
}

func ExtendClient(client *redis.Client) *Client {
	return &Client{
		client,
		&redisProcessor{
			Process: client.Process,
		},
	}
}

func ExtendPipeline(pipeline redis.Pipeliner) *Pipeline {
	return &Pipeline{
		pipeline,
		&redisProcessor{
			Process: pipeline.Process,
		},
	}
}
