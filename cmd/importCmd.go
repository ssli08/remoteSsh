package cmd

import (
	"fmt"
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

	region                    string
	keyFile, sshUser, sshPort string // for sshkey importing
	sshPassword               string // for ssh password importing
	role                      string // jumperHost or realBackendHost

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

		if db, err := database.GetDBConnInfo(database.DatabaseName); err == nil {
			defer db.Close()

			// truncate currently exist instance table
			// if err := database.DBExecute(db, fmt.Sprintf("truncate table %s", database.InstanceTableName)); err != nil {
			// 	log.Fatal(err)
			// }

			// modules.ImportAWSInstancesToDB(db, project, region)
			modules.UpdateInstanceListsInDB(db, project, region)
			if project == "gdms" {
				modules.ImportVPSInstancesToDB(db)
			}
		} else {
			log.Fatal(err)
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
		if db, err := database.GetDBConnInfo(database.DatabaseName); err == nil {
			defer db.Close()

			// truncate currently exist instance table
			if err := database.DBExecute(db, fmt.Sprintf("truncate table %s", database.InstanceTableName)); err != nil {
				log.Fatal(err)
			}

			switch filepath.Ext(instanceFile) {
			case ".xls":
				if err := modules.ReadXLS(db, instanceFile, "utf8"); err != nil {
					log.Fatal(err)
				}
			case ".xlsx":
				if err := modules.ReadXLSX(db, instanceFile); err != nil {
					log.Fatal(err)
				}
			case ".csv":
				if _, err := modules.ReadCSV(db, instanceFile); err != nil {
					log.Fatal(err)
				}
			default:
				log.Fatalf("can NOT recognise %s with Extension (%s)", instanceFile, filepath.Ext(instanceFile))
			}
		} else {
			log.Fatal(err)
		}
	},
}
var importSSHKeysCmd = &cobra.Command{
	Use:   "sshkey",
	Short: "import sshkey to DB from keyfile",
	Run: func(cmd *cobra.Command, args []string) {
		if keyFile == "" && sshPassword == "" {
			fmt.Println("keyFile or sshPassword need to be provided!!")
			cmd.Help()
			return
		}
		// fmt.Println("k: ", keyFile, "p:", sshPassword)

		if db, err := database.GetDBConnInfo(database.DatabaseName); err == nil {
			defer db.Close()
			modules.ImportSSHAuthentication(db, keyFile, sshUser, sshPort, sshPassword, role, modules.Passcode)
		} else {
			log.Fatal(err)
		}
	},
}

/* var importSSHPasswdCmd = &cobra.Command{
	Use:   "sshpass",
	Short: "import ssh password to DB",
	Run: func(cmd *cobra.Command, args []string) {
		if sshPassword == "" {
			cmd.Help()
			return
		}
		if s, err := os.Stat(database.DBConFile); !os.IsNotExist(err) && s.Size() != 0 {
			db, err := database.GetDBConnInfo(database.DatabaseName)
			if err != nil {
				log.Fatal(err)
			}
			defer db.Close()
			modules.ImportSSHPassword(db, project, sshPassword, sshUser, sshPort, modules.Passcode)
		} else {
			log.Fatalf("%s not exist or not readable", database.DBConFile)
		}
	},
} */

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

		if db, err := database.GetDBConnInfo(database.DatabaseName); err == nil {
			defer db.Close()
			database.ExportTableTOCSVFile(db, expTableName)
		} else {
			log.Fatal(err)
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
	// importInstancesCmd.AddCommand(importSSHPasswdCmd)

	// updateInstanceFromAPI.Flags().BoolVarP(&api, "api", "i", false, "import instance from service api, both aws and vps service")
	importInstanceFromAPICmd.Flags().StringVarP(&project, "project", "p", "", "get project instances(aws account used)")
	importInstanceFromAPICmd.Flags().StringVarP(&region, "region", "r", "us-west-2", "get aws instances with different regions")

	importInstanceFromFileCmd.Flags().StringVarP(&instanceFile, "file", "f", "", "import instance from file")

	// import ssh keys cmd args
	importSSHKeysCmd.Flags().StringVarP(&keyFile, "keyfile", "k", "", "import this keyFile to DB")
	importSSHKeysCmd.Flags().StringVarP(&sshUser, "sshUser", "s", "", "import ssh user to DB")
	importSSHKeysCmd.Flags().StringVarP(&sshPort, "sshPort", "p", "26222", "import ssh port to DB")
	importSSHKeysCmd.Flags().StringVarP(&sshPassword, "sshpasswd", "P", "", "import this ssh password to DB")
	// importSSHKeysCmd.Flags().StringVarP(&role, "role", "r", "rs", "instance role [jumperServerrealServer] in DB")

	// import ssh password cmd args
	// importSSHPasswdCmd.Flags().StringVarP(&sshPassword, "sshpasswd", "p", "", "import this ssh password to DB")
	// importSSHPasswdCmd.Flags().StringVarP(&sshUser, "su", "u", "", "import ssh user to DB")
	// importSSHPasswdCmd.Flags().StringVarP(&project, "proj", "j", "", "import consistent ssh password to DB for same project")

	importJumpHostCmd.Flags().StringVar(&jumpHost, "jh", "", "jump host need to import db")
	importJumpHostCmd.Flags().StringVar(&jumpUser, "ju", "ec2-user", "jump host ssh user need to import db")
	importJumpHostCmd.Flags().StringVar(&jumpPass, "jp", "", "jump host ssh password need to import db")
	importJumpHostCmd.Flags().StringVar(&jumpPort, "jport", "26222", "jump host ssh port need to import db")

	exportDBTableToFileCmd.Flags().StringVarP(&expTableName, "exptabname", "t", "", "exported database table name")
}
