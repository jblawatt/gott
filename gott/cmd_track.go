package gott

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var trackCmd = &cobra.Command{
	Use:   "track",
	Short: "Add interval for a date/keyword",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return nil
		}
		return fmt.Errorf("must have the format DATE DURATION -- ANNOTATION")
	},
	Run: func(cmd *cobra.Command, args []string) {
		interval := NewInterval(args[2:])
		if err := lexTrack(args[0:2], &interval); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		} else {
			database.Append(interval)
		}
	},
}

func init() {
	rootCmd.AddCommand(trackCmd)
}
