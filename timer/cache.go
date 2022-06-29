package timer

import (
	"sync"
	"sync/atomic"
	"time"
)

type TimerData struct {
	one  sync.Map
	two  sync.Map
	zone int64  // time zone of ms
	uniq uint64 // unique id
	part int64  // tmp cache time interval check
	next uint32 // seq number to check tmp cache
	done chan struct{}
}

func (t *TimerData) GetTimeMill() int64 {
	return time.Now().UnixNano() / 1000_000
}

func (t *TimerData) GetTimeMillAdd(duration time.Duration) int64 {
	return time.Now().Add(duration).UnixNano() / 1000_000
}

// InitTicker : init timer ticker
// base interval is 100ms, dafualt 500ms, [100ms * interval]
func newTimer(duration time.Duration) *TimerData {
	var tmp = &TimerData{done: make(chan struct{}, 1)}
	tmp.part = int64(duration) * 5 / 1000_000
	if _, zone := time.Now().Zone(); zone != 0 {
		tmp.zone = int64(zone)
	}

	go func(s *TimerData, timer time.Duration) {
		ticker := time.NewTicker(timer)
		for {
			select {
			case <-s.done:
				ticker.Stop()
				return
			case <-ticker.C:
				if count := atomic.AddUint32(&s.next, 1); count >= 4 {
					go s.checkSave()
					atomic.SwapUint32(&s.next, 0)
				}

				now := s.GetTimeMill()
				s.one.Range(func(key, value interface{}) bool {
					if v, ok := value.(*TimerFunc); ok && v != nil {
						if v.next <= now {
							go s.run(v)
							s.one.Delete(key)
						}
					} else {
						s.one.Delete(key)
					}
					return true
				})
			}
		}
	}(tmp, duration)
	return tmp
}

func (s *TimerData) run(data *TimerFunc) {
	if data.funcType < FuncBool {
		switch data.times {
		case 0:
			return
		case 1:
			data.RunFunc()
			return
		case -1:
		default:
			data.times--
		}
	}
	if data.RunFunc() {
		data.next += data.interval
		s.put(data)
	}
}

func (t *TimerData) put(data *TimerFunc) {
	if data.next <= t.GetTimeMill()+t.part {
		t.one.Store(data.uniq, data)
	} else {
		t.two.Store(data.uniq, data)
	}
}

// When insert new data to sort
func (t *TimerData) checkSave() {
	next := t.GetTimeMill() + t.part
	t.two.Range(func(key, value interface{}) bool {
		if v, ok := value.(*TimerFunc); ok && value != nil {
			if v.next <= next {
				t.one.Store(key, value)
				t.two.Delete(key)
			}
		} else {
			t.two.Delete(key)
		}
		return true
	})
}
