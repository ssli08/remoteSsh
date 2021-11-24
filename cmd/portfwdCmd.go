package cmd

import (
	"log"
	"net"
	"os"
	"sshtunnel/cipherText"
	"sshtunnel/database"
	"sshtunnel/modules"
	"sshtunnel/modules/portforward"

	"github.com/spf13/cobra"
)

var (
	localAddr, localPort string
	// jumperHost, jumpPort, jumpUser, jumpPass string
)
var fwdCmd = cobra.Command{
	Use:   "pfd",
	Short: "port forword",
	Run: func(cmd *cobra.Command, args []string) {
		if rmtHost == "" {
			cmd.Help()
			return
		}
		if s, err := os.Stat(database.DBConFile); !os.IsNotExist(err) && s.Size() != 0 {
			db, err := database.GetDBConnInfo(database.DatabaseName)
			if err != nil {
				log.Fatal(err)
			}
			defer db.Close()
			res := modules.ChooseJumperHost(db)
			pass, err := cipherText.DecryptData(res.JmpPass, modules.Passcode)
			if err != nil {
				log.Fatal(err)
			}
			portforward.PortForward(res.JmpUser, string(pass), res.JmpPort, res.JmpHost, net.JoinHostPort(localAddr, localPort), rmtHost, rmtPort)
		} else {
			log.Fatalf("%s not exist or not readable", database.DBConFile)
		}

	},
}

func init() {
	rootCmd.AddCommand(&fwdCmd)
	fwdCmd.DisableFlagParsing = true
	// local listen socket
	fwdCmd.Flags().StringVarP(&localAddr, "localAddr", "l", "127.0.0.1", "local listen address")
	fwdCmd.Flags().StringVarP(&localPort, "localPort", "p", "9000", "local listen port")

	// remote host info
	fwdCmd.Flags().StringVarP(&rmtHost, "rmtHost", "r", "", "remote host ip")
	fwdCmd.Flags().StringVar(&rmtUser, "rmtUser", "ec2-user", "remote user name")
	fwdCmd.Flags().StringVar(&rmtPort, "rmtPort", "26222", "remote ssh port")

}
