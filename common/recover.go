package common

import (
	"bytes"
	"fmt"
	"log"
	"runtime"
	"runtime/debug"
	"strings"
	"sync/atomic"
	"time"
)

var stackNum int32
var stackChan = make(chan *RecoverStack, 1)

type RecoverStack struct {
	Path []string
	Msg  string
	Time time.Time
}

func Recover() {
	if err := recover(); err != nil {
		// Debug:
		es := strings.Split(string(debug.Stack()), "\n")

		go func() {
			num := atomic.AddInt32(&stackNum, 1)
			defer atomic.AddInt32(&stackNum, -1)

			if num > 20 {
				log.Printf("Recover path: %v, error: %v \n", es, err)
				return
			}

			stackChan <- &RecoverStack{
				Path: es,
				Msg:  fmt.Sprint(err),
				Time: time.Now(),
			}
		}()
	}
}

func RecStackErr() <-chan *RecoverStack {
	return stackChan
}

func ParseStackErr(bts []byte) []string {
	comtain := []byte("/src/")
	rows := bytes.Split(bts, []byte("\n"))

	var msg = make([]string, 0, len(rows))
	for _, row := range rows {
		if bytes.Contains(row, comtain) {
			m := strings.Split(string(row), " ")
			msg = append(msg, m[0])
		}
	}
	return msg
}

// GetRunPath : 获取程序运行的文件路径，skip 可跳过最下条数
func GetRunPath(skip int) string {
	_, file, line, ok := runtime.Caller(skip)
	if ok {
		path := strings.Split(file, "go/src")
		if len(path) == 2 {
			return fmt.Sprintf("%s:%d", path[1], line)
		}
		return fmt.Sprintf("%s:%d", file, line)
	}
	return "???"
}
