package logs

import (
	"encoding/json"
	"fmt"
	"time"
)

// NewLogs : init log server.
func NewLogs(l *LogStruct) *LogData {
	// Check struct data.
	d := l.checkStruct()
	// Open file to write and init cache.
	d.open()
	// Init file time split
	d.initStamp()

	return d
}

// SignalKill : when program quit/kill, put log cache in file
// If log data use cache wirte, should be use
// Final execution of the program
func (l *LogData) SignalKill() { l.exit() }

// WriteDebug log data with log level was Debug.
func (l *LogData) WriteDebug(args ...interface{}) error {
	if l.level == LevelDebug {
		return l.put("DEBUG", args)
	}
	return nil
}

// WriteInfo log data with log level was Info.
func (l *LogData) WriteInfo(args ...interface{}) error {
	if l.level <= LevelInfo {
		return l.put("INFO", args)
	}
	return nil
}

// WriteWarn log data with log level was Warn.
func (l *LogData) WriteWarn(args ...interface{}) error {
	if l.level <= LevelWarn {
		return l.put("WARN", args)
	}
	return nil
}

// WriteError log data with log level was Error.
func (l *LogData) WriteError(args ...interface{}) error {
	return l.put("ERROR", args)
}

// WriteFatal log data with log level was Fatal.
func (l *LogData) WriteFatal(args ...interface{}) error {
	return l.put("FATAL", args)
}

// WritePanic Write log data with log level was Fatal and panic
func (l *LogData) WritePanic(err error, args ...interface{}) {
	l.put("FATAL", args)
	// wirter log in file and close
	l.exit()
}

// WriteDebugf log data with log level was Debug.
func (l *LogData) WriteDebugf(format string, args ...interface{}) error {
	if l.level == LevelDebug {
		return l.putf("DEBUG", fmt.Sprintf(format, args...))
	}
	return nil
}

// WriteInfof : Write log data with log level was Info.
func (l *LogData) WriteInfof(format string, args ...interface{}) error {
	if l.level <= LevelInfo {
		return l.putf("INFO", fmt.Sprintf(format, args...))
	}
	return nil
}

// WriteWarnf : Write log data with log level was Warn.
func (l *LogData) WriteWarnf(format string, args ...interface{}) error {
	if l.level <= LevelWarn {
		return l.putf("WARN", fmt.Sprintf(format, args...))
	}
	return nil
}

// WriteErrorf Write log data with log level was Error.
func (l *LogData) WriteErrorf(format string, args ...interface{}) error {
	return l.putf("ERROR", fmt.Sprintf(format, args...))
}

// WriteFatalf Write log data with log level was Fatal.
func (l *LogData) WriteFatalf(format string, args ...interface{}) error {
	return l.putf("FATAL", fmt.Sprintf(format, args...))
}

// WritePanicf Write log data with log level was Fatal and panic
func (l *LogData) WritePanicf(err error, format string, args ...interface{}) {
	// wirter log in file and close
	l.putf("PANIC", fmt.Sprintf(format, args...))
	l.exit()
}

// SaveAndExit : if log use cache write, need save to exit
func (l *LogData) SaveAndExit() {
	l.exit()
}

// WriteBytes Write byte log data, not prefix.
func (l *LogData) WriteBytes(bts []byte) error {
	return l.putByte(time.Now(), bts)
}

// WriterJson : just only write json data
func (l *LogData) WriteJson(data interface{}) error {
	if bts, err := json.Marshal(data); err != nil {
		return err
	} else {
		return l.putByte(time.Now(), bts)
	}
}

// ChangeErrLevel Change log error level.
func (l *LogData) ChangeErrLevel(level LogLevel) {
	l.level = level
}
