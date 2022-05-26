package oabutton

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	permissionsURL   = `https://api.openaccessbutton.org/permissions/`
	PublishedVersion = "publishedVersion"
	AcceptedVerstion = "acceptedVersion"
	SubmittedVersion = "submittedVersion"
)

type ErrServer struct {
	Code    int
	Message string
}

func (e *ErrServer) Error() string {
	return fmt.Sprintf("server response (%d): %s", e.Code, e.Message)
}

var ErrNotArticle = &ErrServer{
	Code:    501,
	Message: "DOI is not a journal article",
}

// ArchiveConditions are conditions from OAB's permissions API
// This may change!
type ArchiveConditions struct {
	CanArchive     bool     `json:"can_archive"`
	EmbargoEnd     string   `json:"embargo_end"`
	Version        string   `json:"version"`
	Versions       []string `json:"versions"`
	Locations      []string `json:"locations"`
	License        string   `json:"licence"`
	StatementGuess string   `json:"deposit_statement"`
}

// GetPermissionsVersion calls the permission endpoint and retirns the ArchiveCondtions
// object for the article version with the doi
func (c *Client) GetPermissions(doi string) ([]ArchiveConditions, error) {
	var permResp struct {
		AllPermissions []ArchiveConditions `json:"all_permissions"`
		BestPermission ArchiveConditions   `json:"best_permission"`
	}
	req, err := c.newReq(http.MethodGet, permissionsURL+doi, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating permissions request: %w", err)
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error during permissions request: %w", err)
	}
	if resp.StatusCode != 200 {
		buff := &bytes.Buffer{}
		io.Copy(buff, resp.Body)
		err := &ErrServer{
			Code:    resp.StatusCode,
			Message: buff.String(),
		}
		if *err == *ErrNotArticle {
			return nil, ErrNotArticle
		}
		return nil, err
	}
	err = json.NewDecoder(resp.Body).Decode(&permResp)
	if err != nil {
		return nil, fmt.Errorf("error decoding permissions request response: %w", err)
	}
	return permResp.AllPermissions, nil
}

func (ac ArchiveConditions) ScholarSphereOK() bool {
	if !ac.CanArchive {
		return false
	}
	var goodLocation, goodVersion bool
	for _, l := range ac.Locations {
		if strings.EqualFold("Institutional Repository", l) {
			goodLocation = true
			break
		}
	}
	for _, v := range ac.Versions {
		if v == AcceptedVerstion {
			goodVersion = true
			break
		}
	}
	if goodLocation && goodVersion {
		return true
	}
	return false
}

func (ac ArchiveConditions) BestLicense() string {
	return ac.License
}

// TestPermissionsAPI checks whether the OAB Permissions API is working as expected.
//
func (c *Client) TestPermissionsAPI() error {
	tests := []string{
		"10.1037/apl0000872",
		"10.7191/jeslib.2021.1211",
		"10.1103/PhysRevLett.125.126801",
	}
	var fails int
	for _, t := range tests {
		perms, err := c.GetPermissions(t)
		if err != nil {
			return err
		}
		for _, p := range perms {
			if p.Version == "" || p.License == "" || len(p.Locations) == 0 {
				fails++
			}
		}
	}
	if fails == len(tests) {
		return fmt.Errorf("Permissions response does not include expected values (API may have changed)")
	}
	return nil
}
