package cmd

//The oastatus command updates the OA_status column in Airtable using
// information from the Unpaywall API. Only active Tasks with confirmed DOIs are
// affected. Previous values in the OA_status column may be overwritten because
// their OA status may change. If the OA status is gold or hybrid, the OA_Link
// column will be set using the "best_oa_location" field from Unpaywall.

import (
	"fmt"
	"log"

	"github.com/muesli/coral"
	"github.com/psu-libraries/oats/unpaywall"
)

// Updates OA_status in Airtable using Unpaywall API
var oastatusCmd = &coral.Command{
	Use:   "oastatus",
	Short: "Updates OA_status in Airtable using Unpaywall API",
	Long: `The oastatus command updates the OA_status column in Airtable using
information from the Unpaywall API. Only active Tasks with confirmed DOIs are
affected. Previous values in the OA_status column may be overwritten because
their OA status may change. If the OA status is gold or hybrid, the OA_Link
column will be set using the "best_oa_location" field from Unpaywall.`,
	RunE: runOAStatus,
}

func init() {
	rootCmd.AddCommand(oastatusCmd)
}

func runOAStatus(cmd *coral.Command, args []string) error {
	// unpaywall client
	unclient := unpaywall.NewClient(oats.Unpaywall.Email)
	// Query Airtable: filter confirmed and present DOIs
	filter := fmt.Sprintf("AND(LEN({%s})>1,{%s},{%s} != \"Complete\")", COL_DOI, COL_DOI_CONF, COL_STATUS)
	// return selected columss
	cols := []string{COL_DOI, COL_DOI_CONF, COL_OA_LINK, COL_OA_STATUS}
	recs, err := oats.GetRecordsFilterFields(oats.Airtable.Tasks, filter, cols)
	if err != nil {
		return fmt.Errorf(`failed to get airtable records: %w`, err)
	}
	log.Printf("Found %d active tasks with confirmed DOIs.", len(recs))
	for _, r := range recs {
		doi, _ := r.Fields[COL_DOI].(string)
		oaStatus, _ := r.Fields[COL_OA_STATUS].(string)
		oaLink, _ := r.Fields[COL_OA_LINK].(string)
		if doi == "" {
			log.Print("❌ skipping record with missing DOI")
			continue
		}
		// record from Unpaywall
		unInfo, err := unclient.GetDOI(doi)
		if err != nil {
			log.Printf("❌ %s, Unpaywall error: %s", doi, err.Error())
			continue
		}
		preferredOALink := unInfo.BestOALink.URLpage
		// record should be updated
		var update = make(map[string]interface{})
		if unInfo.OAStatus != "" && oaStatus != unInfo.OAStatus {
			update[COL_OA_STATUS] = unInfo.OAStatus
		}
		if (unInfo.OAStatus == "gold" || unInfo.OAStatus == "hybrid") && oaLink != preferredOALink {
			update[COL_OA_LINK] = preferredOALink
		}
		if len(update) > 0 {
			_, err := r.UpdateRecordPartial(update)
			if err != nil {
				return fmt.Errorf("Stopped during doi=%s because of Airtable update error: %w", doi, err)
			}
			message := fmt.Sprintf("✅ %s:", doi)
			for k, val := range update {
				message += fmt.Sprintf(" %s=%s", k, val.(string))
			}
			log.Println(message)
		} else {
			log.Printf("- no update: %s", doi)
		}

	}
	return nil
}
