package cmd

import (
	"sshtunnel/modules"

	"github.com/spf13/cobra"
)

var (
	project                            string
	rmtHost, rmtUser, rmtPass, rmtPort string

	print bool
	fcopy bool //file copy
)

var rootCmd = cobra.Command{
	Use:   "rssh",
	Short: "ssh remote server through jumper host",
	Args:  cobra.ArbitraryArgs, // use `Args` to parse non-flag args
	Run: func(cmd *cobra.Command, args []string) {
		// if project == "" || rmtHost == "" {
		// 	cmd.Help()
		// 	return
		// }
		// fmt.Println("args:", args)
		modules.InitSession(print, fcopy, project, rmtHost, rmtPort, rmtUser, rmtPass, args)
	},
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
	// rootCmd.Execute()
}
func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	rootCmd.Flags().StringVarP(&project, "project", "p", "", "project server to connect and show server list, options: gwn|gdms|ipvt")
	rootCmd.Flags().BoolVarP(&print, "print", "s", false, "show instances list")
	rootCmd.Flags().BoolVarP(&fcopy, "fcopy", "f", false, "copy files to remote host")
	// rootCmd.Flags().StringVarP(&jmpHost, "jph", "j", "52.83.235.118", "jumper host")

	// rootCmd.Flags().StringVar(&jmpUser, "jmpUser", "ec2-user", "jump host ssh user")
	// rootCmd.Flags().StringVar(&jmpPort, "jmpPort", "26222", "jump host ssh user")

	rootCmd.Flags().StringVarP(&rmtHost, "rh", "r", "", "remote host to be connected")
	rootCmd.Flags().StringVar(&rmtUser, "ru", "", "remote host ssh user")
	rootCmd.Flags().StringVar(&rmtPass, "rp", "", "remote host ssh password")
	rootCmd.Flags().StringVar(&rmtPort, "rport", "26222", "remote host ssh port")

}
