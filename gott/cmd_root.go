package gott

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use: "gott",
	Run: func(cmd *cobra.Command, args []string) {
		PrintRunningStatus()
	},
}
