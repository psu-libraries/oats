package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/psu-libraries/oats/rmdb"
)

var aiID string
var doi string

func main() {
	flag.StringVar(&aiID, `ai`, ``, `Activity Insight ID`)
	flag.StringVar(&doi, `doi`, ``, `DOI`)
	flag.Parse()

	key := os.Getenv("RMDB_KEY")
	url := os.Getenv("RMDB_URL")

	if key == "" {
		log.Fatal("RMDB_KEY not set")
	}

	if url == "" {
		log.Fatal("RMDB_URL not set")
	}
	cli := rmdb.NewClient(url, key)

	var query map[string]string
	if aiID != "" {
		query = map[string]string{
			`activity_insight_id`: aiID,
		}
	} else if doi != "" {
		query = map[string]string{
			`doi`: doi,
		}
	} else {
		flag.Usage()
		os.Exit(1)
	}

	pubs, err := cli.PublicationsRaw(query)
	if err != nil {
		log.Fatal(err)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent(``, `  `)
	enc.Encode(pubs)

}
