package gott

import (
	"fmt"
	"strings"
	"time"
)

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
