package timer

import "time"

type TimerStruct interface {
	// SetTimeZone : verify time eg:[-08:00, +08:00]
	// If want use UTC to time zone : '+00:00'
	SetTimeZone(string) error

	AddTimerRun(string, int, bool, interface{}, func(interface{})) (error, uint64)
	AddDurationFunction(time.Duration, int, func()) uint64
	AddDurationArgument(time.Duration, int, interface{}, func(interface{})) uint64
	AddDurationBoolean(time.Duration, func() bool) uint64
	AddDurationArgBool(time.Duration, interface{}, func(interface{}) bool) uint64

	// cancel timer goroutine and clear exit
	CloseAndExit()
}

// function run interval = interval * 100ms
// default: 500ms
func NewTimer(duration time.Duration) TimerStruct {
	return TimerStruct(newTimer(duration))
}
