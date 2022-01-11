package gott

import "github.com/spf13/cobra"

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start tracking",
	Run: func(cmd *cobra.Command, args []string) {
		interval := NewInterval(args)
		database.Start(interval)
		PrintRunningStatus()
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
