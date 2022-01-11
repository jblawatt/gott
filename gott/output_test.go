package gott

import (
	"bytes"
	"fmt"
	"testing"
	"text/tabwriter"
	"time"

	"github.com/cheynewallace/tabby"
	"github.com/stretchr/testify/assert"
)

func NewMockTabby() (*bytes.Buffer, *tabby.Tabby) {
	buff := bytes.NewBufferString("")
	writer := tabwriter.NewWriter(buff, 0, 0, 2, ' ', 0)
	tby := tabby.NewCustom(writer)
	return buff, tby
}

func TestDaySumLine(t *testing.T) {
	buff, tby := NewMockTabby()
	dur := 1 * time.Hour
	DaySumLine(tby, dur)
	tby.Print()
	expected := fmt.Sprintf("      day =  %s      \n", fmtDuration(dur))
	assert.Equal(
		t,
		expected,
		buff.String(),
	)
}

func TestWeekSumLine(t *testing.T) {
	buff, tby := NewMockTabby()
	dur := 1 * time.Hour
	WeekSumLine(tby, dur)
	tby.Print()
	expected := fmt.Sprintf("    wk =    %s      \n", fmtDuration(dur))
	assert.Equal(
		t,
		expected,
		buff.String(),
	)
}

type MockFilterableDatabase struct{}

func (m *MockFilterableDatabase) Filter(args []string) ([]*Interval, error) {
	return nil, nil
}

func TestPrintStatus(t *testing.T) {
	cur := &Interval{}
	db := &MockFilterableDatabase{}
	PrintStatus(cur, db)

}
