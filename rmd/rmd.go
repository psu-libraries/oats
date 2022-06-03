package rmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type Client struct {
	http.Client
	baseURL string
	key     string
}

type ServerErr struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (err *ServerErr) Error() string {
	return fmt.Sprintf("server error: %s [%d]", err.Message, err.Code)
}

type PublicationsResponse struct {
	Data []Publication `json:"data"`
}

type Publication struct {
	ID         string           `json:"id"`
	Type       string           `json:"type"`
	Attributes PublicationAttrs `json:"attributes"`
}

type PublicationAttrs struct {
	Title           string        `json:"title"`
	SecondaryTitle  string        `json:"secondary_title"`
	JournalTitle    string        `json:"journal_title"`
	PublicationType string        `json:"publication_type"`
	Publisher       string        `json:"publisher"`
	Status          string        `json:"status"`
	Volume          string        `json:"volume"`
	Issue           string        `json:"issue"`
	Edition         string        `json:"edition"`
	PageRange       string        `json:"page_range"`
	AuthorsEtAl     bool          `json:"authors_et_al"`
	Abstract        string        `json:"abstract"`
	DOI             string        `json:"doi"`
	OAURL           string        `json:"preferred_open_access_url"`
	PublishedOn     string        `json:"published_on"`
	CitationCount   int           `json:"citation_count"`
	Contributors    []Contributor `json:"contributors"`
	Tags            []struct {
		Name string  `json:"name"`
		Rank float32 `json:"rank"`
	}
	PureIDS                        []string `json:"pure_ids"`
	ActivityInsightIDS             []string `json:"activity_insight_ids"`
	ActivityInsightPostPrintStatus string   `json:"activity_insight_postprint_status"`
}

type Contributor struct {
	FirstName  string `json:"first_name"`
	MiddleName string `json:"middle_name"`
	LastName   string `json:"last_name"`
	PSUID      string `json:"psu_user_id"`
}

func NewClient(url, k string) *Client {
	return &Client{
		baseURL: url,
		key:     k,
	}
}

func (c *Client) newReq(m, u string, b io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(m, u, b)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	if c.key != "" {
		req.Header.Add("X-API-Key", c.key)
	}
	return req, nil
}

func (c *Client) publications(filter map[string]string) ([]Publication, error) {
	query := url.Values{}
	for k, v := range filter {
		query.Set(k, v)
	}
	u := c.baseURL + "/v1/publications?" + query.Encode()
	req, err := c.newReq("GET", u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		var srvErr ServerErr
		err := json.NewDecoder(resp.Body).Decode(&srvErr)
		if err != nil {
			return nil, err
		}
		return nil, &srvErr
	}
	var pubs PublicationsResponse
	err = json.NewDecoder(resp.Body).Decode(&pubs)
	if err != nil {
		return nil, err
	}
	return pubs.Data, nil
}

func (c *Client) PublicationsRaw(filter map[string]string) (interface{}, error) {
	query := url.Values{}
	for k, v := range filter {
		query.Set(k, v)
	}
	u := c.baseURL + "/v1/publications?" + query.Encode()
	req, err := c.newReq("GET", u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		var srvErr ServerErr
		err := json.NewDecoder(resp.Body).Decode(&srvErr)
		if err != nil {
			return nil, err
		}
		return nil, &srvErr
	}
	var raw interface{}
	err = json.NewDecoder(resp.Body).Decode(&raw)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func (c *Client) PublicationsAI(aiID string) ([]Publication, error) {
	query := map[string]string{
		"activity_insight_id": aiID,
	}
	return c.publications(query)
}

func (c *Client) PublicationsDOI(doi string) ([]Publication, error) {
	query := map[string]string{
		"doi": doi,
	}
	return c.publications(query)
}

func (c *Client) UserPublications(user string) ([]Publication, error) {
	if user == "" {
		return nil, errors.New("cannot get publications for empty user")
	}

	url := fmt.Sprintf("%s/v1/users/%s/publications", c.baseURL, user)
	req, err := c.newReq("GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		var srvErr ServerErr
		err := json.NewDecoder(resp.Body).Decode(&srvErr)
		if err != nil {
			return nil, err
		}
		return nil, &srvErr
	}
	var pubs PublicationsResponse
	err = json.NewDecoder(resp.Body).Decode(&pubs)
	if err != nil {
		return nil, err
	}
	return pubs.Data, nil
}

// UpdateScholarSpherLink submits a request to update the scholarsphere
// link for records with the activity insight ID aiID
func (c *Client) UpdateScholarSphereLink(aiID, link string) error {
	update := map[string]string{
		"activity_insight_id":           aiID,
		"scholarsphere_open_access_url": link,
	}
	b, err := json.Marshal(update)
	if err != nil {
		return err
	}
	req, err := c.newReq("PATCH", c.baseURL+"/v1/publications", bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("new request:%w", err)
	}
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("request:%w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		var srvErr ServerErr
		err := json.NewDecoder(resp.Body).Decode(&srvErr)
		if err != nil {
			return err
		}
		return &srvErr
	}
	return nil
}

func (attr *PublicationAttrs) CompleteTitle() string {
	if attr.SecondaryTitle == "" {
		return attr.Title
	}
	return fmt.Sprintf("%s: %s", attr.Title, attr.SecondaryTitle)
}
