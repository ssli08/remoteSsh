package cmd

import "github.com/spf13/cobra"

var rootCmd = cobra.Command{
	Use:   "remoteSsh",
	Short: "ssh remote",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}
func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}
