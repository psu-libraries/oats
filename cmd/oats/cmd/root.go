/*
Copyright © 2022

*/
package cmd

import (
	"log"
	"os"

	"github.com/muesli/coral"
	"github.com/psu-libraries/oats/cmd/oats/base"
)

var oats *base.Oats

const (
	COL_ID          = "ID"
	COL_AI_ID       = "AI_ID"
	COL_VERSION     = "Article_Version"
	COL_STATUS      = "Status"
	COL_DOI         = "DOI"
	COL_DOI_CONF    = "DOI_Confirmed"
	COL_OA_STATUS   = "OA_status"
	COL_OA_LINK     = "OA_Link"
	COL_PERM        = "Permissions"
	COL_PERM_SRC    = "Permissions_Source"
	COL_LICENSE     = "License"
	COL_EMBARGO     = "Embargo_End"
	COL_STMNT       = "Set_Statement"
	COL_TITLE       = "Title"
	COL_ABSTRACT    = "Abstract"
	COL_PUBDATE     = "Publication_Date"
	COL_JOURNAL     = "Journal_Name"
	COL_USER        = "User"
	COL_SCHOLINK    = "ScholarSphere_Link"
	COL_RMD_UPDATED = "RMD_Updated"
)

var rootFlags struct {
	configFile string
	production bool
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &coral.Command{
	Use:          "oats",
	Short:        "OA Tools: a collection of programs for managing the OA workflow",
	Long:         ``,
	SilenceUsage: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	coral.OnInitialize(initConfig)
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.PersistentFlags().StringVarP(&rootFlags.configFile, "config", "c", "config.yml", "config file")
	rootCmd.PersistentFlags().BoolVarP(&rootFlags.production, "production", "p", false, "run in production mode")
}

func initConfig() {
	var err error
	oats, err = base.NewOats(rootFlags.configFile)
	if err != nil {
		log.Fatal(err)
	}
	oats.Production = rootFlags.production
}
