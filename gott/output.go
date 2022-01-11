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

func fmtDuration(d time.Duration) string {
	d = d.Round(time.Minute)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	return fmt.Sprintf("%02d:%02d", h, m)
}
