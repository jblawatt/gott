package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	"text/tabwriter"

	"github.com/cheynewallace/tabby"
	uuid "github.com/nu7hatch/gouuid"
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

		filterList := []filterFunc{}

		if len(args) > 0 {
			for _, arg := range args {
				switch arg {
				case KeyToday:
					filterList = append(filterList, createDateFilter(time.Now()))
					continue
				case KeyYesterday:
					yesterday := time.Now().AddDate(0, 0, -1)
					filterList = append(filterList, createDateFilter(yesterday))
					continue
				case KeyWeek:
					begin := time.Now()
					for begin.Weekday() != time.Monday {
						begin = begin.AddDate(0, 0, -1)
					}
					filterList = append(filterList, createDateRangeFilter(begin, time.Now()))
					continue
				case KeyMonth:
					begin := time.Now()
					for begin.Day() != 1 {
						begin = begin.AddDate(0, 0, -1)
					}
					filterList = append(filterList, createDateRangeFilter(begin, time.Now()))
					continue
				case KeyAll:
					continue
				default:
					if t, err := time.Parse("2006-01-02", arg); err != nil {
						fmt.Fprintf(os.Stderr, "ERROR: invalid summary filter %s, %s\n", arg, err.Error())
						os.Exit(1)
					} else {
						filterList = append(filterList, createDateFilter(t))
					}
					continue
				}
			}
		} else {
			// default only today
			filterList = append(filterList, createDateFilter(time.Now()))
		}

		t.AddHeader("CWEEK", "DAY", "BEGIN", "END", "DURATION", "PROJECT", "TAG", "ANNOTATION")

		sort.SliceStable(database.Intervals, func(i, j int) bool {
			a := database.Intervals[i]
			b := database.Intervals[j]
			return a.Begin.Before(b.Begin)
		})

		weekGroup := 0
		weekText := ""
		dayGroup := ""
		dayText := ""
		var dayDurationSum time.Duration
		var weekDurationSum time.Duration
		for _, interval := range database.Intervals {
			if !applyFilter(interval, filterList) {
				continue
			}

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
		t.AddLine("wk =", "", "", "", fmtDuration(dur), "", "", "")
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
		if len(database.Intervals) > 0 {
			last := database.Intervals[len(database.Intervals)-1]
			database.Start(NewInterval(strings.Split(last.Raw, " ")))
			PrintRunningStatus()
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

var editCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit the intervals in the provided timespan",
	Run: func(cmd *cobra.Command, args []string) {
		// filter
		// create tempfile
		// write tabtable to tempfile
		// start $EDITOR
		// parse tempfile to item

		var beforeIDs []string

		f, _ := ioutil.TempFile(os.TempDir(), ".md")
		defer f.Close()

		f.WriteString("# Edit below values to change tracking data\n")
		f.WriteString("# - delete rows to delete \n")
		f.WriteString("# - set time values 00:00 or leave empty to just set duration \n")
		f.WriteString("# - leave duration empty or 0s if you set date \n")
		f.WriteString("\n")

		writer := tabwriter.NewWriter(f, 0, 0, 2, ' ', 0)
		for _, i := range database.Intervals {
			beforeIDs = append(beforeIDs, i.ID)
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
		f.WriteString("\n\n# NEW ENTRIES HERE ############################################# \n")
		f.WriteString("# [ID (empty)] [DATE] [BEGIN] [END] [DURATION] [ANNOTATION] \n")
		f.WriteString("# [] [] [] [] [] [] \n")

		excCmd := exec.Command("nvim", f.Name())
		excCmd.Stdout = os.Stdout
		excCmd.Stderr = os.Stderr
		excCmd.Stdin = os.Stdin
		excCmd.Run()

		f.Seek(0, 0)

		scanner := bufio.NewScanner(f)
		scanner.Split(bufio.ScanLines)
		// pattern := regexp.MustCompile(`[(.*)] +[(.*)] +[(.*)] +[(.*)] +[(.*)] +[(.*)]`)
		pattern := regexp.MustCompile(`\[[\w\w\-\_0-9 :]*\]`)

		var afterIDs []string
		var i int = 0
		for scanner.Scan() {
			t := scanner.Text()
			// ignore commented line
			if strings.HasPrefix(t, "#") {
				continue
			}
			// ignore empty line
			if len(strings.Trim(t, " ")) == 0 {
				continue
			}
			result := pattern.FindAllString(t, -1)
			var cleaned []string
			for _, col := range result {
				cleaned = append(cleaned, strings.Trim(stripBraces(col), " "))
			}
			var id, date, begin, end, duration, annotation string
			unpackSlice(cleaned, &id, &date, &begin, &end, &duration, &annotation)
			var interval *Interval
			annotationSlice := strings.Split(annotation, " ")
			if id != "" {
				afterIDs = append(afterIDs, id)
				if intl, found := database.Get(id); found {
					interval = intl
				} else {
					fmt.Fprintf(os.Stderr, "[ERROR at %d] no interval with id %s\n", i, id)
					os.Exit(1)
				}
				lexInterval(annotationSlice, interval)
			} else {
				intv := NewInterval(annotationSlice)
				interval = &intv
			}
			if date == "" {
				fmt.Fprintf(os.Stderr, "[ERROR at %d] you have to provide a valid date. empty is not valid.\n", i)
				os.Exit(1)
			}
			tdate, tdateErr := time.Parse(dateFormat, date)
			if tdateErr != nil {
				fmt.Fprintf(os.Stderr, "[ERROR at %d] parsing date '%s': %s\n", i, date, tdateErr.Error())
				os.Exit(1)
			}
			tbegin, terr := time.Parse(timeFormat, begin)
			if begin != "" && terr != nil {
				fmt.Fprintf(os.Stderr, "[ERROR at %d] parsing begin time '%s': %s\n", i, begin, terr.Error())
				os.Exit(1)
			}
			tend, tendErr := time.Parse(timeFormat, end)
			if begin != "" tendErr != nil {
				fmt.Fprintf(os.Stderr, "[ERROR at %d] parsing end time '%s': %s\n", i, end, tendErr.Error())
				os.Exit(1)
			}

			if tbegin.Format(timeFormat) != "00:00" && tend.Format(timeFormat) != "00:00" {
				interval.Begin = tdate.Add(time.Duration(tbegin.Hour())*time.Hour + time.Duration(tbegin.Minute())*time.Minute)
				interval.End = tdate.Add(time.Duration(tend.Hour())*time.Hour + time.Duration(tend.Minute())*time.Minute)
			} else {
				tdur, tdurErr := time.ParseDuration(duration)
				if tdurErr != nil {
					fmt.Fprintf(os.Stderr, "[ERROR at %d] no valid times and no duration.\n", i)
					os.Exit(1)
				}
				interval.Begin = tbegin
				interval.End = tbegin
				interval.Duration = tdur
			}
			i += 1

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
	now := time.Now()
	for _, i := range database.Intervals {
		// TODO: better check with end also
		if i.Begin.Year() == now.Year() && i.Begin.Month() == now.Month() && i.Begin.Day() == now.Day() {
			todayDur += i.GetDuration()
		}
	}
	t := tabby.New()
	t.AddLine("\t", "Started", interval.Begin.Format(datetimeFormatShort))
	if !interval.End.IsZero() {
		t.AddLine("\t", "Stopped", interval.Begin.Format(datetimeFormatShort))
	}
	t.AddLine("\t", "Current (mins)", fmtDuration(curDiff))
	t.AddLine("\t", "Total (today)", fmtDuration(todayDur))
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

type Interval struct {
	ID         string
	Begin      time.Time
	End        time.Time
	Duration   time.Duration
	Tags       []string
	Project    string
	Ref        string
	Annotation string
	Raw        string
	UDA        map[string]interface{}
	Status     string
}

func (i *Interval) GetDuration() time.Duration {
	// completely the same. duration only
	if i.End.Equal(i.Begin) {
		return i.Duration
	}
	if i.End.IsZero() {
		return time.Now().Sub(i.Begin)
	}
	return i.End.Sub(i.Begin)
}

func NewInterval(raw []string) (interval Interval) {
	id, _ := uuid.NewV4()
	interval.ID = id.String()
	lexInterval(raw, &interval)
	return interval
}

func StopInterval(i *Interval) {
	i.End = time.Now()
	i.Status = StatusEnded
}

type Database struct {
	Current   string
	Intervals []*Interval
}

func (d *Database) GetCurrent() (*Interval, bool) {
	return d.Get(d.Current)
}

func (d *Database) Get(id string) (*Interval, bool) {
	for _, i := range d.Intervals {
		if i.ID == id {
			return i, true
		}
	}
	return &Interval{}, false
}

func (d *Database) Start(interval Interval) {
	interval.Begin = time.Now()
	interval.Status = StatusStarted
	d.Intervals = append(d.Intervals, &interval)
	if d.Current != "" {
		d.Stop()
	}
	d.Current = interval.ID
}

func (d *Database) Cancel() {
	if cur, found := d.GetCurrent(); found {
		for i, interval := range d.Intervals {
			if cur.ID == interval.ID {
				d.Intervals = append(d.Intervals[:i], d.Intervals[i+1:]...)
				d.Current = ""
				break
			}
		}
	}
}

func (d *Database) Stop() {
	for _, interval := range d.Intervals {
		if interval.ID == d.Current {
			d.Current = ""
			StopInterval(interval)
			break
		}
	}
}

func (d *Database) Append(interval Interval) {
	d.Intervals = append(d.Intervals, &interval)
}

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

func loadDatabase(name string) {
	if _, err := os.Stat(name); errors.Is(err, os.ErrNotExist) {
		saveDatabase(name)
	}
	file, _ := ioutil.ReadFile(databaseFilename)
	_ = json.Unmarshal([]byte(file), &database)
}

func saveDatabase(name string) {
	data, _ := json.Marshal(database)
	ioutil.WriteFile(name, data, 0644)
}

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
	loadDatabase(databaseName)
}

func main() {
	databaseName := viper.GetString(ConfDatabaseName)
	defer saveDatabase(databaseName)
	execute()
}
