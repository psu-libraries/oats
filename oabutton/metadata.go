package oabutton

import (
	"encoding/json"
	"fmt"
)

const (
	findURL = baseURL + "/metadata"
)

// PubMeta data
type PubMeta struct {
	Title     string   `json:"title"`
	Author    []Author `json:"author"`
	Abstract  string   `json:"abstract"`
	Subject   []string `json:"subject"`
	Type      string   `json:"crossref_type"`
	ISSN      string   `json:"issn"`
	Keywords  []string `json:"keywords"`
	Journal   string   `json:"journal"`
	Publisher string   `json:"publisher"`
	Published string   `json:"published"`
	URL       []string `json:"url"`
}

// Author field in the citation
type Author struct {
	Family   string `json:"family"`
	Given    string `json:"given"`
	Sequence string `json:"sequence"`
	ORCID    string
}

// GetPub returns PubMeta for doi
func (c *Client) GetPub(doi string) (*PubMeta, error) {
	url := fmt.Sprintf("%s?id=%s", findURL, doi)
	req, err := c.newReq(`GET`, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating metadata request: %w", err)

	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error during metadata request: %w", err)
	}
	pub := PubMeta{}
	return &pub, json.NewDecoder(resp.Body).Decode(&pub)
}
