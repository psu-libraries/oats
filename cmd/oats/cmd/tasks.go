package cmd

// The tasks command creates new Tasks in Airtable for Activity Insight entries
// that do not have a corresponding Task.

import (
	"fmt"
	"log"
	"strings"

	"github.com/mehanizm/airtable"
	"github.com/muesli/coral"
)

var tasksCmd = &coral.Command{
	Use:   "tasks",
	Short: "Creates new Tasks in Airtable",
	Long: `The tasks command creates new Tasks in Airtable for Activity Insight entries that
do not have a corresponding Task. Tasks are created with empty "Status" and
"Assigned_To" values. Values for the Task are set as follows:
- Title: from AI TITLE field
- Article_Version: from AI Version field
- DOI: from AI DOI field (cleaned)
- Journal_Name: from AI JOURNAL_NAME
- User: from AI USERNAME`,
	RunE: runTasks,
}

func init() {
	rootCmd.AddCommand(tasksCmd)
}

func runTasks(cmd *coral.Command, args []string) error {
	// all the records in the Activity Insight table without a corresponding Task
	needTasks, err := oats.GetRecordsFilterFields(oats.Airtable.ActivityInsight, `{Tasks} = ''`, nil)
	if err != nil {
		return fmt.Errorf(`failed to retrieve IDS from %s Airtable: %w`, oats.Airtable.ActivityInsight, err)
	}
	log.Printf("found %d Activity Insight entries without Tasks", len(needTasks))
	newTasks := make([]*airtable.Record, len(needTasks))
	for i, ai := range needTasks {
		newTasks[i] = newTask(ai)
	}
	created, err := oats.PostRecords(oats.Airtable.Tasks, newTasks)
	if err != nil {
		return fmt.Errorf("failed to create tasks. Airtable Error Response: %w", err)
	}

	fmt.Printf("%d new tasks created\n", len(created))
	return nil
}

// returns a corresponding task for the AI record
func newTask(ai *airtable.Record) *airtable.Record {
	fields := make(map[string]interface{})
	// copy some fields from Activity Insight Record to the Task
	fields[COL_AI_ID] = []interface{}{ai.ID}
	fields[COL_VERSION] = ai.Fields["Version"]
	if doi, ok := ai.Fields["DOI"].(string); ok {
		fields[COL_DOI] = cleanDOI(doi)
	}
	fields[COL_TITLE] = ai.Fields["TITLE"]
	fields[COL_JOURNAL] = ai.Fields["JOURNAL_NAME"]
	if user, ok := ai.Fields["USERNAME"].(string); ok {
		fields[COL_USER] = strings.ToLower(user)
	}
	return &airtable.Record{Fields: fields}
}
