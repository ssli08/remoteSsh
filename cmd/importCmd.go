package cmd

import (
	"log"
	"os"
	"sshtunnel/database"
	"sshtunnel/modules"

	"github.com/spf13/cobra"
)

var (
	// api          bool
	instanceFile string

	project, region   string
	keyFile, passcode string
)
var importInstancesCmd = &cobra.Command{
	Use:   "import",
	Short: "import/update instances list in DB from api or file",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}
var importInstanceFromAPI = &cobra.Command{
	Use:   "api",
	Short: "import instances to DB from Service(aws/vps) API",
	Run: func(cmd *cobra.Command, args []string) {
		if project == "" || region == "" {
			cmd.Help()
			return
		}

		if s, err := os.Stat(database.DBConFile); !os.IsNotExist(err) && s.Size() != 0 {
			db, err := database.GetDBConnInfo("sshServers")
			if err != nil {
				log.Fatal(err)
			}
			defer db.Close()

			modules.ImportAWSInstancesToDB(db, project, region)
			modules.ImportVPSInstancesToDB(db)
		} else {
			log.Fatalf("%s not exist or not readable", database.DBConFile)
		}

	},
}

var importInstanceFromFile = &cobra.Command{
	Use:   "file",
	Short: "import instances to DB from file",
	Run: func(cmd *cobra.Command, args []string) {
		if instanceFile == "" {
			cmd.Help()
			return
		}
	},
}
var importSSHKeys = &cobra.Command{
	Use:   "sshkey",
	Short: "import sshkey to DB from keyfile",
	Run: func(cmd *cobra.Command, args []string) {
		if keyFile == "" {
			cmd.Help()
			return
		}
		if s, err := os.Stat(database.DBConFile); !os.IsNotExist(err) && s.Size() != 0 {
			db, err := database.GetDBConnInfo("sshServers")
			if err != nil {
				log.Fatal(err)
			}
			defer db.Close()
			modules.ImportSSHKey(db, keyFile, passcode)
		} else {
			log.Fatalf("%s not exist or not readable", database.DBConFile)
		}

	},
}

func init() {
	rootCmd.AddCommand(importInstancesCmd)
	importInstancesCmd.AddCommand(importInstanceFromAPI)
	importInstancesCmd.AddCommand(importInstanceFromFile)
	importInstancesCmd.AddCommand(importSSHKeys)
	// updateInstanceFromAPI.Flags().BoolVarP(&api, "api", "i", false, "import instance from service api, both aws and vps service")
	importInstanceFromAPI.Flags().StringVarP(&project, "project", "p", "gwn", "get project instances(aws account used)")
	importInstanceFromAPI.Flags().StringVarP(&region, "region", "r", "us-west-2", "get aws instances with different regions")

	importInstanceFromFile.Flags().StringVarP(&instanceFile, "file", "f", "", "import instance from file")

	// import ssh keys cmd args
	importSSHKeys.Flags().StringVarP(&keyFile, "keyfile", "k", "", "import this keyFile to DB")
	importSSHKeys.Flags().StringVarP(&passcode, "passcode", "P", "rmttssh", "password for encypting keyFile content")
}
