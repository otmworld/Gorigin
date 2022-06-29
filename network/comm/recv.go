package comm

import (
	"strings"
)

// When FuncMsg.FuncID < 100; is Built-in api

type CommReq struct {
	Data interface{} `json:"data"`
	Num  int         `json:"num"`
	Func int         `json:"fid"`
}

const (
	// 内置接收接口
	BUILT_IN_MAX  = 30
	WATCH_IN_MAX  = 100
	BUILT_IN_NAME = "builtin"

	// didn't return data
	PingNetwork  = 1
	DialRegister = 2

	// make builtin retrun
	UpFuncMapList = 11
	UpNodeConnMsg = 12
	UpServerState = 13
	UpWatcherList = 14
)

func SplitServName(name string) string {
	rows := strings.Split(name, ".")
	if len(rows) == 1 {
		return name
	} else if len(rows) == 2 {
		return rows[0]
	} else if len(rows) == 3 {
		return rows[1]
	}
	return ""
}

func SplitFuncName(name string) string {
	rows := strings.Split(name, ".")
	if len(rows) == 1 {
		return name
	} else if len(rows) == 2 {
		return rows[1]
	} else if len(rows) == 3 {
		return rows[2]
	}
	return ""
}

func SplitApiName(name string) (string, string) {
	rows := strings.Split(name, ".")
	if len(rows) == 2 {
		return rows[0], rows[1]
	} else if len(rows) == 3 {
		return rows[1], rows[2]
	}
	return "", ""
}
