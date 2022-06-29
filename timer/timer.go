package timer

import (
	"errors"
	"strconv"
	"sync/atomic"
	"time"
)

// NewTimer ：make new ticker function
// stamp -> Time timing: 15:04:05; 04:05; 05;
// stamp -> time interval: s-m-h-d:  10s; 30m; 60h; 7d;
// times: 	run times [-1:forever, 0:return not run]
// run:  	defalut: running next time, if true run one times now.
func (s *TimerData) AddTimerRun(stamp string, times int, run bool, msg interface{}, function func(interface{})) (error, uint64) {
	if stamp == "" || function == nil {
		return errors.New("time stamp or function wrong"), 0
	}

	if next, interval, err := s.checkTime(stamp); err != nil {
		return err, 0
	} else {
		if times < 0 {
			times = -1
		} else if times == 0 {
			return errors.New("ticker run times can not be zero"), 0
		}

		if run {
			switch times {
			case 1:
				function(msg)
				return nil, 0
			case -1:
				function(msg)
			default:
				times--
				function(msg)
			}
		}

		var uniq = atomic.AddUint64(&s.uniq, 1)
		s.put(&TimerFunc{
			uniq:     uniq,
			funcType: FuncArg,
			funcArg:  function,
			times:    times,
			next:     next,
			interval: interval,
			msg:      msg,
		})
		return nil, uniq
	}
}

// NewRunDuration : Make a new function run
// times: [-1 meas forever], [0 meas not run]
// if argument or function is nil, return 0 with not run
func (s *TimerData) AddDurationArgument(duration time.Duration, times int, arg interface{}, function func(interface{})) uint64 {
	if times == 0 {
		return 0
	} else if times < 0 {
		times = -1
	}
	if function == nil || arg == nil {
		return 0
	}
	var uniq = atomic.AddUint64(&s.uniq, 1)
	s.put(&TimerFunc{
		uniq:     uniq,
		next:     s.GetTimeMillAdd(duration),
		times:    times,
		interval: int64(duration / 1000_000),
		funcType: FuncArg,
		funcArg:  function,
		msg:      arg,
	})
	return uniq
}

func (s *TimerData) AddDurationFunction(duration time.Duration, times int, function func()) uint64 {
	if times == 0 {
		return 0
	} else if times < 0 {
		times = -1
	}
	if function == nil {
		return 0
	}
	var uniq = atomic.AddUint64(&s.uniq, 1)
	s.put(&TimerFunc{
		uniq:     uniq,
		next:     s.GetTimeMillAdd(duration),
		times:    times,
		interval: int64(duration / 1000_000),
		funcType: FuncOnly,
		funcOnly: function,
	})
	return uniq
}

func (s *TimerData) AddDurationBoolean(duration time.Duration, function func() bool) uint64 {
	var uniq = atomic.AddUint64(&s.uniq, 1)
	s.put(&TimerFunc{
		uniq:     uniq,
		next:     s.GetTimeMillAdd(duration),
		funcType: FuncBool,
		funcBool: function,
		interval: int64(duration / 1000_000),
	})
	return uniq
}

func (s *TimerData) AddDurationArgBool(duration time.Duration, arg interface{}, function func(interface{}) bool) uint64 {
	var uniq = atomic.AddUint64(&s.uniq, 1)
	s.put(&TimerFunc{
		uniq:        uniq,
		next:        s.GetTimeMillAdd(duration),
		funcType:    FuncArgBool,
		funcArgBool: function,
		interval:    int64(duration / 1000_000),
		msg:         arg,
	})
	return uniq
}

func (s *TimerData) CloseAndExit() {
	s.done <- struct{}{}
}

// check timerstamp value
func (s *TimerData) checkTime(stamp string) (int64, int64, error) {
	var err error
	var temp int
	var interval int64
	var next = s.GetTimeMill()

	switch stamp[len(stamp)-1:] {
	case "s", "S":
		temp, err = strconv.Atoi(stamp[:len(stamp)-1])
		interval = int64(temp * SecondTimeUnit)

	case "m", "M":
		temp, err = strconv.Atoi(stamp[:len(stamp)-1])
		interval = int64(temp * MinuteTimeUnit)

	case "h", "H":
		temp, err = strconv.Atoi(stamp[:len(stamp)-1])
		interval = int64(temp * HourTimeUnit)

	case "d", "D":
		temp, err = strconv.Atoi(stamp[:len(stamp)-1])
		interval = int64(temp * DayTimeUnit)

	case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9":
		var t time.Time
		timeString := time.Now().Format(TimeFormatString)
		switch len(stamp) {
		case 2: // second
			t, err = time.ParseInLocation(TimeFormatString, timeString[:17]+stamp, time.UTC)
			next = (t.UTC().Unix() - s.zone) / 60 * 60_000
			interval = MinuteTimeUnit

		case 5: // min
			t, err = time.ParseInLocation(TimeFormatString, timeString[:14]+stamp, time.UTC)
			next = (t.UTC().Unix() - s.zone) / 3600 * 3600_000
			interval = HourTimeUnit

		case 8: // hour
			t, err = time.ParseInLocation(TimeFormatString, timeString[:11]+stamp, time.UTC)
			next = (t.UTC().Unix() - s.zone) / 86400 * 86400_000
			interval = DayTimeUnit

		default:
			err = errors.New("can't parst time, please check it")
		}

	default:
		err = errors.New("can't parst stamp value, please check it")
	}

	if err == nil && interval > 0 {
		for next <= s.GetTimeMill() {
			next += interval
		}
	}
	return next, interval, err
}
