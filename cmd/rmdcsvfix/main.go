package main

import (
	"encoding/csv"
	"flag"
	"log"
	"os"
	"strings"
)

const (
	srcCol = "Source [Imports]"            // column with "Activity Insight"
	idCol  = "Source identifier [Imports]" // column with Activity Insight IDs
	ai     = "Activity Insight"            // source string to match

)

var (
	exportCol string // name of input column with values to include in output
	renameCol string // rename column in output
)

func main() {
	flag.StringVar(&exportCol, "c", "", "required column name in input with values to include in output")
	flag.StringVar(&renameCol, "r", "", "new name for column in output")
	flag.Parse()
	filename := flag.Arg(0)
	if filename == "" {
		log.Fatal("input csv required")
	}
	if exportCol == "" {
		log.Fatal("missing required flag: -c COLNAME")
	}
	if renameCol == "" {
		renameCol = exportCol
	}
	f, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	// read input
	allRows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		log.Fatal(err)
	}
	if len(allRows) <= 1 {
		log.Fatal("nothing to do: no rows in input")
	}
	// map header names & validate
	header := map[string]int{}
	for i, n := range allRows[0] {
		header[n] = i
	}
	if _, exists := header[exportCol]; !exists {
		log.Fatalf("input doesn't include column: %s", exportCol)
	}
	if _, exists := header[srcCol]; !exists {
		log.Fatalf("input doesn't include column: %s", srcCol)
	}
	if _, exists := header[idCol]; !exists {
		log.Fatalf("input doesn't include column: %s", idCol)
	}
	// build output
	out := [][]string{{"ID", renameCol}}
	for i, row := range allRows[1:] {
		if len(row) != len(header) {
			log.Fatalf("row %d: different number of columns than header", i+2)
		}
		outval := row[header[exportCol]]
		sources := strings.Split(row[header[srcCol]], ",")
		ids := strings.Split(row[header[idCol]], ",")
		if len(sources) != len(ids) {
			log.Fatalf("row %d: number of sources doesn't match number of source IDs", i+2)
		}
		for j, id := range ids {
			source := sources[j]
			if source == ai {
				out = append(out, []string{id, outval})
			}
		}
	}
	// print output
	err = csv.NewWriter(os.Stdout).WriteAll(out)
	if err != nil {
		log.Fatal(err)
	}
}
