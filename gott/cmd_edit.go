package gott

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"text/tabwriter"
	"time"

	uuid "github.com/nu7hatch/gouuid"
	"github.com/spf13/cobra"
)

func writeEditFile(f *os.File, intervals []*Interval, filterArgs []string) {

	f.WriteString("# Edit below values to change tracking data\n")
	f.WriteString("# - delete rows to delete \n")
	f.WriteString("# - set time values 00:00 or leave empty to just set duration \n")
	f.WriteString("# - leave duration empty or 0s if you set date \n")
	f.WriteString("\n")

	writer := tabwriter.NewWriter(f, 0, 0, 2, ' ', 0)
	for _, i := range intervals {
		line := fmt.Sprintf(
			"(%s)\t(%s)\t(%s)\t(%s)\t(%s)\t(%s)\n",
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
	f.WriteString("# (ID [leave empty]) (DATE) (BEGIN) (END) (DURATION) (ANNOTATION)\n")
	f.WriteString(fmt.Sprintf("\n# () (%s) () () () ()\n", time.Now().Format(dateFormat)))

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

func parseEditLine(line string) (Interval, error) {

	pattern := regexp.MustCompile(`\([\w\-\_0-9 :]*\)`)
	result := pattern.FindAllString(line, -1)
	var cleaned []string
	for _, col := range result {
		cleaned = append(cleaned, strings.Trim(stripBraces(col), " "))
	}

	var id, date, begin, end, duration, annotation string
	unpackSlice(cleaned, &id, &date, &begin, &end, &duration, &annotation)
	annotationSlice := strings.Split(annotation, " ")
	interval := NewInterval(annotationSlice)

	if date == "" {
		return interval, fmt.Errorf("date is empty but should be filled with format YYYY-MM-DD")
	}

	tdate, tdateErr := time.Parse(dateFormat, date)
	if tdateErr != nil {
		return interval, fmt.Errorf("error parsing date '%s'. Error: %s", date, tdateErr.Error())
	}

	tbegin, terr := time.Parse(timeFormat, begin)
	if begin != "" && terr != nil {
		return interval, fmt.Errorf("error parsing begin time '%s': %s", begin, terr.Error())
	}

	tend, tendErr := time.Parse(timeFormat, end)
	if begin != "" && tendErr != nil {
		return interval, fmt.Errorf("error parsing end time '%s': %s", end, tendErr.Error())
	}

	if tbegin.Format(timeFormat) != "00:00" && tend.Format(timeFormat) != "00:00" {
		interval.Begin = tdate.Add(time.Duration(tbegin.Hour())*time.Hour + time.Duration(tbegin.Minute())*time.Minute)
		interval.End = tdate.Add(time.Duration(tend.Hour())*time.Hour + time.Duration(tend.Minute())*time.Minute)
	} else {
		tdur, tdurErr := time.ParseDuration(duration)
		if tdurErr != nil {
			return interval, fmt.Errorf("error parsing duration: %s", tdurErr.Error())
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
			return nil, err
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

		beforeContent, _ := ioutil.ReadFile(f.Name())
		runEditFile(f)
		afterContent, _ := ioutil.ReadFile(f.Name())

		if string(beforeContent) == string(afterContent) {
			fmt.Println("file unchanged. nothing to do.")
			return
		}

		// TODO: optimize:
		// - test if content changed
		// - test diff content and not walk through it
		// - do not update every interval
		// - better diff

		f.Seek(0, 0)

		editIntervals, err := parseEditFile(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: error in parsing file: %s", err.Error())
			os.Exit(1)
		}

		var afterIDs []string

		for _, intv := range editIntervals {
			if intv.ID == "" {
				id, _ := uuid.NewV4()
				intv.ID = id.String()
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

func init() {
	rootCmd.AddCommand(editCmd)
}
