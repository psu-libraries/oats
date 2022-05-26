package cmd

import (
	"errors"
	"fmt"
	"log"
	"time"

	"git.psu.edu/sre53/oats/oabutton"
	"github.com/muesli/coral"
)

// new column values
const (
	PERM_OPEN     = "Accepted Version OK"
	PERM_CLOSED   = "No Policy (Accepted Version)"
	PERM_NOTFOUND = "No Policy Found: OAB"
	PERMSRC       = "OAB"
)

// OA Button load permissions
var permissionsCmd = &coral.Command{
	Use:   "permissions",
	Short: "Updates deposit permissions in Airtable using Open Access Button's Permissions API",
	Long: `The permissions command updates deposit permissions information in Airtable
using article information from Open Access Button's Permissions API. Only
Tasks with confirmed DOIs and empty Permissions values are affected.
Permissions_Source is set to "OAB". The Permissions column is set to one of:
 - Accepted Version OK
 - No Policy (Accepted Version)
 - No Policy Found: OAB
Other columns that may be updated are:
 - License
 - Embargo_end
 - Set_Statement`,
	RunE: runPermissions,
}

func init() {
	rootCmd.AddCommand(permissionsCmd)
}

func runPermissions(cmd *coral.Command, args []string) error {
	oabc := oabutton.NewClient(oats.OpenAccessButton.Key)

	// test Permissions API
	if err := oabc.TestPermissionsAPI(); err != nil {
		return fmt.Errorf("OAB Permissions API is not behaving as expected: %w", err)
	}

	// Query Airtable: filter confirmed and present DOIs and no Permissions
	filter := fmt.Sprintf("AND(LEN({%s})>1,{%s},NOT({%s}))", COL_DOI, COL_DOI_CONF, COL_PERM)
	// return selected columns
	recs, err := oats.GetRecordsFilterFields(oats.Airtable.Tasks, filter, []string{COL_DOI})
	if err != nil {
		return fmt.Errorf(`failed to get airtable records: %w`, err)
	}
	log.Printf("Found %d records with confirmed DOIs and no set permissions", len(recs))
	for _, r := range recs {
		doi, ok := r.Fields[COL_DOI].(string)
		if !ok {
			log.Print("❌ missing DOI")
			continue
		}
		time.Sleep(500 * time.Millisecond)
		perms, err := oabc.GetPermissions(doi)
		if err != nil && !errors.Is(err, oabutton.ErrNotArticle) {
			log.Printf("❌ unexpected error from OAB Permissions API, %s: %s", doi, err.Error())
			continue
		}
		if errors.Is(err, oabutton.ErrNotArticle) || len(perms) == 0 {
			_, err := r.UpdateRecordPartial(map[string]interface{}{
				COL_PERM:     PERM_NOTFOUND,
				COL_PERM_SRC: PERMSRC,
			})
			if err != nil {
				return fmt.Errorf("Stopped during doi=%s because of Airtable update error: %w", doi, err)
			}
			log.Printf("❌ no policies found for %s (%s=%s)", doi, COL_PERM, PERM_NOTFOUND)
			continue
		}
		var perm oabutton.ArchiveConditions
		for i := range perms {
			if perms[i].ScholarSphereOK() {
				perm = perms[i]
				break
			}
		}
		if !perm.ScholarSphereOK() {
			_, err := r.UpdateRecordPartial(map[string]interface{}{
				COL_PERM:     PERM_CLOSED,
				COL_PERM_SRC: PERMSRC,
			})
			if err != nil {
				return fmt.Errorf("Stopped during doi=%s because of Airtable update error: %w", doi, err)
			}
			log.Printf("✅ updated %s (%s=%s)\n", doi, COL_PERM, PERM_CLOSED)
			continue
		}
		license := perm.BestLicense()
		if license == "" {
			license = "other-closed"
		}
		_, err = r.UpdateRecordPartial(map[string]interface{}{
			COL_EMBARGO:  perm.EmbargoEnd,
			COL_STMNT:    perm.StatementGuess,
			COL_LICENSE:  license,
			COL_PERM:     PERM_OPEN,
			COL_PERM_SRC: PERMSRC,
		})
		if err != nil {
			return fmt.Errorf("Stopped during doi=%s because of Airtable update error: %w", doi, err)
		}
		log.Printf("✅ updated %s (%s=%s)\n", doi, COL_PERM, PERM_OPEN)
	}
	return nil
}
