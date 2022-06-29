# Time Ticker

### use timer eg
```go
	// init timer ticker
	timer := NewTimer(time.Millisecond * 500)
	
	// SetTimeZone : verify time eg:[-08:00, +08:00]
	// If want use UTC to time zone : '+00:00'
	timer.SetTimeZone("+08:00")

	// NewTimer ：make new ticker function
	// stamp -> Time timing: 15:04:05; 04:05; 05;
	// stamp -> time interval: s-m-h-d:  10s; 30m; 60h; 7d;
	// times: 	run times [-1:forever, 0:return not run]
	// run:  	defalut: running next time, if true run one times now.
	timer.AddTimerRun(stamp string, times int, run bool, msg interface{}, function func(interface{})) (error, uint64)

	// exit timer
	timer.CloseAndExit()
```