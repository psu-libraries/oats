package cmd

// The dois command attempts to confirm unconfirmed DOIs in Airtable using
// information from CrossRef and RMD. Only active Tasks (Status!="Complete")
// with unconfirmed DOIs (DOI_Confirmed=false) are affected. The confirmation
// process is as follows: If the Task has a DOI value, query CrossRef and
// compare titles; if the titles are similar, the DOI is confirmed. For Tasks
// without DOI values, query RMD using the Activity Insight ID. If a DOI is
// found and the titles in RMD and Airtable are similar, validate the DOI as
// above.

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/hbollon/go-edlib"
	"github.com/mehanizm/airtable"
	"github.com/muesli/coral"
	"github.com/psu-libraries/oats/crossref"
	"github.com/psu-libraries/oats/rmd"
)

type TitleMatchErr struct {
	ID       string
	Expected string
	Got      string
	Source   string
}

func (e *TitleMatchErr) Error() string {
	return fmt.Sprintf("title mismatch for doi=%s (source=%s). Expected=%s, Got=%s", e.ID, e.Source, e.Expected, e.Got)
}

// OAStatusCmd is an oats command for important new
// Activity Insight postprint info
var doisCmd = &coral.Command{
	Use:   "dois",
	Short: "Confirms unconfirmed DOIs in Airtable using CrossRef and RMD",
	Long: `The dois command attempts to confirm unconfirmed DOIs in Airtable using
information from CrossRef and RMD. Only active Tasks (Status!="Complete")
with unconfirmed DOIs (DOI_Confirmed=false) are affected. The confirmation
process is as follows: If the Task has a DOI value, query CrossRef and
compare titles; if the titles are similar, the DOI is confirmed. For Tasks
without DOI values, query RMD using the Activity Insight ID. If a DOI is
found and the titles in RMD and Airtable are similar, validate the DOI as 
above.`,
	RunE: runDOIs,
}

func init() {
	rootCmd.AddCommand(doisCmd)
}

// Run implements Cmd for PermissionsCmd
func runDOIs(cmd *coral.Command, args []string) error {
	// Note: always using production rmb url
	rmdbURL := oats.RMDB.Production
	rmdbC := rmd.NewClient(rmdbURL, oats.RMDB.APIKey)
	// map: Airtable Record ID -> Activity Insight ID
	// Needed to get actual Activity Insight ID for Task
	AIIDlookup := map[string]string{}
	aiRecs, err := oats.GetRecordsFilterFields(oats.Airtable.ActivityInsight, "", []string{COL_ID})
	if err != nil {
		return fmt.Errorf(`failed to get airtable records: %w`, err)
	}
	for _, rec := range aiRecs {
		AIIDlookup[rec.ID], _ = rec.Fields[COL_ID].(string)
	}
	// filter unconfirmed DOIs for active Tasks
	filter := fmt.Sprintf("AND(NOT({%s}),{%s} != \"Complete\")", COL_DOI_CONF, COL_STATUS)
	cols := []string{COL_AI_ID, COL_DOI, COL_DOI_CONF, COL_STATUS, COL_TITLE}
	recs, err := oats.GetRecordsFilterFields(oats.Airtable.Tasks, filter, cols)
	if err != nil {
		return fmt.Errorf(`failed to get airtable records: %w`, err)
	}
	log.Printf("Found %d active tasks with unconfirmed DOIs in Airtable", len(recs))

	for _, r := range recs {
		doi, _ := r.Fields[COL_DOI].(string)
		airTitle, _ := r.Fields[COL_TITLE].(string)
		doi = cleanDOI(doi)

		if doi != "" {
			// If DOI is present, try to confirm with CrossRef
			err := confirmDOICrossRef(doi, airTitle)
			if err != nil {
				var titleErr *TitleMatchErr
				if errors.As(err, &titleErr) {
					log.Printf("❌ %s: Mismatch between titles in Airtable and CrossRef", doi)
					log.Println(" - Airtable:", airTitle)
					log.Println(" - CrossRef:", titleErr.Got)
				} else {
					log.Printf("❌ %s: %s", doi, err.Error())
				}
				continue
			}
			if err := updateConfirmDOI(r, doi); err != nil {
				return err
			}
			continue
		}

		// Try to find DOI from RMD using Activity Insight ID
		var AIID string
		airIDs, ok := r.Fields[COL_AI_ID].([]interface{})
		if !ok || len(airIDs) != 1 {
			return fmt.Errorf(`task not linked to a single Activity Insight record, title=%s`, airTitle)
		}
		AIID = AIIDlookup[airIDs[0].(string)]
		if AIID == "" {
			return fmt.Errorf(`failed to find get Activity Insight ID for title=%s`, airTitle)
		}
		doi, err := confirmRMD(rmdbC, AIID, airTitle)
		if err != nil {
			var titleErr *TitleMatchErr
			if errors.As(err, &titleErr) {
				log.Printf("❌ %s: Mismatch between titles in Airtable and RMD", AIID)
				log.Println(" - Airtable:", airTitle)
				log.Println(" - RMD:", titleErr.Got)
			} else {
				log.Printf("❌ %s: %s", AIID, err.Error())
			}
			continue
		}
		if err := updateConfirmDOI(r, doi); err != nil {
			return err
		}
	}
	return nil
}

func confirmDOICrossRef(doi, title string) error {
	if !resolvableDOI(doi) {
		return fmt.Errorf("DOI does not resolve: %s", doi)
	}
	doiMeta, err := crossref.GetCitation(doi)
	if err != nil {
		return err
	}
	crossTitle := strings.Join(doiMeta.Title, ": ")
	if !similarTitles(title, crossTitle) {
		return &TitleMatchErr{
			ID:       doi,
			Source:   "CrossRef",
			Expected: title,
			Got:      crossTitle,
		}
	}
	return nil
}

func confirmRMD(rmdc *rmd.Client, AIID, title string) (string, error) {
	rmdPubs, err := rmdc.PublicationsAI(AIID)
	if err != nil {
		return "", fmt.Errorf(`RMD request failed for %s: %w`, AIID, err)
	}
	doi := cleanDOI(findPubDOI(rmdPubs))
	if len(rmdPubs) > 0 {
		rmdbTitle := rmdPubs[0].Attributes.CompleteTitle()
		if !similarTitles(title, rmdbTitle) {
			return "", &TitleMatchErr{
				ID:       AIID,
				Source:   "RMD",
				Expected: title,
				Got:      rmdbTitle,
			}
		}

	}
	if doi == "" {
		return "", fmt.Errorf("No DOI for %s in RMD", AIID)
	}
	if err := confirmDOICrossRef(doi, title); err != nil {
		return "", err
	}
	return doi, nil
}

// titles similar
func similarTitles(a, b string) bool {
	// for comparison, remove white space and some characters
	repl := strings.NewReplacer(
		" ", "",
		"\n", "",
		",", "",
		":", "",
		"“", "",
		"“", "",
		`"`, "")
	a = repl.Replace(strings.ToLower(a))
	b = repl.Replace(strings.ToLower(b))
	if a == "" || b == "" {
		return false
	}
	if strings.HasPrefix(a, b) || strings.HasPrefix(b, a) {
		return true
	}
	simVal, err := edlib.StringsSimilarity(a, b, edlib.Levenshtein)
	if err != nil {
		return false
	}
	if simVal < 0.7 {
		return false
	}
	return true
}

// update airtable to confirm doi
func updateConfirmDOI(r *airtable.Record, doi string) error {
	update := make(map[string]interface{})
	update[COL_DOI] = doi
	update[COL_DOI_CONF] = true
	_, err := r.UpdateRecordPartial(update)
	if err != nil {
		return fmt.Errorf("failed to confirm DOI: %w", err)
	}
	log.Printf("✅ confirmed: %s", doi)
	return nil
}

// extract DOIs from an RMD record
func findPubDOI(pubs []rmd.Publication) string {
	var dois []string
	for _, p := range pubs {
		dois = append(dois, strings.ToLower(p.Attributes.DOI))
	}
	if len(dois) == 0 {
		return ""
	}
	if len(dois) == 1 {
		return dois[0]
	}
	a := dois[0]
	for _, d := range dois[1:] {
		if d != a {
			log.Println("DEBUG RMD DOIs", dois)
			return ""
		}
	}
	return a
}
