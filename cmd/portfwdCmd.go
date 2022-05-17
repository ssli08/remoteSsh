package cmd

import (
	"fmt"
	"log"
	"net"
	"sshtunnel/modules"
	"sshtunnel/modules/portforward"

	"github.com/spf13/cobra"
)

var (
	localAddr, localPort   string
	sshu, sshh, sshp       string
	targetHost, targetPort string
)
var fwdCmd = &cobra.Command{
	Use:   "pwd",
	Short: "port forword",
	Long:  fmt.Sprintf("Example Usage:\n\t%s\n\n", "rssh pwd -P 3306 -s remoteHost"),
	Run: func(cmd *cobra.Command, args []string) {
		if sshh == "" {
			cmd.Help()
			return
		}
		sshpass, err := modules.GetInputPassword()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(sshu, sshp)
		portforward.PortForward(sshu, sshpass, sshp, sshh, net.JoinHostPort(localAddr, localPort), targetHost, targetPort)

	},
}

func init() {
	rootCmd.AddCommand(fwdCmd)
	// fwdCmd.DisableFlagParsing = true
	// local listen socket
	fwdCmd.Flags().StringVarP(&localAddr, "localAddr", "l", "127.0.0.1", "local listen address")
	fwdCmd.Flags().StringVarP(&localPort, "localPort", "p", "9000", "local listen port")

	// target socket info
	// target socket info
	fwdCmd.Flags().StringVarP(&targetHost, "tgh", "t", "127.0.0.1", "target host ip")
	fwdCmd.Flags().StringVarP(&targetPort, "tgp", "P", "26222", "target host port")

	// remote host info
	fwdCmd.Flags().StringVarP(&sshh, "sshHost", "s", "", "remote ssh server ip")
	// fwdCmd.Flags().StringVar(&sshp, "sp", "", "remote ssh password")
	fwdCmd.Flags().StringVar(&sshu, "su", "ec2-user", "remote ssh user name")
	fwdCmd.Flags().StringVar(&sshp, "sshPort", "26222", "remote ssh port")

}
