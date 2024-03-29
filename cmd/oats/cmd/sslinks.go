package cmd

// The sslink command tries to set the ScholarSphere_Link column for Tasks in
// Airtable that have DOIs. It uses the DOI to check if the article represented
// in the Task has already been deposited to ScholarSphere. If it has, the
// ScholarSphere link is set in Airtable. Tasks that already have links are
// ignored.

import (
	"fmt"
	"log"
	"strings"

	"github.com/muesli/coral"
	"github.com/psu-libraries/oats/scholargo"
)

var sslinkCmd = &coral.Command{
	Use:   "sslink",
	Short: "Find ScholarSphere Links for Tasks in Airtable",
	Long: `The sslink command tries to set the ScholarSphere_Link column for Tasks in
Airtable that have DOIs. It uses the DOI to check if the article represented
in the Task has already been deposited to ScholarSphere. If it has, the
ScholarSphere link is set in Airtable. Tasks that already have links are
ignored.`,

	RunE: runSSLink,
}

func init() {
	rootCmd.AddCommand(sslinkCmd)
}

func runSSLink(cmd *coral.Command, args []string) error {
	server := oats.ScholarSphere.Test
	if oats.Production {
		server = oats.ScholarSphere.Production
	}
	// ScholarSphere Client
	cli := scholargo.Client{
		BaseURL: server,
		Key:     oats.ScholarSphere.APIKey,
	}
	scholDOIs, err := cli.DOIs()
	if err != nil {
		return err
	}

	// Query Airtable:
	// filter confirmed and present DOIs
	filter := fmt.Sprintf("AND(LEN({%s})>1,{%s},LEN({ScholarSphere_Link})<4,{Status} != \"Complete\")", COL_DOI, COL_DOI_CONF)
	// return selected columss
	cols := []string{COL_AI_ID, COL_DOI, COL_SCHOLINK}
	recs, err := oats.GetRecordsFilterFields(oats.Airtable.Tasks, filter, cols)
	if err != nil {
		return fmt.Errorf(`failed to get airtable records: %w`, err)
	}

	log.Printf("Found %d active tasks with DOI and no ScholarSphere Link", len(recs))
	for _, r := range recs {

		doi := r.Fields[COL_DOI].(string)
		if doi == "" || doi == "-" {
			continue
		}

		var scholID string
		for d, ids := range scholDOIs {
			if strings.EqualFold(d, "doi:"+doi) {
				if len(ids) == 0 {
					continue
				}
				scholID = ids[0]
				break
			}
		}
		if scholID == "" {
			continue
		}
		link := "https://scholarsphere.psu.edu/resources/" + scholID
		var update = make(map[string]interface{})
		update["ScholarSphere_Link"] = link
		_, err := r.UpdateRecordPartial(update)
		if err != nil {
			return fmt.Errorf(`failed to update task with DOI %s: %w`, doi, err)
		}
		log.Printf("✅ updated %s: %s", doi, link)
	}
	return nil
}
