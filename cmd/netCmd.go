package cmd

import (
	"fmt"
	"log"
	"sshtunnel/database"
	"sshtunnel/modules/networkdetect"

	"github.com/spf13/cobra"
)

var (
	port     uint16
	testType string
)

var netTestCmd = cobra.Command{
	Use:   "net",
	Short: "network latency test",
	Long:  "a counter that count duration between sending syn to server and receiving rst/sync+ack, icmp also support",
	Run: func(cmd *cobra.Command, args []string) {
		if rmtHost == "" || port == 0 {
			cmd.Help()
			return
		}

		if db, err := database.GetDBConnInfo(database.DatabaseName); err != nil {
			log.Fatal(err)
		} else {
			networkdetect.UpdateJumperHostLatency(db, port)
		}

		if testType == "icmp" {
			a, err := networkdetect.ICMPPingLatency(rmtHost)
			if err != nil {
				fmt.Println("icmp test failed with error ", err)
				return
			}
			fmt.Printf("ICMP Latency for %s is %s\n", rmtHost, a)
			return
		}
		a := networkdetect.LatencyTest(rmtHost, port)
		fmt.Printf("TCP Latency for %s is %s\n", rmtHost, a)

	},
}

func init() {
	rootCmd.AddCommand(&netTestCmd)
	netTestCmd.Flags().StringVarP(&rmtHost, "rmtHost", "r", "", "remote host ip  to test latency")
	netTestCmd.Flags().StringVarP(&testType, "type", "t", "tcp", "tcp or icmp latency test")
	netTestCmd.Flags().Uint16Var(&port, "port", 0, "remote host port")

}
