package rmd_test

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/psu-libraries/oats/rmd"
)

const env = "RMD_KEY"
const url = "https://metadata-qa.libraries.psu.edu"

func TestGetUserPubs(t *testing.T) {
	user := "auu4"
	key := os.Getenv(env)
	if key == "" {
		log.Fatal("environment variable not set:", env)
	}
	cli := rmd.NewClient(url, key)
	pubs, err := cli.UserPublications(user)
	if err != nil {
		t.Fatal(err)
	}
	if len(pubs) == 0 {
		t.Fatal("no publications returned")
	}
}

func TestGetAIPubs(t *testing.T) {
	aiUD := "155081269248"
	key := os.Getenv(env)
	if key == "" {
		log.Fatal("environment variable not set:", env)
	}
	cli := rmd.NewClient(url, key)
	pubs, err := cli.PublicationsAI(aiUD)
	if err != nil {
		t.Fatal(err)
	}
	if len(pubs) == 0 {
		t.Fatal("no publications returned")
	}
}

func TestGetDOIPubs(t *testing.T) {
	doi := "10.1093/mnras/staa2325"
	key := os.Getenv(env)
	if key == "" {
		log.Fatal("environment variable not set:", env)
	}
	cli := rmd.NewClient(url, key)
	pubs, err := cli.PublicationsDOI(doi)
	if err != nil {
		t.Fatal(err)
	}
	if len(pubs) == 0 {
		t.Fatal("no publications returned")
	}
}

func TestUpdateLink(t *testing.T) {
	aiUD := "155081269248"
	key := os.Getenv(env)
	if key == "" {
		log.Fatal("environment variable not set:", env)
	}
	cli := rmd.NewClient(url, key)
	err := cli.UpdateScholarSphereLink(aiUD, "nil")
	if err != nil {
		t.Fatal(err)
	}
	pubs, err := cli.PublicationsAI(aiUD)
	if err != nil {
		t.Fatal(err)
	}
	if len(pubs) == 0 {
		t.Fatal("no publications returned")
	}
	fmt.Println()
}
