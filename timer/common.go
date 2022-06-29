package timer

import (
	"sort"
)

// parse time format
const (
	TimeFormatString = "2006-01-02 15:04:05"
	TimeFormatLength = 19
)

// time unit
const (
	SecondTimeUnit = 1000
	MinuteTimeUnit = 60 * SecondTimeUnit
	HourTimeUnit   = 60 * MinuteTimeUnit
	DayTimeUnit    = 24 * HourTimeUnit
)

type FunctionType int

const (
	FuncOnly FunctionType = iota
	FuncArg
	FuncBool
	FuncArgBool
)

// TimerFunc ticker function struct
type TimerFunc struct {
	next        int64 // Next run time
	uniq        uint64
	interval    int64 // 时间间隔
	times       int   // run times
	funcType    FunctionType
	funcOnly    func()
	funcBool    func() bool
	funcArg     func(interface{})
	funcArgBool func(interface{}) bool

	msg interface{}
}

func (t *TimerFunc) RunFunc() bool {
	switch t.funcType {
	case FuncOnly:
		t.funcOnly()
	case FuncBool:
		return t.funcBool()
	case FuncArg:
		t.funcArg(t.msg)
	case FuncArgBool:
		return t.funcArgBool(t.msg)
	}
	return true
}

// ByNext sort by next time
type ByNext []*TimerFunc

func (t ByNext) Len() int {
	return len(t)
}

func (t ByNext) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func (t ByNext) Less(i, j int) bool {
	return t[i].next < t[j].next
}

func TimeSplit(rows []*TimerFunc, timestamp int64) ([]*TimerFunc, []*TimerFunc) {
	if len(rows) == 0 {
		return nil, nil
	}
	sort.Sort(ByNext(rows))
	for i, row := range rows {
		if row.next > timestamp {
			return rows[:i], rows[i:]
		}
	}
	return rows, nil
}
