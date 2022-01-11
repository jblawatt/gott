package gott

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"time"
)

const (
	ConfDatabaseName = "databasename"
	databseFilename  = "db.json"
)

type Database interface {
	GetCurrent() (*Interval, bool)
	Get(id string) (*Interval, bool)
	Start(interval Interval)
	Cancel()
	Stop()
	Append(interval Interval)
	AppendPtr(interval *Interval)
	RemoveById(id string)
	// Remove(interval *Interval)
	Filter(args []string) ([]*Interval, error)
	Apply(interval Interval) error
	Load() error
	Save() error
	Count() int
	Latest() (*Interval, error)
}

type DatabaseJson struct {
	filename  string
	Current   string
	Intervals []*Interval
}

func (d *DatabaseJson) GetCurrent() (*Interval, bool) {
	return d.Get(d.Current)
}

func (d *DatabaseJson) Get(id string) (*Interval, bool) {
	for _, i := range d.Intervals {
		if i.ID == id {
			return i, true
		}
	}
	return nil, false
}

func (d *DatabaseJson) Start(interval Interval) {
	interval.Begin = time.Now()
	interval.Status = StatusStarted
	d.Intervals = append(d.Intervals, &interval)
	if d.Current != "" {
		d.Stop()
	}
	d.Current = interval.ID
}

func (d *DatabaseJson) Cancel() {
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

func (d *DatabaseJson) Stop() {
	for _, interval := range d.Intervals {
		if interval.ID == d.Current {
			d.Current = ""
			interval.Stop()
			break
		}
	}
}

func (d *DatabaseJson) Append(interval Interval) {
	d.AppendPtr(&interval)
}

func (d *DatabaseJson) AppendPtr(interval *Interval) {
	d.Intervals = append(d.Intervals, interval)
}

func (d *DatabaseJson) RemoveById(id string) {
	// TODO optimize
	for i, interval := range d.Intervals {
		if interval.ID == id {
			d.Intervals = append(d.Intervals[:i], d.Intervals[i+1:]...)
		}
	}
}

func (d *DatabaseJson) Filter(args []string) ([]*Interval, error) {
	var resultSet []*Interval

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
					return resultSet, fmt.Errorf("invalid summary filter %s, %s", arg, err.Error())
				} else {
					filterList = append(filterList, createDateFilter(t))
				}
				continue
			}
		}
	}

	// sort by date
	sort.SliceStable(d.Intervals, func(i, j int) bool {
		a := d.Intervals[i]
		b := d.Intervals[j]
		return a.Begin.Before(b.Begin)
	})

	for _, interval := range d.Intervals {
		if applyFilter(interval, filterList) {
			resultSet = append(resultSet, interval)
		}
	}

	return resultSet, nil
}

func (d *DatabaseJson) Apply(i Interval) error {
	e, found := d.Get(i.ID)
	if !found {
		return fmt.Errorf("Interval with id %s does not exist", i.ID)
	}
	e.Begin = i.Begin
	e.End = i.End
	e.Duration = i.Duration
	e.Project = i.Project
	e.Ref = i.Ref
	e.Tags = i.Tags
	e.UDA = i.UDA
	e.Annotation = i.Annotation
	e.Status = i.Status
	e.Raw = i.Raw
	return nil
}

func (d *DatabaseJson) Count() int {
	return len(d.Intervals)
}

func (d *DatabaseJson) Save() error {
	data, _ := json.Marshal(d)
	ioutil.WriteFile(d.filename, data, 0644)
	return nil
}

func (d *DatabaseJson) Load() error {
	if _, err := os.Stat(d.filename); errors.Is(err, os.ErrNotExist) {
		d.Save()
	}
	if file, errFile := ioutil.ReadFile(d.filename); errFile != nil {
		return fmt.Errorf("error reading file. %s", errFile.Error())
	} else {
		if unmarshalErr := json.Unmarshal([]byte(file), &d); unmarshalErr != nil {
			return fmt.Errorf("error unmashaling database file: %s", unmarshalErr.Error())
		}
	}
	return nil
}

func (d *DatabaseJson) Latest() (*Interval, error) {
	if length := d.Count(); length > 0 {
		return d.Intervals[length-1], nil
	} else {
		return nil, errors.New("database empty. no latest")
	}
}

func NewDatabaseJson(filename string) *DatabaseJson {
	return &DatabaseJson{
		filename: filename,
	}
}
