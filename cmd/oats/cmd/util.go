package cmd

// Some utility functions used by several commands

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var doiRE *regexp.Regexp = regexp.MustCompile(`10.\d{4,9}\/[-._;()\/:a-z0-9]+`)

func cleanDOI(old string) string {
	doiRE.Longest()
	escaped, err := url.QueryUnescape(old)
	if err == nil {
		old = escaped
	}
	return doiRE.FindString(strings.ToLower(old))
}

func resolvableDOI(doi string) bool {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	doi = cleanDOI(doi)
	if doi == "" {
		return false
	}
	url := "https://doi.org/" + doi
	rsp, err := client.Get(url)
	if err != nil {
		return false
	}
	if rsp.StatusCode != 302 {
		return false
	}
	return true
}
