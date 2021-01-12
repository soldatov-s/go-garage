package httpclient

import (
	"net/http"
)

type Pool struct {
	ch           chan *Client
	netTransport *http.Transport
}

func NewPool(clientCfg *ClientConfig, poolCfg *PoolConfig) *Pool {
	p := &Pool{}
	p.ch = make(chan *Client, poolCfg.Size)

	p.netTransport = NewNetTransport(clientCfg)

	for i := 0; i < poolCfg.Size; i++ {
		p.ch <- NewClient(clientCfg, p.netTransport)
	}
	return p
}

func (p *Pool) GetFromPool() *Client {
	return <-p.ch
}

func (p *Pool) PutToPool(client *Client) {
	go func() {
		p.ch <- client
	}()
}
