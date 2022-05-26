package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/psu-libraries/oats/scholargo"
)

const (
	keyEnvVar = "SCHOLARSPHERE_KEY"
	urlEnvVar = "SCHOLARSPHERE_URL"
)

var (
	Key, URL, doi string
)

func main() {
	flag.Parse()
	doi = flag.Arg(0)
	Key = os.Getenv(keyEnvVar)
	URL = os.Getenv(urlEnvVar)

	if doi == "" {
		log.Fatal("missing doi to check")
	}

	c := scholargo.Client{Key: Key, BaseURL: URL}
	dois, err := c.DOIs()
	if err != nil {
		log.Fatal(fmt.Errorf("failed to get DOIs: %w", err))
	}

	ids, exists := dois[doi]
	if !exists {
		log.Fatalf("DOI not found %s\n", doi)
	}
	for _, id := range ids {
		fmt.Println(id)
	}
}
