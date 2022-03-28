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
	reset    bool
)

var netTestCmd = cobra.Command{
	Use:   "net",
	Short: "network latency test",
	// Long:  "a counter that count duration between sending syn to server and receiving rst/sync+ack, icmp also support",
	Long: "a counter getting RTT duration between local and remote host",
	Run: func(cmd *cobra.Command, args []string) {

		if port == 0 {
			if reset {
				db, err := database.GetDBConnInfo(database.DatabaseName)
				if err != nil {
					log.Fatal(err)
				}
				defer db.Close()
				networkdetect.ResetRTT(db)
				return
			}
			cmd.Help()
			return
		}

		if rmtHost == "" {
			db, err := database.GetDBConnInfo(database.DatabaseName)
			if err != nil {
				log.Fatal(err)
			}
			defer db.Close()
			networkdetect.UpdateJumperHostLatency(db, port)
			return
		} else {
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
		}
	},
}

func init() {
	rootCmd.AddCommand(&netTestCmd)
	netTestCmd.Flags().StringVarP(&rmtHost, "rmtHost", "r", "", "remote host ip  to test latency")
	netTestCmd.Flags().StringVarP(&testType, "type", "t", "tcp", "tcp or icmp latency test")
	netTestCmd.Flags().Uint16Var(&port, "port", 0, "remote host port")

	netTestCmd.Flags().BoolVar(&reset, "reset", false, "reset RTT time to 0 for all jump hosts")
}
