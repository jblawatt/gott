package gott

import "time"

const (
	KeyToday     = ":today"
	KeyYesterday = ":yesterday"
	KeyWeek      = ":week"
	KeyMonth     = ":month"
	KeyAll       = ":all"
)

var Keys = []string{KeyToday, KeyYesterday, KeyWeek, KeyMonth, KeyAll}

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
