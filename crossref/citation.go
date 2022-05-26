package crossref

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Citation data
type Citation struct {
	Title          []string `json:"title"`
	Abstract       string   `json:"abstract"`
	Author         []Author `json:"author"`
	ContainerTitle []string `json:"container-title"`
	Publisher      string   `json:"publisher"`
	Subject        []string `json:"subject"`
	Language       string   `json:"language"`
	Type           string   `json:"type"`
	ISSN           []string
	Source         string `json:"source"` // eg CrossRef
	Issued         struct {
		DateParts []DateParts `json:"date-parts"`
	} `json:"issued"`
}

// DateParts [y,m,d]
type DateParts [3]int

// Author field in the citation
type Author struct {
	Family   string `json:"family"`
	Given    string `json:"given"`
	Sequence string `json:"sequence"`
	Name     string `json:"name"`
	ORCID    string
}

// AuthorStrings maps citation authors to a slice of strings using the function
func (c Citation) AuthorStrings(f func(a *Author) string) []string {
	var ret []string
	for i := range c.Author {
		ret = append(ret, f(&c.Author[i]))
	}
	return ret
}

func GetCitation(doi string) (*Citation, error) {
	const (
		serviceURL = `https://api.crossref.org/works/`
	)
	req, err := http.NewRequest(`GET`, serviceURL+doi, nil)
	req.Header.Add("Accept", "application/json")
	if err != nil {
		return nil, err
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Server responded with status: %d", resp.StatusCode)
	}
	//{"status":"ok","message-type":"work","message-version":"1.0.0","message": {...}
	var body struct {
		Status  string
		Message *Citation
	}
	return body.Message, json.NewDecoder(resp.Body).Decode(&body)
}
