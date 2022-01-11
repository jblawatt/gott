package gott

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var annotateCmd = &cobra.Command{
	Use:   "annotate",
	Short: "Set annotation for currently running tracking",
	Run: func(cmd *cobra.Command, args []string) {
		c, found := database.GetCurrent()
		if !found {
			fmt.Fprintln(os.Stderr, "ERROR: no tracking in process. unpointed annotionation is only valid for running trackings")
			os.Exit(1)
		}
		lexInterval(args, c)
		PrintRunningStatus()
	},
}

func init() {
	rootCmd.AddCommand(annotateCmd)
}
