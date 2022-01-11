package gott

import (
	"fmt"
	"strings"
	"time"

	"github.com/cheynewallace/tabby"
)

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

func PrintStatus(current *Interval, db FilterableDatabase) {

	fmt.Printf("tracking %s", current.Annotation)
	if current.Project != "" {
		fmt.Printf(" -- proj:%s", current.Project)
	}
	if len(current.Tags) > 0 {
		fmt.Printf(" -- %s", strings.Join(current.Tags, ", "))
	}
	if current.Ref != "" {
		fmt.Printf(" -- ref:%s", current.Ref)
	}
	fmt.Printf("\n")

	curDiff := time.Since(current.Begin)
	var todayDur time.Duration
	intervals, _ := db.Filter([]string{KeyToday})
	for _, i := range intervals {
		todayDur += i.GetDuration()
	}
	t := tabby.New()
	t.AddLine("\t", "Started", current.Begin.Format(datetimeFormatShort))
	if !current.End.IsZero() {
		t.AddLine("\t", "Stopped", current.Begin.Format(datetimeFormatShort))
	}
	t.AddLine("\t", "Current (mins)", fmtDuration(curDiff))
	t.AddLine("\t", "Total   (today)", fmtDuration(todayDur))
	t.Print()

}

func PrintRunningStatus() {
	if current, found := database.GetCurrent(); !found {
		fmt.Println("<< no tracking in progress >>")
	} else {
		PrintStatus(current, database)
	}
}

func fmtDuration(d time.Duration) string {
	d = d.Round(time.Minute)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	return fmt.Sprintf("%02d:%02d", h, m)
}
