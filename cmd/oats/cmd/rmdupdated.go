package cmd

import (
	"fmt"
	"log"
	"strings"

	"git.psu.edu/sre53/oats/rmdb"
	"github.com/muesli/coral"
)

const SSLinkPrefix = "https://scholarsphere.psu.edu/resources/"

var rmdUpdatedCmd = &coral.Command{
	Use:   "rmdupdated",
	Short: "Updates Tasks' RMD_Updated column in Airtable",
	Long: `The rmdupdated command updates the RMD_Updated column for Tasks in Airtable
using information from RMD. Only Tasks with RMD_Updated unchecked (false) are
considered. The command will query RMD using the Task's Activity Insight ID;
if a ScholarSphere link is found, the Task's RMD_Updated column is set to
true.`,
	RunE: runRMDUpdated,
}

func init() {
	rootCmd.AddCommand(rmdUpdatedCmd)
}

func runRMDUpdated(cmd *coral.Command, args []string) error {

	// always use rmd production data
	rmdbURL := oats.RMDB.Production
	rmdbC := rmdb.NewClient(rmdbURL, oats.RMDB.APIKey)

	// Query Airtable:
	// return selected columns
	cols := []string{COL_AI_ID, COL_SCHOLINK}
	filter := fmt.Sprintf("NOT({%s})", COL_RMD_UPDATED)
	recs, err := oats.GetRecordsFilterFields(oats.Airtable.Tasks, filter, cols)
	if err != nil {
		return fmt.Errorf(`failed to get airtable records: %w`, err)
	}
	log.Printf("Found %d active tasks that aren't updated in RMD", len(recs))
	allFound := 0
	for _, task := range recs {

		intIDs, ok := task.Fields[COL_AI_ID].([]interface{})
		if !ok || len(intIDs) != 1 {
			return fmt.Errorf(`expected single task with ID %s`, task.ID)
		}
		aiRec, err := oats.GetRecord(oats.Airtable.ActivityInsight, intIDs[0].(string))
		if err != nil {
			return err
		}
		aiID := aiRec.Fields["ID"].(string)
		if aiID == "" {
			return fmt.Errorf(`could not get AI ID for %s`, task.ID)
		}
		airLink, _ := task.Fields[COL_SCHOLINK].(string)

		pubs, err := rmdbC.PublicationsAI(aiID)
		if err != nil {
			return err
		}
		rmdLink := ""
		for _, p := range pubs {
			if strings.HasPrefix(p.Attributes.OAURL, SSLinkPrefix) {
				rmdLink = p.Attributes.OAURL
			}
		}
		var msg string
		if rmdLink != "" {
			update := map[string]interface{}{
				COL_RMD_UPDATED: true,
			}
			msg = fmt.Sprintf("✅ %s: found ScholarSphere link in RMD", aiID)
			if airLink == "" {
				msg = fmt.Sprintf("%s, setting %s", msg, COL_SCHOLINK)
				update[COL_SCHOLINK] = rmdLink
			} else if rmdLink != airLink {
				msg = fmt.Sprintf("%s, RMD/Airtable links don't match", msg)
			}
			_, err = task.UpdateRecordPartial(update)
			if err != nil {
				return fmt.Errorf("error: %w", err)
			}
			allFound++
		} else {
			msg = fmt.Sprintf("❌ %s: no ScholarSphere link in RMD", aiID)
		}
		log.Println(msg)
	}
	log.Printf("Done: set %s for %d tasks\n", COL_RMD_UPDATED, allFound)
	return nil
}
