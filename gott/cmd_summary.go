package gott

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/cheynewallace/tabby"
	"github.com/spf13/cobra"
)

var summaryCmd = &cobra.Command{
	Use:       "summary",
	Short:     "Print tracking summary for a given timespan",
	ValidArgs: Keys,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return nil
		}
		if len(args) > 1 {
			return fmt.Errorf(
				"args should only be one. Choose one of the keys %s, %s, %s, %s or %s or provide in the format YYYY-MM-DD",
				KeyToday, KeyYesterday, KeyWeek, KeyMonth, KeyAll,
			)
		}
		for _, key := range Keys {
			// key found. so it's valid
			if containsString(args, key) {
				return nil
			}
		}
		if _, err := time.Parse(dateFormat, args[0]); err != nil {
			return fmt.Errorf(
				"invalid date format. Choose one of the keys %s, %s, %s, %s or %s or provide in the format YYYY-MM-DD",
				KeyToday, KeyYesterday, KeyMonth, KeyWeek, KeyAll,
			)

		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {

		writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		t := tabby.NewCustom(writer)

		t.AddHeader("CWEEK", "DAY", "BEGIN", "END", "DURATION", "PROJECT", "TAG", "ANNOTATION")

		weekGroup := 0
		weekText := ""
		dayGroup := ""
		dayText := ""
		var dayDurationSum time.Duration
		var weekDurationSum time.Duration
		if len(args) == 0 {
			args = []string{KeyToday}
		}
		intervals, filterError := database.Filter(args)
		if filterError != nil {
			fmt.Fprintf(os.Stderr, "ERROR: invalid filter: %s", filterError.Error())
			os.Exit(1)
		}
		for _, interval := range intervals {

			endText := "tracking..."
			if !interval.End.IsZero() {
				endText = interval.End.Format(timeFormat)
			}

			if d := interval.Begin.Format(dateFormatShort); d != dayGroup {
				dayGroup = d
				dayText = d
				DaySumLine(t, weekDurationSum)
			} else {
				dayText = ""
			}

			if _, w := interval.Begin.ISOWeek(); w != weekGroup {
				weekGroup = w
				weekText = fmt.Sprint(w)
				WeekSumLine(t, dayDurationSum)
			} else {
				weekText = ""
			}

			weekDurationSum += interval.GetDuration()
			dayDurationSum += interval.GetDuration()

			t.AddLine(
				weekText,
				dayText,
				interval.Begin.Format(timeFormat),
				endText,
				fmtDuration(interval.GetDuration()),
				interval.Project,
				strings.Join(interval.Tags, ", "),
				interval.Annotation,
			)
		}
		DaySumLine(t, dayDurationSum)
		WeekSumLine(t, dayDurationSum)

		t.Print()
	},
}

func init() {
	rootCmd.AddCommand(summaryCmd)
}
