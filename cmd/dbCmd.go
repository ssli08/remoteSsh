package cmd

import (
	"log"
	"sshtunnel/database"

	"github.com/spf13/cobra"
)

var (
	// dbType        string
	sqlite3DBFile string

	mysqlHost, mysqlPort, mysqlUser, mysqlPass string
)

var dbChoiceCmd = &cobra.Command{
	Use:   "initdb",
	Short: "initial db used to store instance info, currently support sqlite3 and mysql",
	Run: func(cmd *cobra.Command, args []string) {
		/*
			switch dbType {
			case "sqlite3":
				// initiate sqlite3 with customizate sql
				log.Println("initialize sqlite3...")
				if err := database.SQLite3Init(sqlite3DBFile); err != nil {
					log.Fatal(err)
				}
				log.Println("initialize SQLite3 DB successfully.")
			case "mysql":
				log.Println("initialize mysql...")
				// initiate mysql with customizate sql
				err := database.MySQLConnInit(mysqlHost, mysqlPort, mysqlUser, mysqlPass)
				if err != nil {
					log.Fatal(err)
				}
				log.Println("initialize MySQL DB successfully.")

			default:
				fmt.Println("no db choose, exit")
				cmd.Help()
				return
			}
		*/
		cmd.Help()
	},
}
var mySQLCmd = &cobra.Command{
	Use:   "mysql",
	Short: "use mysql and specify arguments",
	Run: func(cmd *cobra.Command, args []string) {
		if mysqlHost == "" {
			cmd.Help()
			return
		}
		log.Println("initialize mysql...")
		// initiate mysql with customizate sql
		err := database.MySQLConnInit(mysqlHost, mysqlPort, mysqlUser, mysqlPass)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("initialize MySQL DB successfully.")
	},
}

var sqliCmd = &cobra.Command{
	Use:   "sqlite3",
	Short: "use sqlite3 and specify sqlite3 db file",
	Run: func(cmd *cobra.Command, args []string) {
		if sqlite3DBFile == "" {
			cmd.Help()
			return
		}

		// initiate sqlite3 with customizate sql
		log.Println("initialize sqlite3...")
		if err := database.SQLite3Init(sqlite3DBFile); err != nil {
			log.Fatal(err)
		}
		log.Println("initialize SQLite3 DB successfully.")
	},
}

func init() {
	rootCmd.AddCommand(dbChoiceCmd)
	dbChoiceCmd.AddCommand(mySQLCmd)
	dbChoiceCmd.AddCommand(sqliCmd)

	// dbChoiceCmd.Flags().StringVarP(&dbType, "dbtype", "d", "sqlite3", "choose a db to store instance info, currently support both sqlite3 and mysql")

	mySQLCmd.Flags().StringVarP(&mysqlHost, "dbHost", "s", "127.0.0.1", "mysql server host")
	mySQLCmd.Flags().StringVarP(&mysqlUser, "dbUser", "u", "ssh", "user for connect mysql")
	mySQLCmd.Flags().StringVarP(&mysqlPass, "dbPass", "p", "ssh@gs.com", "password for connect mysql")
	mySQLCmd.Flags().StringVarP(&mysqlPort, "dbPort", "t", "3306", "mysql listen port")

	sqliCmd.Flags().StringVarP(&sqlite3DBFile, "sqliteDBFile", "s", "/opt/sqlite3/logserver.db", "sqlite3 db file")

	/*
		flag.StringVar(&sqlite3DBFile, "sf", "/opt/sqlite3/logserver.db", "sqlite3 db file path")
		flag.StringVar(&mysqlHost, "mh", "127.0.0.1", "mysql server host")
		flag.StringVar(&mysqlPort, "mport", "3306", "mysql server port")
		flag.StringVar(&mysqlUser, "mu", "ssh", "mysql user for connection")
		flag.StringVar(&mysqlPass, "mp", "ssh@gs.com", "mysql password for connection") */
}
