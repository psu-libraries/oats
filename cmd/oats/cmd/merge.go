package cmd

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/dimchansky/utfbom"
	"github.com/muesli/coral"
	"github.com/psu-libraries/oats/cmd/oats/base"
)

var mergeCmd = &coral.Command{
	Use:   "merge CSV_FILE",
	Short: "Updates Tasks on Airtable with data from a csv file",
	Long: `The merge command updates existing Tasks in Airtable using data from a
provided CSV file. The CSV file must include columns with names matching
columns of the Tasks Airtable. The ID column is the only required column:
it is used to join rows in the CSV file to rows in the Tasks table.`,
	RunE: runMerge,
	Args: coral.MinimumNArgs(1),
}

func init() {
	rootCmd.AddCommand(mergeCmd)
}

func runMerge(cmd *coral.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("expected csv file argument")
	}
	csvFile := args[0]

	if oats.Production {
		fmt.Println(`merging csv to production airtable:`, oats.AirtableBase())
	} else {
		fmt.Println(`merging csv to testing airtable:`, oats.AirtableBase())
	}

	file, err := os.Open(csvFile)
	if err != nil {
		return err
	}
	defer file.Close()
	reader := csv.NewReader(utfbom.SkipOnly(file))
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf(`could not read csv: %w`, err)
	}

	var csvIDCol = -1
	for i, col := range header {
		if col == COL_ID {
			csvIDCol = i
		}
	}
	if csvIDCol == -1 {
		return fmt.Errorf(`missing required column: %s`, COL_ID)
	}

	// read CSV into updateRecs
	updateRecs := make(map[string]map[string]interface{})
	rowNum := 2 // header is 1
	for {
		row, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf(`error while reading csv: %w`, err)
		}
		rowNum++
		id := row[csvIDCol]
		if id == "" {
			return fmt.Errorf(`invalid csv on row %d: ID can't be blank`, rowNum)
		}
		if _, exists := updateRecs[id]; exists {
			return fmt.Errorf(`duplicate ID in csv on row %d: %s`, rowNum, id)
		}
		fields := rowToFields(row, header)
		if fields == nil {
			return fmt.Errorf(`empty field data in csv on row %d`, rowNum)
		}
		updateRecs[id] = fields
		rowNum++
	}

	// get list of IDs from airtable
	currRecs, err := oats.GetRecordsFilterFields(oats.Airtable.ActivityInsight, ``, []string{COL_ID, "Tasks"})
	if err != nil {
		return fmt.Errorf(`failed to retrieve IDS from %s Airtable: %w`, oats.Airtable.ActivityInsight, err)
	}
	indexCurr, err := base.IndexAirtableRecords(currRecs, COL_ID)
	if err != nil {
		return err
	}

	for id, fields := range updateRecs {
		prevs, present := indexCurr[id]
		if !present {
			fmt.Printf("ID %s doesn't exist\n", id)
			continue
		}
		if len(prevs) != 1 {
			return fmt.Errorf(`expected exactly one entry in the Activity Insight Airtable with the id: %s`, id)
		}
		prev := prevs[0]
		taskIDs, ok := prev.Fields["Tasks"].([]interface{})
		if !ok {
			return fmt.Errorf("failed to get Task information for %s", id)
		}
		if len(taskIDs) == 0 {
			return fmt.Errorf("no Task associated with ID %s", id)
		}
		taskID, ok := taskIDs[0].(string)
		if !ok {
			return fmt.Errorf("failed to get Task information for %s", id)
		}
		rec, err := oats.GetRecord(oats.Airtable.Tasks, taskID)
		if err != nil {
			return err
		}
		delete(fields, COL_ID)
		_, err = rec.UpdateRecordPartial(fields)
		if err != nil {
			return fmt.Errorf("failed to update record with ID %s: %w", id, err)
		}
		fmt.Printf("updated Activity Insight entry %s\n", id)
		continue
	}

	// // create new records in the Activity Insight airtable
	// created, err := cmd.postRecords(cmd.config.Airtable.ActivityInsight, toCreate)
	// if err != nil {
	// 	return fmt.Errorf(`failed to create new airtable records: %w`, err)
	// }
	// fmt.Printf("created %d records\n", len(created))
	// return nil

	return nil
}
