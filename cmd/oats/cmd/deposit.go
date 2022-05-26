/*
Copyright © 2022

*/
package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/muesli/coral"
	"github.com/psu-libraries/oats/crossref"
	"github.com/psu-libraries/oats/rmdb"
	"github.com/psu-libraries/oats/scholargo"
)

var depositFlags struct {
	filePath   string
	skipStatus bool
	skipPerm   bool
	skipRMD    bool
}

var depositCmd = &coral.Command{
	Use:   "deposit ID",
	Short: "Deposit to ScholarSphere",
	Long: `The deposit command deposits articles to ScholarSphere using information
in Airtable. The Task's ID (Activity Insight) is a required argument. By
default, the Task must have 'Status'='To Deposit' and
'Permissions'='Accepted Version OK', however there are options to skip
these checks. The file matching the 'POST_FILE_1_DOC' value in the most
recent Activity Insight export is used for the deposit. The file search
is scoped to the directory set with the 'article_path' configuration.
Deposit metadata is based on data in RMD, CrossRef, and the Task table.`,
	RunE: runDeposit,
	Args: coral.MinimumNArgs(1),
}

func init() {
	rootCmd.AddCommand(depositCmd)
	depositCmd.Flags().StringVarP(&depositFlags.filePath, "file", "f", "", "file to deposit")
	depositCmd.Flags().BoolVarP(&depositFlags.skipStatus, "no-status", "", false, "skip check: Status='To Deposit'")
	depositCmd.Flags().BoolVarP(&depositFlags.skipPerm, "no-permissions", "", false, "skip check: Permissions='Accepted Version OK'")
	depositCmd.Flags().BoolVarP(&depositFlags.skipRMD, "skip-rmd", "", false, "don't do RMD update")
}

func runDeposit(cmd *coral.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("expected deposit id")
	}

	// Activity Insight ID for Task
	depositID := args[0]

	// api endpoints
	scholURL := oats.Config.ScholarSphere.Test
	rmdbURL := oats.Config.RMDB.Test
	if oats.Production {
		scholURL = oats.Config.ScholarSphere.Production
		rmdbURL = oats.Config.RMDB.Production
	}
	log.Printf("using airtable=%s, scholarsphere=%s, rmd=%s", oats.AirtableBase(), scholURL, rmdbURL)

	//RMDB client
	rmdbCli := rmdb.NewClient(rmdbURL, oats.RMDB.APIKey)
	var rmdPubs []rmdb.Publication // RMD publications with depositID

	// ScholarSphere Client
	schol := scholargo.Client{
		BaseURL: scholURL,
		Key:     oats.Config.ScholarSphere.APIKey,
	}
	// big list of DOIS in ScholarSphere - used to check existing deposit
	scholDOIs, err := schol.DOIs()
	if err != nil {
		return fmt.Errorf(`❌ failed to get current DOIs from ScholarSphere: %w`, err)
	}

	// Get Activity Insight and Task records from Airtable
	filter := fmt.Sprintf("{%s} = '%s'", COL_ID, depositID)
	aiRecs, err := oats.GetRecordsFilterFields(oats.Airtable.ActivityInsight, filter, nil)
	if err != nil {
		return fmt.Errorf(`❌ failed to get Airtable records: %w`, err)
	}
	if l := len(aiRecs); l != 1 {
		return fmt.Errorf(`❌ %s: expected exactly 1 Activity Insight record, found %d`, depositID, l)
	}
	aiRec := aiRecs[0]
	intTaskIDs, ok := aiRec.Fields["Tasks"].([]interface{})
	if l := len(intTaskIDs); !ok || l != 1 {
		return fmt.Errorf(`❌ %s: expected 1 Task record, found %d`, depositID, l)
	}
	taskRec, err := oats.GetRecord(oats.Airtable.Tasks, intTaskIDs[0].(string))
	if err != nil {
		return fmt.Errorf("❌ %s: %w", depositID, err)
	}

	// check that deposit is appropriate
	if !depositFlags.skipStatus {
		status, _ := taskRec.Fields[COL_STATUS].(string)
		if status != `To Deposit` {
			return fmt.Errorf("❌ %s: task status not 'To Deposit'", depositID)
		}
	}
	if !depositFlags.skipPerm {
		oabPermissions, _ := taskRec.Fields[COL_PERM].(string)
		if oabPermissions != `Accepted Version OK` {
			return fmt.Errorf("❌ %s: task cannot be deposited: Permissions not `Accepted Version OK`", depositID)
		}
	}
	if ssLink, _ := taskRec.Fields[COL_SCHOLINK].(string); ssLink != "" {
		return fmt.Errorf("❌ %s: already deposited: %s", depositID, ssLink)
	}

	// depositor
	depositor, _ := taskRec.Fields[COL_USER].(string)
	depositor = strings.ToLower(depositor)
	if depositor == "" {
		return fmt.Errorf("❌ %s: task cannot be deposited: missing depositor", depositID)
	}

	// File
	if depositFlags.filePath == "" {
		docpath, _ := aiRec.Fields["POST_FILE_1_DOC"].(string)
		if docpath == "" {
			return fmt.Errorf("❌ %s: task cannot be deposited: missing POST_FILE_1_DOC in Activity Insight Record", depositID)
		}
		fileName := filepath.Base(docpath)
		filePath, err := findFile(oats.ArticlePath, fileName)
		if err != nil {
			// try name, replacing white space with "+"
			filePath, err = findFile(oats.ArticlePath, strings.ReplaceAll(fileName, " ", "+"))
			if err != nil {
				return fmt.Errorf("❌ %s: task cannot be deposited: %s: %w", depositID, fileName, err)
			}
		}
		depositFlags.filePath = filePath
	} else {
		inf, err := os.Stat(depositFlags.filePath)
		if err != nil {
			return fmt.Errorf("❌ %s: task cannot be deposited: %w", depositID, err)
		}
		if !inf.Mode().IsRegular() {
			return fmt.Errorf("❌ %s: task cannot be deposited: %s is not a regular file", depositID, depositFlags.filePath)
		}
	}

	//build deposit metadata
	meta := &scholargo.WorkMeta{
		WorkType:   "article",
		Visibility: "open",
	}

	// value from task record
	meta.Title, _ = taskRec.Fields[COL_TITLE].(string)
	meta.Description, _ = taskRec.Fields[COL_ABSTRACT].(string)
	meta.PublishedDate, _ = taskRec.Fields[COL_PUBDATE].(string)
	meta.Embargo, _ = taskRec.Fields[COL_EMBARGO].(string)
	meta.PublisherStatement, _ = taskRec.Fields[COL_STMNT].(string)
	airLicense, _ := taskRec.Fields[COL_LICENSE].(string)
	meta.Rights = convertLicense(airLicense)

	// get doi - try Airtable and RMD
	doi, _ := taskRec.Fields[COL_DOI].(string)
	doi = cleanDOI(doi)
	if doi == "" {
		rmdPubs, err = rmdbCli.PublicationsAI(depositID)
		if err != nil {
			return fmt.Errorf("❌ %s: task cannot be deposited: %w", depositID, err)
		}
		if doi = findPubDOI(rmdPubs); doi != "" {
			doi = cleanDOI(doi)
		}
	}

	if doi != "" {
		// Additional check if we have a DOI
		for d, recs := range scholDOIs {
			if strings.EqualFold("doi:"+doi, d) && len(recs) > 0 {
				return fmt.Errorf("❌ %s: already deposited: %s (%s)", depositID, doi, recs[0])
			}
		}
		// use CrossRef metadata if available
		citation, err := crossref.GetCitation(doi)
		if err != nil {
			return fmt.Errorf("❌ %s: task cannot be deposited: %w", depositID, err)
		}
		if meta.PublishedDate == "" && len(citation.Issued.DateParts) > 0 {
			meta.PublishedDate, _ = convertDate(citation.Issued.DateParts[0])
		}
		meta.Identifier = []string{doi}
		if meta.Description == "" {
			meta.Description = citation.Abstract
		}
		meta.Creators = convertCrossRefAuthors(citation.Author)
		if citation.Publisher != "" {
			meta.Publisher = []string{citation.Publisher}
		}
		meta.Source = citation.ContainerTitle
	}

	// if we have any missing values, try RMD as a last resort
	if meta.Description == "" || meta.PublishedDate == "" || len(meta.Creators) == 0 {
		if rmdPubs == nil {
			rmdPubs, err = rmdbCli.PublicationsAI(depositID)
			if err != nil {
				return fmt.Errorf("❌ %s: failed to connect to rmdb: %w", depositID, err)
			}
		}
		for _, p := range rmdPubs {
			if strings.HasSuffix(strings.ToLower(p.Attributes.DOI), strings.ToLower(doi)) {
				if meta.Description == "" {
					meta.Description = p.Attributes.Abstract
				}
				if meta.PublishedDate == "" {
					meta.PublishedDate = p.Attributes.PublishedOn
				}
				if len(meta.Creators) == 0 {
					meta.Creators = convertRMDAuthors(p.Attributes.Contributors)
				}
				if len(meta.Source) == 0 {
					meta.Source = []string{p.Attributes.JournalTitle}
				}
				if len(meta.Publisher) == 0 {
					meta.Publisher = []string{p.Attributes.Publisher}
				}
				break
			}
		}
	}

	// check all required values are present
	if meta.Title == "" {
		return fmt.Errorf("❌ %s: task cannot be deposited: not title for %s. Try setting in Airtable", depositID, doi)
	}
	if meta.Description == "" {
		return fmt.Errorf("❌ %s: task cannot be deposited: no abstract for %s. Try setting in Airtable", depositID, doi)
	}
	if meta.PublishedDate == "" {
		return fmt.Errorf("❌ %s: task cannot be deposited: no publication date for %s. Try setting in Airtable.", depositID, doi)
	}
	if meta.Rights == "" {
		return fmt.Errorf("❌ %s: task cannot be deposited: unknown license %s", depositID, airLicense)
	}
	if len(meta.Creators) == 0 {
		return fmt.Errorf("❌ %s: task cannot be deposited: missing creators", depositID)
	}

	// do deposit
	resp, err := schol.Deposit(meta, depositor, depositFlags.filePath)
	if err != nil {
		log.Println("------ JSON Dump -----------")
		defer log.Println("---------------------")
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent(``, `  `)
		enc.Encode(meta)
		return fmt.Errorf("❌ %s: deposit failed: %w", depositID, err)
	}
	scholLink := scholURL + resp.URL
	log.Printf("✅ %s: deposited file=%s, doi=%s\n", depositID, depositFlags.filePath, doi)
	if err != nil {
		return fmt.Errorf("❌ %s: failed to update Airtable with deposit information: %w", depositID, err)
	}
	var rmdUpdated bool
	if !depositFlags.skipRMD {
		//update RMDB with scholarsphere links
		err = rmdbCli.UpdateScholarSphereLink(depositID, scholLink)
		if err != nil {
			log.Printf("❌ %s: failed to update RMDB: %s", depositID, err)
		} else {
			rmdUpdated = true
			log.Printf("✅ %s: RMD updated \n", depositID)
		}
	} else {
		log.Println("skipped RMD update")
	}
	updates := map[string]interface{}{
		"Status":             "Deposited",
		"ScholarSphere_Link": scholLink,
		"RMD_Updated":        rmdUpdated,
	}
	_, err = taskRec.UpdateRecordPartial(updates)
	return err
}

func findFile(base string, name string) (string, error) {
	fsys := os.DirFS(base)
	matches, err := fs.Glob(fsys, "*/"+name)
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", fs.ErrNotExist
	}
	sort.Sort(sort.Reverse(sort.StringSlice(matches)))
	if len(matches) > 1 {
		log.Printf("using file: %s from: %s", matches[0], strings.Join(matches, ", "))
	}
	return filepath.Join(base, matches[0]), nil
}

// convert oab license string to scholarsphere license string
func convertLicense(license string) string {
	switch license {
	case "":
		return "https://rightsstatements.org/page/InC/1.0/"
	case "other-closed":
		return "https://rightsstatements.org/page/InC/1.0/"
	case "other (non-commercial)":
		return "https://rightsstatements.org/page/InC/1.0/"
	case "cc-by":
		return "https://creativecommons.org/licenses/by/4.0/"
	case "cc-by-nc":
		return "https://creativecommons.org/licenses/by-nc/4.0/"
	case "cc-by-nc-sa":
		return "https://creativecommons.org/licenses/by-nc-sa/4.0/"
	case "cc-by-nc-nd":
		return "https://creativecommons.org/licenses/by-nc-nd/4.0/"
	case "cc0":
		return "http://creativecommons.org/publicdomain/zero/1.0/"
	}
	return ""
}

// convert slice of crossRef authors to slice of ScholarSphere Creators
func convertCrossRefAuthors(auths []crossref.Author) []scholargo.Creator {
	var ret []scholargo.Creator
	for _, a := range auths {
		if a.Family == "" && a.Name == "" {
			continue
		}
		auth := scholargo.Creator{Orcid: a.ORCID}
		if a.Name == "" {
			auth.Name = a.Given + " " + a.Family
		} else {
			auth.Name = a.Name
		}
		ret = append(ret, auth)
	}
	return ret
}

func convertRMDAuthors(auths []rmdb.Contributor) []scholargo.Creator {
	var ret []scholargo.Creator
	for _, a := range auths {
		ret = append(ret, scholargo.Creator{
			Name:  fmt.Sprintf("%s %s", a.FirstName, a.LastName),
			PSUID: a.PSUID,
		})
	}
	return ret
}

// convert [y,m,d] to "y-m-d"
func convertDate(issue [3]int) (string, error) {
	if issue[0] == 0 {
		return "", errors.New("empty date")
	}
	date := fmt.Sprintf("%d", issue[0])

	if issue[1] != 0 {
		date += fmt.Sprintf("-%02d", issue[1])
		if issue[2] != 0 {
			date += fmt.Sprintf("-%02d", issue[2])
		}
	}
	return date, nil
}
