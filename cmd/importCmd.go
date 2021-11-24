package cmd

import (
	"log"
	"os"
	"path/filepath"
	"sshtunnel/database"
	"sshtunnel/modules"

	"github.com/spf13/cobra"
)

var (
	// api          bool
	instanceFile string

	region           string
	keyFile, sshUser string

	jumpHost, jumpUser, jumpPass, jumpPort string

	expTableName string // used to export db table record to csv file
)
var importInstancesCmd = &cobra.Command{
	Use:   "import",
	Short: "import/update instances list in DB from api or file",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}
var importInstanceFromAPICmd = &cobra.Command{
	Use:   "api",
	Short: "import instances to DB from Service(aws/vps) API",
	Run: func(cmd *cobra.Command, args []string) {
		if project == "" || region == "" {
			cmd.Help()
			return
		}

		if s, err := os.Stat(database.DBConFile); !os.IsNotExist(err) && s.Size() != 0 {
			db, err := database.GetDBConnInfo(database.DatabaseName)
			if err != nil {
				log.Fatal(err)
			}
			defer db.Close()

			modules.ImportAWSInstancesToDB(db, project, region)
			if project == "gdms" {
				modules.ImportVPSInstancesToDB(db)
			}
		} else {
			log.Fatalf("%s not exist or not readable", database.DBConFile)
		}

	},
}

var importInstanceFromFileCmd = &cobra.Command{
	Use:   "file",
	Short: "import instances to DB from file",
	Run: func(cmd *cobra.Command, args []string) {
		if instanceFile == "" {
			cmd.Help()
			return
		}
		switch filepath.Ext(instanceFile) {
		case ".xls":
			if err := modules.ReadXLS(instanceFile, "utf8"); err != nil {
				log.Fatal(err)
			}
		case ".xlsx":
			if err := modules.ReadXLSX(instanceFile); err != nil {
				log.Fatal(err)
			}
		case ".csv":
			if _, err := modules.ReadCSV(instanceFile); err != nil {
				log.Fatal(err)
			}
		default:
			log.Fatalf("can NOT recognise %s with Extension (%s)", instanceFile, filepath.Ext(instanceFile))
		}
	},
}
var importSSHKeysCmd = &cobra.Command{
	Use:   "sshkey",
	Short: "import sshkey to DB from keyfile",
	Run: func(cmd *cobra.Command, args []string) {
		if keyFile == "" || sshUser == "" {
			cmd.Help()
			return
		}
		if s, err := os.Stat(database.DBConFile); !os.IsNotExist(err) && s.Size() != 0 {
			db, err := database.GetDBConnInfo(database.DatabaseName)
			if err != nil {
				log.Fatal(err)
			}
			defer db.Close()
			modules.ImportSSHKey(db, keyFile, sshUser, modules.Passcode)
		} else {
			log.Fatalf("%s not exist or not readable", database.DBConFile)
		}

	},
}
var importJumpHostCmd = &cobra.Command{
	Use:   "jph",
	Short: "import jump host info to DB",
	Run: func(cmd *cobra.Command, args []string) {
		if jumpHost == "" {
			cmd.Help()
			return
		}
		if s, err := os.Stat(database.DBConFile); !os.IsNotExist(err) && s.Size() != 0 {
			db, err := database.GetDBConnInfo(database.DatabaseName)
			if err != nil {
				log.Fatal(err)
			}
			defer db.Close()
			modules.ImportJumperHosts(db, jumpHost, jumpUser, jumpPass, jumpPort, modules.Passcode)
		} else {
			log.Fatalf("%s not exist or not readable", database.DBConFile)
		}
	}}

var exportDBTableToFileCmd = &cobra.Command{
	Use:   "export",
	Short: "export db record to csv format file",
	Run: func(cmd *cobra.Command, args []string) {
		if expTableName == "" {
			cmd.Help()
			return
		}
		if s, err := os.Stat(database.DBConFile); !os.IsNotExist(err) && s.Size() != 0 {
			db, err := database.GetDBConnInfo(database.DatabaseName)
			if err != nil {
				log.Fatal(err)
			}
			defer db.Close()
			database.ExportTableTOCSVFile(db, expTableName)
		}

	},
}

func init() {
	rootCmd.AddCommand(importInstancesCmd)
	rootCmd.AddCommand(exportDBTableToFileCmd)

	importInstancesCmd.AddCommand(importInstanceFromAPICmd)
	importInstancesCmd.AddCommand(importInstanceFromFileCmd)
	importInstancesCmd.AddCommand(importSSHKeysCmd)
	importInstancesCmd.AddCommand(importJumpHostCmd)

	// updateInstanceFromAPI.Flags().BoolVarP(&api, "api", "i", false, "import instance from service api, both aws and vps service")
	importInstanceFromAPICmd.Flags().StringVarP(&project, "project", "p", "gwn", "get project instances(aws account used)")
	importInstanceFromAPICmd.Flags().StringVarP(&region, "region", "r", "us-west-2", "get aws instances with different regions")

	importInstanceFromFileCmd.Flags().StringVarP(&instanceFile, "file", "f", "", "import instance from file")

	// import ssh keys cmd args
	importSSHKeysCmd.Flags().StringVarP(&keyFile, "keyfile", "k", "", "import this keyFile to DB")
	importSSHKeysCmd.Flags().StringVarP(&sshUser, "sshUser", "s", "", "ssh user")
	// importSSHKeys.Flags().StringVarP(&passcode, "passcode", "P", "rmttssh", "password for encypting keyFile content")

	importJumpHostCmd.Flags().StringVar(&jumpHost, "jh", "", "jump host need to import db")
	importJumpHostCmd.Flags().StringVar(&jumpUser, "ju", "ec2-user", "jump host ssh user need to import db")
	importJumpHostCmd.Flags().StringVar(&jumpPass, "jp", "", "jump host ssh password need to import db")
	importJumpHostCmd.Flags().StringVar(&jumpPort, "jport", "26222", "jump host ssh port need to import db")

	exportDBTableToFileCmd.Flags().StringVarP(&expTableName, "exptabname", "t", "", "exported database table name")
}
