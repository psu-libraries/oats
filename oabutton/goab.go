package oabutton

import (
	"io"
	"net/http"
	"time"
)

const (
	baseURL = `https://api.openaccessbutton.org`
)

// Client is an http client for calling OAB endpoit
type Client struct {
	http.Client
	key string
}

// NewClient returns new OABClient
func NewClient(k string) *Client {
	return &Client{
		Client: http.Client{Timeout: 15 * time.Second},
		key:    k,
	}
}

func (c *Client) newReq(m, u string, b io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(m, u, b)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/json")
	if c.key != "" {
		req.Header.Add("x-apikey", c.key)
	}
	return req, nil
}
