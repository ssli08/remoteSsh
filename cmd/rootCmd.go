package cmd

import (
	"fmt"
	"sshtunnel/modules"

	"github.com/spf13/cobra"
)

var (
	project, role                      string
	rmtHost, rmtUser, rmtPass, rmtPort string

	print    bool
	fcopy    bool   // file copy
	destPath string // file copy dest path

	command bool // execute command in batach

	// directly bool // directly connect real instance
)

var rootCmd = cobra.Command{
	Use:   "rssh",
	Short: "ssh remote server through jumper host",
	Long:  fmt.Sprintf("Example Usage:\n\t%s\n\t%s\n", "rssh -p proj -r remoteHost ", "rssh -p proj -R role -c cmd "),
	Args:  cobra.ArbitraryArgs, // use `Args` to parse non-flag args
	Run: func(cmd *cobra.Command, args []string) {
		// if project == "" || rmtHost == "" {
		// 	cmd.Help()
		// 	return
		// }
		// fmt.Println("args:", args, len(args))
		modules.InitSession(print, fcopy, command, project, role, destPath, rmtHost, rmtPort, rmtUser, rmtPass, args, args)
	},
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
	// rootCmd.Execute()
}
func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	rootCmd.Flags().StringVarP(&project, "project", "p", "", "project server to connect and show server list, options: gwn|gdms|ipvt")
	rootCmd.Flags().StringVarP(&role, "role", "R", "", "required if you get commands executed in batch, options: web|ssh|turn")

	rootCmd.Flags().BoolVarP(&print, "print", "s", false, "show instances list")
	rootCmd.Flags().BoolVarP(&fcopy, "fcopy", "f", false, "copy files to remote host")
	rootCmd.Flags().StringVarP(&destPath, "dest", "d", "", "copy file to dest path")

	rootCmd.Flags().BoolVarP(&command, "cmd", "c", false, "switch for run cmd in batch")

	// rootCmd.Flags().StringVar(&jmpUser, "jmpUser", "ec2-user", "jump host ssh user")
	// rootCmd.Flags().StringVar(&jmpPort, "jmpPort", "26222", "jump host ssh user")

	rootCmd.Flags().StringVarP(&rmtHost, "rh", "r", "", "remote host to be connected")
	rootCmd.Flags().StringVar(&rmtUser, "ru", "ec2-user", "remote host ssh user")
	rootCmd.Flags().StringVar(&rmtPass, "rp", "", "remote host ssh password")
	rootCmd.Flags().StringVar(&rmtPort, "rport", "26222", "remote host ssh port")

	// directly connect real instance
	// rootCmd.Flags().BoolVarP(&directly, "direct", "d", false, "directly connect instance")

}
