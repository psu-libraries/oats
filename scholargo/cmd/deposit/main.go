package main

import (
	"encoding/json"
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
	Key, URL, depositor, metafile string
	newMeta                       bool
)

func main() {
	// command line flags
	flag.StringVar(&depositor, "d", "", "Depositor's PSU Access ID")
	flag.StringVar(&metafile, "m", "", "MetaFile")
	flag.BoolVar(&newMeta, "new-meta", false, "prints empty json meta to stdout")
	flag.Parse()

	if newMeta {
		// print sample json meta to stdout and exit
		printNewMeta()
		return
	}

	files := flag.Args()
	Key = os.Getenv(keyEnvVar)
	URL = os.Getenv(urlEnvVar)

	flagsOK := true
	if depositor == "" {
		log.Println("missing depositor")
		flagsOK = false
	}
	if metafile == "" {
		log.Println("missing metadata file")
		flagsOK = false
	}
	if len(files) == 0 {
		log.Println("missing upload files")
		flagsOK = false
	}
	if Key == "" {
		log.Printf("%s not set\n", keyEnvVar)
		flagsOK = false
	}
	if URL == "" {
		log.Printf("%s not set\n", urlEnvVar)
		flagsOK = false
	}
	if !flagsOK {
		flag.Usage()
		os.Exit(1)
	}

	c := scholargo.Client{Key: Key, BaseURL: URL}

	var meta scholargo.WorkMeta
	metaReader, err := os.Open(metafile)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to read metadata file: %w", err))
	}
	enc := json.NewDecoder(metaReader)
	enc.DisallowUnknownFields()
	err = enc.Decode(&meta)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to read metadata file: %w", err))
	}
	resp, err := c.Deposit(&meta, depositor, files...)
	if err != nil {
		log.Fatal(fmt.Errorf("deposit failed: %w", err))
	}
	fmt.Println("Deposited: ", URL+resp.URL)
}

func printNewMeta() {
	meta := scholargo.WorkMeta{
		WorkType:      "article",
		Title:         "Article Title",
		Description:   "Abstract...",
		PublishedDate: "2020",
		Visibility:    "open",
		Embargo:       "2025-01-01",
		Rights:        "https://creativecommons.org/licenses/by/4.0/",
		Creators: []scholargo.Creator{
			{Orcid: "0000-0001-2345-6789", Name: "Nobody Anderson"}, {PSUID: "sre53"},
		},
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(&meta)
}
