package cmd

// Description of the import process
// - the csv must have the defined columns
// - must check that the csv does not have duplicate values in the ID column
// - check existing Airtable records for rows we are updating (all fields are updated)
//

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"git.psu.edu/sre53/oats/cmd/oats/base"
	"github.com/dimchansky/utfbom"
	"github.com/mehanizm/airtable"
	"github.com/muesli/coral"
)

// required fields for activity insight report
var requiredColumns = []string{
	COL_ID,
	"Version", // renamed from "AI Status"
	"Report_Date",
	"First Name",
	"Middle Name",
	"Last Name",
	"USERNAME",
	"CREATED",
	"LAST_MODIFIED",
	"CONTYPE",
	"STATUS",
	"TITLE",
	"JOURNAL_NAME",
	"TITLE_SECONDARY",
	"REFEREED",
	"VOLUME",
	"ISSUE",
	"DOI",
}

var optional = []string{
	"EDITORS",
	"PUBLISHER",
	"PUBCTYST",
	"ISBNISSN",
	"WEB_ADDRESS",
	"DTY_PUB",
	"DTM_PUB",
	"DTD_PUB",
	"EDITION",
	"PAGENUM",
	"CONTYPEOTHER",
	"PUBLICAVAIL",
	"POST_FILE_1_DOC",
}

// long description
var description = fmt.Sprintf(`The import command creates and updates Activity Insight entries in Airtable
using information from a provided csv file. The csv file must include a
header row. Required and optional column names are listed below.

Required Columns: 
 - %s

Optional Columns:
 - %s`, strings.Join(requiredColumns, "\n - "), strings.Join(optional, "\n - "))

var importCmd = &coral.Command{
	Use:   "import CSV_FILE",
	Short: "Import Activity Insight Records to Airtable",
	Long:  description,
	RunE:  runImport,
	Args:  coral.MinimumNArgs(1),
}

func init() {
	rootCmd.AddCommand(importCmd)
}

func runImport(cmd *coral.Command, args []string) error {
	if oats.Production {
		fmt.Println(`importing csv to production airtable:`, oats.AirtableBase())
	} else {
		fmt.Println(`importing csv to testing airtable:`, oats.AirtableBase())
	}
	if len(args) == 0 {
		return errors.New("expected csv file argument")
	}
	csvFile := args[0]
	file, err := os.Open(csvFile)
	if err != nil {
		return err
	}
	defer file.Close()
	// skip BOM that may appear in CSV -- Excel tends to add them.
	reader := csv.NewReader(utfbom.SkipOnly(file))
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf(`could not read csv: %w`, err)
	}

	csvIDCol, err := checkImportColumns(header, requiredColumns, optional, COL_ID)
	if err != nil {
		return fmt.Errorf(`csv is invalid: %w`, err)
	}
	// read CSV into importRecs
	importRecs := make(map[string]map[string]interface{})
	rowNum := 2 // header is 1
	for {
		row, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf(`error while reading csv: %w`, err)
		}
		id := row[csvIDCol]
		if id == "" {
			return fmt.Errorf(`invalid csv on row %d: ID can't be blank`, rowNum)
		}
		if _, exists := importRecs[id]; exists {
			return fmt.Errorf(`duplicate ID in csv on row %d: %s`, rowNum, id)
		}
		fields := rowToFields(row, header)
		if fields == nil {
			return fmt.Errorf(`empty field data in csv on row %d`, rowNum)
		}
		importRecs[id] = fields
		rowNum++
	}
	log.Println("Indexing existing Activity Insight entries ...")
	// get list of IDs from airtable
	currRecs, err := oats.GetRecordsFilterFields(oats.Airtable.ActivityInsight, ``, []string{COL_ID})
	if err != nil {
		return fmt.Errorf(`failed to retrieve IDS from %s Airtable: %w`, oats.Airtable.ActivityInsight, err)
	}
	currByID, err := base.IndexAirtableRecords(currRecs, COL_ID)
	if err != nil {
		return fmt.Errorf("in Activity Insight Airtable: %w", err)
	}
	var toCreate []*airtable.Record
	var numUpdated int
	for id, fields := range importRecs {
		prevs, exists := currByID[id]
		if !exists {
			toCreate = append(toCreate, &airtable.Record{Fields: fields})
			continue
		}
		if len(prevs) > 1 {
			return fmt.Errorf(`found duplicate Activity Insight entries in Airtable: %s`, id)
		}
		if len(prevs) == 0 {
			return fmt.Errorf(`DEBUG: table index shouldn't have empty entries %s`, id)
		}
		prev := prevs[0]
		if _, err := prev.UpdateRecordPartial(fields); err != nil {
			return fmt.Errorf("failed to update record with ID %s: %w", id, err)
		}
		fmt.Printf("updated Activity Insight ID: %s\n", id)
		numUpdated++
	}
	// create new records in the Activity Insight airtable
	created, err := oats.PostRecords(oats.Airtable.ActivityInsight, toCreate)
	if err != nil {
		return fmt.Errorf(`failed to create new airtable records: %w`, err)
	}
	fmt.Printf("Created %d and Updated %d Activity Insight Records in Airtable.\n", len(created), numUpdated)
	fmt.Println("Use 'tasks' command to create corresponding Task entries")
	return nil
}

// converts the row to a Fields for Airtable
func rowToFields(row []string, header []string) map[string]interface{} {
	if len(row) == 0 {
		return nil
	}
	if len(row) != len(header) {
		return nil
	}
	fields := make(map[string]interface{})
	for i, h := range header {
		// all rows in Activity Insight table are string values
		fields[h] = row[i]
	}
	return fields
}

// check that header has required fields and return ID column id

func checkImportColumns(header []string, required []string, optional []string, id string) (int, error) {
	cols := map[string]bool{}
	req := map[string]bool{}
	opt := map[string]bool{}
	idcol := -1
	for _, r := range required {
		req[r] = true
	}
	for _, o := range optional {
		opt[o] = true
	}
	for i, h := range header {
		if _, exists := cols[h]; exists {
			return 0, fmt.Errorf(`header has duplicate column: %s`, h)
		}
		if !req[h] && !opt[h] {
			return 0, fmt.Errorf(`header has unexpected column: %s`, h)
		}
		if h == id {
			idcol = i
		}
		cols[h] = true
	}
	for r := range req {
		if _, exists := cols[r]; !exists {
			return 0, fmt.Errorf(`header missing required column: %s`, r)
		}
	}
	return idcol, nil
}
