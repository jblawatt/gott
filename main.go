package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"text/tabwriter"

	// github.com/r3labs/diff/v2

	"github.com/cheynewallace/tabby"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	dateFormat          = "2006-01-02"
	dateFormatShort     = "01-02"
	datetimeFormat      = "2006-01-02 15:04:05"
	datetimeFormatShort = "01-02 15:04"
	timeFormat          = "15:04"
)

// Commands

var rootCmd = &cobra.Command{
	Use: "gott",
	Run: func(cmd *cobra.Command, args []string) {
		PrintRunningStatus()
	},
}

type filterFunc = func(i *Interval) bool

func containsString(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

func createProjectFilter(project string) filterFunc {
	return func(i *Interval) bool {
		return i.Project == project
	}
}

func createTagFilter(tag string) filterFunc {
	return func(i *Interval) bool {
		return containsString(i.Tags, tag)
	}
}

func createDateFilter(t time.Time) filterFunc {
	return func(i *Interval) bool {
		a := i.Begin.Truncate(24 * time.Hour)
		b := t.Truncate(24 * time.Hour)
		return a.Equal(b)
	}
}

func createDateRangeFilter(from, to time.Time) filterFunc {
	from = from.Truncate(24 * time.Hour)
	to = to.Truncate(24*time.Hour).AddDate(0, 0, 1)
	return func(i *Interval) bool {
		return i.Begin.After(from) && i.Begin.Before(to)
	}
}

func applyFilter(i *Interval, flist []filterFunc) bool {
	for _, ffunc := range flist {
		if !(ffunc(i)) {
			return false
		}
	}
	return true
}

const (
	KeyToday     = ":today"
	KeyYesterday = ":yesterday"
	KeyWeek      = ":week"
	KeyMonth     = ":month"
	KeyAll       = ":all"
)

var Keys = []string{KeyToday, KeyYesterday, KeyWeek, KeyMonth, KeyAll}

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
				"Args should only be one. Choose one of the keys %s, %s, %s, %s or %s or provide in the format YYYY-MM-DD",
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
				"Invalid date format. Choose one of the keys %s, %s, %s, %s or %s or provide in the format YYYY-MM-DD",
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

func DaySumLine(t *tabby.Tabby, dur time.Duration) {
	if dur > 0 {
		t.AddLine("", "", "", "day =", fmtDuration(dur), "", "", "")
	}
}

func WeekSumLine(t *tabby.Tabby, dur time.Duration) {
	if dur > 0 {
		t.AddLine("", "", "wk =", "", fmtDuration(dur), "", "", "")
	}
}

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

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start tracking",
	Run: func(cmd *cobra.Command, args []string) {
		interval := NewInterval(args)
		database.Start(interval)
		PrintRunningStatus()
	},
}

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
		if database.Count() > 0 {
		}
	},
}

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

func writeEditFile(f *os.File, intervals []*Interval, filterArgs []string) {

	f.WriteString("# Edit below values to change tracking data\n")
	f.WriteString("# - delete rows to delete \n")
	f.WriteString("# - set time values 00:00 or leave empty to just set duration \n")
	f.WriteString("# - leave duration empty or 0s if you set date \n")
	f.WriteString("\n")

	writer := tabwriter.NewWriter(f, 0, 0, 2, ' ', 0)
	for _, i := range intervals {
		line := fmt.Sprintf(
			"[%s]\t[%s]\t[%s]\t[%s]\t[%s]\t[%s]\n",
			i.ID,
			i.Begin.Format(dateFormat),
			i.Begin.Format(timeFormat),
			i.End.Format(timeFormat),
			i.Duration,
			i.Raw,
		)
		writer.Write([]byte(line))
	}

	writer.Flush()

	f.WriteString("\n\n# NEW ENTRIES HERE #############################################\n")
	f.WriteString("# [ID (empty)] [DATE] [BEGIN] [END] [DURATION] [ANNOTATION]\n")
	f.WriteString(fmt.Sprintf("\n# [] [%s] [] [] [] []\n", time.Now().Format(dateFormat)))

	f.WriteString("\n\n\n\n# meta #########################################################\n")
	f.WriteString(fmt.Sprintf("# ;; filter == %s\n", strings.Join(filterArgs, " ")))

}

func runEditFile(f *os.File) {
	excCmd := exec.Command("nvim", f.Name())
	excCmd.Stdout = os.Stdout
	excCmd.Stderr = os.Stderr
	excCmd.Stdin = os.Stdin
	excCmd.Run()
}

func parseEditLine(t string) (Interval, error) {

	pattern := regexp.MustCompile(`\[[\w\w\-\_0-9 :]*\]`)
	result := pattern.FindAllString(t, -1)
	var cleaned []string
	for _, col := range result {
		cleaned = append(cleaned, strings.Trim(stripBraces(col), " "))
	}
	var id, date, begin, end, duration, annotation string
	unpackSlice(cleaned, &id, &date, &begin, &end, &duration, &annotation)
	annotationSlice := strings.Split(annotation, " ")
	interval := NewInterval(annotationSlice)

	if date == "" {
		return interval, fmt.Errorf("Date is empty but should be filled with format YYYY-MM-DD")
	}

	tdate, tdateErr := time.Parse(dateFormat, date)
	if tdateErr != nil {
		return interval, fmt.Errorf("Error parsing date '%s'. Error: %s", date, tdateErr.Error())
	}

	tbegin, terr := time.Parse(timeFormat, begin)
	if begin != "" && terr != nil {
		return interval, fmt.Errorf("Error parsing begin time '%s': %s", begin, terr.Error())
	}

	tend, tendErr := time.Parse(timeFormat, end)
	if begin != "" && tendErr != nil {
		return interval, fmt.Errorf("Error parsing end time '%s': %s", end, tendErr.Error())
	}

	if tbegin.Format(timeFormat) != "00:00" && tend.Format(timeFormat) != "00:00" {
		interval.Begin = tdate.Add(time.Duration(tbegin.Hour())*time.Hour + time.Duration(tbegin.Minute())*time.Minute)
		interval.End = tdate.Add(time.Duration(tend.Hour())*time.Hour + time.Duration(tend.Minute())*time.Minute)
	} else {
		tdur, tdurErr := time.ParseDuration(duration)
		if tdurErr != nil {
			return interval, fmt.Errorf("Error parsing duration: %s", tdurErr.Error())
		}
		interval.Begin = tdate
		interval.End = tdate
		interval.Duration = tdur
	}
	interval.ID = id
	return interval, nil
}

func parseEditFile(f *os.File) ([]Interval, error) {
	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)
	var result []Interval
	var line = 1
	var pos int64 = 0
	// 	var line int = 1
	for scanner.Scan() {
		t := scanner.Text()

		pos += int64(len(t) + 1)
		// ignore commented line
		if strings.HasPrefix(t, "#") {
			line += 1
			continue
		}
		// ignore empty line
		if len(strings.Trim(t, " ")) == 0 {
			line += 1
			continue
		}
		if i, err := parseEditLine(t); err != nil {
			return []Interval{}, err
		} else {
			line += 1
			result = append(result, i)
		}
	}

	return result, nil
}

var editCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit the intervals in the provided timespan",
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) == 0 {
			args = []string{KeyToday}
		}
		intervals, errFilter := database.Filter(args)
		if errFilter != nil {
			fmt.Fprintf(os.Stderr, "ERROR: invalid filter: %s", errFilter.Error())
			os.Exit(1)
		}

		var beforeIDs []string
		for _, i := range intervals {
			beforeIDs = append(beforeIDs, i.ID)
		}

		f, _ := ioutil.TempFile(os.TempDir(), ".md")
		defer f.Close()
		defer os.Remove(f.Name())

		writeEditFile(f, intervals, args)
		runEditFile(f)

		f.Seek(0, 0)

		// TODO: optimize:
		// - test if content changed
		// - test diff content and not walk through it
		// - do not update every interval
		editIntervals, err := parseEditFile(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: error in parsing file: %s", err.Error())
			os.Exit(1)
		}

		var afterIDs []string

		for _, intv := range editIntervals {
			if intv.ID == "" {
				database.Append(intv)
			} else {
				database.Apply(intv)
				afterIDs = append(afterIDs, intv.ID)
			}
		}

		if len(beforeIDs) != len(afterIDs) {
			for _, beforeID := range beforeIDs {
				if !containsString(afterIDs, beforeID) {
					database.RemoveById(beforeID)
				}
			}
		}

	},
}

func stripBraces(s string) string {
	return s[1 : len(s)-1]
}

func unpackSlice(s []string, vars ...*string) {
	for i, str := range s {
		*vars[i] = str
	}
}

func PrintStatus(interval *Interval) {

	fmt.Printf("tracking %s", interval.Annotation)
	if interval.Project != "" {
		fmt.Printf(" -- proj:%s", interval.Project)
	}
	if len(interval.Tags) > 0 {
		fmt.Printf(" -- %s", strings.Join(interval.Tags, ", "))
	}
	if interval.Ref != "" {
		fmt.Printf(" -- ref:%s", interval.Ref)
	}
	fmt.Printf("\n")

	curDiff := time.Now().Sub(interval.Begin)
	var todayDur time.Duration
	intervals, _ := database.Filter([]string{KeyToday})
	for _, i := range intervals {
		todayDur += i.GetDuration()
	}
	t := tabby.New()
	t.AddLine("\t", "Started", interval.Begin.Format(datetimeFormatShort))
	if !interval.End.IsZero() {
		t.AddLine("\t", "Stopped", interval.Begin.Format(datetimeFormatShort))
	}
	t.AddLine("\t", "Current (mins)", fmtDuration(curDiff))
	t.AddLine("\t", "Total   (today)", fmtDuration(todayDur))
	t.Print()

}

func PrintRunningStatus() {
	if current, found := database.GetCurrent(); !found {
		fmt.Println("<< no tracking in progress >>")
	} else {
		PrintStatus(current)
	}
}

// Structs

const (
	StatusStarted = "started"
	StatusEnded   = "ended"
)

func Hook(name string, interval *Interval) {}

// Parsers / Lexers

const (
	ProjectPrefix      = "project:"
	ProjectPrefixShort = "proj:"
	RefPrefix          = "ref:"
	TagPrefix          = "+"
)

func lexTrack(args []string, interval *Interval) error {
	switch args[0] {
	case KeyToday:
		interval.Begin = time.Now().Truncate(24 * time.Hour)
		interval.End = time.Now().Truncate(24 * time.Hour)
	case KeyYesterday:
		interval.Begin = time.Now().Truncate(24*time.Hour).AddDate(0, 0, -1)
		interval.End = time.Now().Truncate(24*time.Hour).AddDate(0, 0, -1)
	default:
		if startDate, err := time.Parse(dateFormat, args[0]); err != nil {
			return fmt.Errorf("ERROR: Invalid date format. %s", err.Error())
		} else {
			interval.Begin = startDate
			interval.End = startDate
		}
	}

	if duration, err := time.ParseDuration(args[1]); err != nil {
		return fmt.Errorf("ERROR: Invalid duration format. %s", err.Error())
	} else {
		interval.Duration = duration
	}
	return nil
}

func lexInterval(args []string, interval *Interval) {

	interval.Raw = strings.Join(args, " ")
	// reset if relexing for annotate
	interval.Annotation = ""

	for _, part := range args {
		// tag
		if tag := strings.TrimPrefix(part, TagPrefix); tag != part {
			interval.Tags = append(interval.Tags, tag)
			continue
		}

		// project short
		if proj := strings.TrimPrefix(part, ProjectPrefixShort); proj != part {
			interval.Project = proj
			continue
		}

		// project long
		if proj := strings.TrimPrefix(part, ProjectPrefix); proj != part {
			interval.Project = proj
			continue
		}

		// ref
		if ref := strings.TrimPrefix(part, RefPrefix); ref != part {
			interval.Ref = ref
			continue
		}

		// TODO: uda

		interval.Annotation = strings.Trim(strings.Join([]string{interval.Annotation, part}, " "), " ")
	}

}

func fmtDuration(d time.Duration) string {
	d = d.Round(time.Minute)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	return fmt.Sprintf("%02d:%02d", h, m)
}

// executes the root commnad
func execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

var database Database

const databaseFilename = "db.json"

const (
	ConfDatabaseName = "databasename"
)

func init() {

	viper.SetConfigName(".gottrc")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME")

	viper.SetDefault(ConfDatabaseName, "db.json")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("[WARNING] ", err.Error())
	}

	// init root command
	rootCmd.AddCommand(trackCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(summaryCmd)
	rootCmd.AddCommand(annotateCmd)
	rootCmd.AddCommand(cancelCmd)
	rootCmd.AddCommand(continueCmd)
	rootCmd.AddCommand(editCmd)

	databaseName := viper.GetString(ConfDatabaseName)
	database = NewDatabaseJson(databaseName)
	database.Load()
}

func main() {
	defer database.Save()
	execute()
}
