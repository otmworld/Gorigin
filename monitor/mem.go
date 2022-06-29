package monitor

import (
	"bytes"
	"io/ioutil"
	"strconv"
	"strings"
)

type SystemMemInfo struct {
	Total uint64 `json:"t"`
	Used  uint64 `json:"u"`
	Free  uint64 `json:"f"`
	Cache uint64 `json:"c"`
	Buff  uint64 `json:"b"`
}

func MemStat() (*SystemMemInfo, error) {
	var tmp = &SystemMemInfo{}
	contents, err := ioutil.ReadFile("/proc/meminfo")
	if err != nil {
		return &SystemMemInfo{}, err
	}

	lines := bytes.SplitN(contents, []byte("\n"), 5)
	if err != nil {
		return &SystemMemInfo{}, err
	}

	for _, line := range lines {
		key := strings.Split(string(line), ":")[0]
		num := ByteSplitOnNumber(line)
		switch key {
		case "MemTotal":
			tmp.Total = uint64(num)
		case "MemFree":
			tmp.Free = uint64(num)
		case "Buffers":
			tmp.Buff = uint64(num)
		case "Cached":
			tmp.Cache = uint64(num)
		}
	}
	tmp.Used = tmp.Total - tmp.Free - tmp.Buff - tmp.Cache
	return tmp, nil
}

func ByteSplitOnNumber(data []byte) int64 {
	var bts []byte
	for i, r := range data {
		if r >= 48 && r <= 57 {
			data = data[i:]
			break
		}
	}
	for _, r := range data {
		if r >= 48 && r <= 57 {
			bts = append(bts, r)
		} else {
			break
		}
	}
	if len(bts) > 0 {
		num, _ := strconv.ParseInt(string(bts), 10, 64)
		return num
	}
	return 0
}
