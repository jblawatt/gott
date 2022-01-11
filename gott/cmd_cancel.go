package gott

import (
	"fmt"

	"github.com/spf13/cobra"
)

var cancelCmd = &cobra.Command{
	Use:   "cancel",
	Short: "Cancel currently running tracking",
	Run: func(cmd *cobra.Command, args []string) {
		if _, found := database.GetCurrent(); found {
			database.Cancel()
		} else {
			fmt.Println("no tracking in progress")
		}
	},
}

func init() {
	rootCmd.AddCommand(cancelCmd)
}
