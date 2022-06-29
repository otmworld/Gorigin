package logs

import (
	"fmt"
	"testing"
	"time"
)

// func TestLogJustLogInit(t *testing.T) {
// 	l := LogInit(&LogStruct{
// 		Cache:    false,
// 		FileTime: TimeMinute,
// 	})

// 	for i := 1; i < 1000; i++ {
// 		l.WriteError("<xml>123</xml>")
// 		time.Sleep(time.Millisecond * 100)
// 	}

// 	time.Sleep(time.Second * 80)
// }

/*
func TestDirCreate(t *testing.T) {
	l := LogInit(&LogStruct{
		FileTime: TimeHour,
		FilePath: "data",
		Dir:      true,
	})

	l.WriteError("message")
}

func TestFileCreate(t *testing.T) {
	l := LogInit(&LogStruct{
		FileName:   "log.data",
		TimeFormat: "15:04:05",
		FileTime:   TimeMinute,
	})

	for i := 0; i <= 65; i++ {
		l.WriteError(i)
		time.Sleep(time.Second)
	}
}
*/

// func TestLogLevel(t *testing.T) {
// 	l := LogInit(&LogStruct{
// 		Level: LevelWarn,
// 	})

// 	err := l.WriteWarn("message", "111")
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	err = l.WriteError("message", "222")
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	err = l.WriteWarnf("%s %d", "message", 333)
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	err = l.WriteErrorf("%s %d", "message", 444)
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

func TestCacheWrite(t *testing.T) {
	str := []string{
		"Stray birds of summer come to my window to sing and fly away. " +
			"And yellow leaves of autumn, which have no songs, flutter and fall there with a sigh. ",

		"Sorrow is hushed into peace in my heart like the evening among the silent trees.",

		"Listen, my heart, to the whispers of the world with which it makes love to you.",
	}

	l := NewLogs(&LogStruct{
		TimeFormat: "15:04:05",
		Cache:      true,
		CacheSize:  1024,
	})

	for _, s := range str {
		fmt.Println(1024/len([]byte(s)) + 1)
	}

	for i := 0; i <= 10; i++ {
		l.WriteError(str[0])
		time.Sleep(time.Millisecond * 500)
	}

	for i := 0; i <= 20; i++ {
		l.WriteError(str[1])
		time.Sleep(time.Millisecond * 500)
	}

	for i := 0; i <= 20; i++ {
		l.WriteError(str[2])
		time.Sleep(time.Millisecond * 500)
	}
}
