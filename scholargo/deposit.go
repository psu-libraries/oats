package scholargo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Work metadata sent with deposit
type WorkMeta struct {
	// required
	WorkType      string    `json:"work_type"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	Creators      []Creator `json:"creators"`
	Visibility    string    `json:"visibility"`
	Rights        string    `json:"rights"`
	PublishedDate string    `json:"published_date"`
	// optional
	Embargo            string   `json:"embargoed_until,omitempty"`
	Subtitle           string   `json:"subtitle,omitempty"`
	PublisherStatement string   `json:"publisher_statement,omitempty"`
	Keyword            []string `json:"keyword,omitempty"`
	Identifier         []string `json:"identifier,omitempty"`
	RelatedURL         []string `json:"related_url,omitempty"`
	Source             []string `json:"source,omitempty"`
	BasedNear          []string `json:"based_near,omitempty"`
	Language           []string `json:"language,omitempty"`
	Subject            []string `json:"subject,omitempty"`
	Publisher          []string `json:"publisher,omitempty"`
	Contributor        []string `json:"contributor,omitempty"`
}

// Creator information sent with deposi
type Creator struct {
	Name  string `json:"display_name,omitempty"`
	Email string `json:"email,omitempty"`
	PSUID string `json:"psu_id,omitempty"`
	Orcid string `json:"orcid,omitempty"`
}

type Permissions struct{}

// DepositResponse is API response on successful deposit
type DepositResponse struct {
	Message string `json:"message"`
	URL     string `json:"url"`
}

// deposit is object POSTed to API's ingest endpoint.
type deposit struct {
	Metadata    *WorkMeta    `json:"metadata"`
	Content     []content    `json:"content"`
	Depositor   string       `json:"depositor"`
	Permissions *Permissions `json:"permissions"`
}

// content wraps file info in a json-encoded string.
type content struct {
	File string `json:"file"`
}

func (c *Client) Deposit(meta *WorkMeta, depositor string, files ...string) (*DepositResponse, error) {
	var conts []content
	for _, f := range files {
		up, err := c.upload(f)
		if err != nil {
			return nil, fmt.Errorf("upload %s failed: %w", f, err)
		}
		// Content is json-encoded string
		wrapped, err := json.Marshal(&up)
		if err != nil {
			return nil, fmt.Errorf("upload %s failed: %w", f, err)
		}
		conts = append(conts, content{File: string(wrapped)})
	}
	d := deposit{
		Metadata:    meta,
		Content:     conts,
		Depositor:   depositor,
		Permissions: &Permissions{},
	}
	body, err := json.Marshal(&d)
	if err != nil {
		return nil, err
	}
	req, err := c.NewRequest("POST", "ingest", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		buff := &bytes.Buffer{}
		io.Copy(buff, resp.Body)
		return nil, fmt.Errorf("server error %d: %s", resp.StatusCode, buff.String())
	}
	var depResp DepositResponse
	err = json.NewDecoder(resp.Body).Decode(&depResp)
	if err != nil {
		return nil, err
	}
	return &depResp, nil
}
