package unpaywall

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const (
	baseURL     = `https://api.unpaywall.org/v2`
	minDelay    = time.Millisecond * 250
	maxRequests = 100_000
)

type DOIResp struct {
	DOI           string `json:"doi"`
	Title         string
	Genre         string
	Year          int
	PublishedDate string `json:"published_date"`
	JournalName   string `json:"journal_name"`
	OAStatus      string `json:"oa_status"`
	BestOALink    struct {
		URL      string `json:"url"`
		URLpage  string `json:"url_for_landing_page"`
		URLpdf   string `json:"url_for_pdf"`
		Version  string
		License  string
		HostType string `json:"host_type"`
	} `json:"best_oa_location"`
}

// Client is an http client for calling OAB endpoit
type Client struct {
	http.Client
	email        string
	requestCount int
	lastRequest  time.Time
}

// NewClient returns new OABClient
func NewClient(email string) *Client {
	return &Client{
		email: email,
	}
}

func (c *Client) GetDOI(doi string) (*DOIResp, error) {
	if c.requestCount > maxRequests {
		return nil, errors.New(`too many requests to unpaywall`)
	}
	url := fmt.Sprintf("%s/%s?email=%s", baseURL, doi, c.email)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/json")
	time.Sleep(time.Until(c.lastRequest.Add(minDelay)))
	c.lastRequest = time.Now()
	c.requestCount++
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf(`HTTP Status: %s`, resp.Status)
	}
	var ret DOIResp
	return &ret, json.NewDecoder(resp.Body).Decode(&ret)
}
