package gott

import (
	"fmt"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop currently running tracking",
	Run: func(cmd *cobra.Command, args []string) {
		current, found := database.GetCurrent()
		database.Stop()
		if !found {
			fmt.Println("<< no tracking in progress >>")
		} else {
			PrintStatus(current)
		}
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
