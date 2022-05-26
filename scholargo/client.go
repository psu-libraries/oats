package scholargo

import (
	"io"
	"net/http"
)

// Client is base object for interacting with ScholarSphere API
type Client struct {
	BaseURL string
	Key     string
}

// NewRequest creates a new request to ScholarSphere API endpoint path.
func (c *Client) NewRequest(method string, path string, body io.Reader) (*http.Request, error) {
	url := c.BaseURL + "/api/v1/" + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("X-API-KEY", c.Key)
	req.Header.Add("Content-Type", "application/json")
	return req, nil
}
