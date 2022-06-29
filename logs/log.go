package logs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// check struct data and supplement.
func (l *LogStruct) checkStruct() *LogData {
	var d = &LogData{
		cache:  true,
		size:   1024 * 1024,
		name:   "log",
		time:   TimeDaily,
		level:  LevelError,
		path:   "./",
		format: "2006-01-02 15:04:05",
		types:  DataTypeJson,
	}

	if l == nil {
		return d
	} else {
		if l.FileName != "" {
			d.name = l.FileName
		}
		if l.FileTime != "" {
			d.time = l.FileTime
		}
		if l.Level != 0 {
			d.level = l.Level
		}
		if l.TimeFormat != "" {
			d.format = l.TimeFormat
		}
		if l.CacheSize == 0 && !l.Cache {
			d.cache = false
		} else {
			if l.CacheSize != 0 {
				d.size = l.CacheSize
			}
			d.buf = new(bytes.Buffer)
		}
		if l.Dir {
			d.dir = true
		}
		if l.FilePath != "" {
			d.path = l.FilePath
		}
		if l.DataType == DataTypeByte {
			d.types = DataTypeByte
		}
		return d
	}
}

// time utc
func (l *LogData) initStamp() {
	if timestamp, err := time.ParseInLocation(string(l.time),
		time.Now().Format(string(l.time)), time.Local); err != nil {
		panic(err)
	} else {
		l.stamp = timestamp.Unix()
	}
	l.upStamp()
}

// open file and put in struct with *os.file
func (l *LogData) open() {
	var err error
	name := filepath.Join(l.path, l.name+"."+time.Now().Format(fmt.Sprint(l.time)))

	if l.dir {
		d := l.createDir()
		name = filepath.Join(l.path, d, l.name+"."+time.Now().Format(fmt.Sprint(l.time)))
	}

	l.flock.Lock()
	defer l.flock.Unlock()
	l.file.Close()
	l.file, err = os.OpenFile(name, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		panic("Open log file error: " + err.Error())
	}
}

const (
	TimeUnitMinute = 1 * 60
	TimeUnitHour   = 60 * TimeUnitMinute
	TimeUnitDay    = 24 * TimeUnitHour
)

// sleep time to make new file open.
func (l *LogData) upStamp() {
	switch l.time {
	case TimeMonth:
		ts := time.Unix(l.stamp, 0)
		l.stamp = ts.AddDate(0, 1, 0).Unix()
	case TimeDaily:
		l.stamp += TimeUnitDay
	case TimeHour:
		l.stamp += TimeUnitHour
	case TimeMinute:
		l.stamp += TimeUnitMinute
	}
}

// JsonData : DataType is json to wirte
type JsonData struct {
	Time  string `json:"time"`
	Level string `json:"level,omitempty"`
	Body  string `json:"body"`
}

// put log data and level in buffer.
func (l *LogData) put(level string, args []interface{}) error {
	var now = time.Now()
	if l.types == DataTypeJson {
		bts, _ := json.Marshal(&JsonData{
			Time:  now.Format(l.format),
			Level: level,
			Body:  fmt.Sprint(args...),
		})
		return l.putByte(now, bts)
	} else {
		message := fmt.Sprintf("%s\t%s\t%s", now.Format(l.format), level, fmt.Sprint(args...))
		return l.putString(now, message)
	}
}

// put log data and level in buffer by string.
func (l *LogData) putf(level string, msg string) error {
	var now = time.Now()
	if l.types == DataTypeJson {
		bts, _ := json.Marshal(&JsonData{
			Time:  now.Format(l.format),
			Level: level,
			Body:  msg,
		})
		return l.putByte(now, bts)
	} else {
		msg = fmt.Sprintf("%s\t%s\t%s", now.Format(l.format), level, msg)
		return l.putString(now, msg)
	}
}

func (l *LogData) exit() {
	if l.cache {
		l.mu.Lock()
		l.sendChann(l.buf.String())
		l.mu.Unlock()
	}
	l.file.Close()
}

// put byte in cache or file.
func (l *LogData) putByte(ts time.Time, bts []byte) error {
	// check new file create time status
	if err := l.check(ts.Unix()); err != nil {
		return err
	}

	if l.cache {
		go l.sendCache(string(bts) + "\n")
	} else {
		go l.sendChann(string(bts) + "\n")
	}
	return nil
}

func (l *LogData) putString(ts time.Time, str string) error {
	// check new file create time status.
	if err := l.check(ts.Unix()); err != nil {
		return err
	}

	if l.cache {
		go l.sendCache(str + "\n")
	} else {
		go l.sendChann(str + "\n")
	}
	return nil
}

// send data to write by channel
func (l *LogData) sendCache(str string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.buf.WriteString(str)
	if l.buf.Len() >= l.size {
		l.sendCache(l.buf.String())
		l.buf.Reset()
	}
	return
}

// send data to write by channel
func (l *LogData) sendChann(data string) {
	data = strings.Replace(data, "\\u003c", "<", -1)
	data = strings.Replace(data, "\\u003e", ">", -1)
	data = strings.Replace(data, "\\u0026", "&", -1)

	l.flock.RLock()
	_, err := l.file.WriteString(data)
	l.flock.RUnlock()
	if err != nil {
		l.reset(data, 1)
	}
}

func (l *LogData) reset(data string, num int) {
	l.open()

	l.flock.Lock()
	_, err := l.file.WriteString(data)
	l.flock.Unlock()
	if err != nil {
		if num >= 4 {
			l.function()
		}
		time.Sleep(time.Second)
		l.reset(data, num+1)
	}
}

// check new file open.
func (l *LogData) check(now int64) error {
	if l.stamp <= now {
		l.mu.Lock()
		if l.stamp > now {
			return nil
		}
		if l.cache {
			l.sendCache(l.buf.String())
			l.buf.Reset()
		}

		go func(file *os.File) {
			time.Sleep(time.Second * 10)
			file.Close()
		}(l.file)

		l.open()
		l.upStamp()
		l.mu.Unlock()
	}
	return nil
}

// make dir about FileTime.
func (l *LogData) createDir() string {
	// Create log file dir with year.
	l.create(time.Now().Format("2006"))

	// Create log file dir with month.
	if l.time != TimeMonth {
		l.create(time.Now().Format("2006/01"))
	} else {
		return time.Now().Format("2006/")
	}
	// Create log file dir with day.
	if l.time != TimeDaily {
		l.create(time.Now().Format("2006/01/02"))
	} else {
		return time.Now().Format("2006/01/")
	}
	// Create log file dir with hour.
	if l.time != TimeHour {
		l.create(time.Now().Format("2006/01/02/15"))
	} else {
		return time.Now().Format("2006/01/02/")
	}
	return time.Now().Format("2006/01/02/15/")
}

// check dir exist and create.
func (l *LogData) create(path string) {
	path = filepath.Join(l.path, path)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			if err := os.Mkdir(path, os.ModePerm); err != nil {
				panic("Create log file dir error!")
			}
		}
	}
}
