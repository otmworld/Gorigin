package logs

import (
	"bytes"
	"os"
	"sync"
)

type (
	LogLevel int
	DataType int
	LogTime  string
)

const (
	LevelDebug LogLevel = 1
	LevelInfo  LogLevel = 2
	LevelWarn  LogLevel = 3
	LevelError LogLevel = 4
)

const (
	DataTypeJson DataType = 1
	DataTypeByte DataType = 2
)

const (
	TimeMonth  LogTime = "200601"
	TimeDaily  LogTime = "20060102"
	TimeHour   LogTime = "2006010215"
	TimeMinute LogTime = "200601021504"
)

// It can be null.
type LogStruct struct {
	// if true, mean log data first put in cache, than cache full put in file.
	// if false, mean log data put in file as first time.
	// when LogStruct was null, Cache is true.
	Cache bool
	// cache save size (byte).
	// when cache was true and cache was null (default 1024*1024 byte)
	CacheSize int
	// log time format (default "2006-01-02 15:04:05")
	TimeFormat string
	// log file pre name. (default "log")
	FileName string
	// file save path.
	FilePath string
	// log save level. (default LevelError)
	Level LogLevel
	// how long about file create. (default TimeDay)
	FileTime LogTime
	// whether create dir to save log file. (default: false)
	Dir bool
	// write log file data type, like: json, byte
	DataType DataType
	// Error open file to write with deal function
	ErrFunc func()
}

type LogData struct {
	cache    bool
	size     int
	format   string
	name     string
	path     string
	level    LogLevel
	time     LogTime
	dir      bool
	buf      *bytes.Buffer
	file     *os.File
	flock    sync.RWMutex
	stamp    int64
	mu       sync.Mutex
	types    DataType
	function func()
}
