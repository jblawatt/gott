package gott

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var continueCmd = &cobra.Command{
	Use:   "continue",
	Short: "Continue last running tracking",
	Run: func(cmd *cobra.Command, args []string) {
		if _, found := database.GetCurrent(); found {
			fmt.Fprintln(os.Stderr, "ERROR: there is a tracking in progress. Nothing to continue.")
			os.Exit(1)
		}
		if latest, err := database.Latest(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
			os.Exit(1)
		} else {
			database.Start(NewInterval(strings.Split(latest.Raw, " ")))
			PrintRunningStatus()
		}
	},
}

func init() {
	rootCmd.AddCommand(continueCmd)
}
