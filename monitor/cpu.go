package monitor

import (
	"errors"
	"fmt"
	"io/ioutil"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	CpuStatUser = iota
	CpuStatNice
	CpuStatSystem
	CpuStatIdle
	CpuStatIowait
	CpuStatIrq
	CpuStatSoftirq
	CpuStatSteal
)

type SystemCpuInfo struct {
	Core int    `json:"c"`
	Rate uint64 `json:"r"` // 千进保存
}

func CpuStat() (*SystemCpuInfo, error) {
	rate, err := CpuRate(time.Millisecond * 100)
	if err != nil {
		return nil, err
	}
	return &SystemCpuInfo{
		Core: runtime.NumCPU(),
		Rate: uint64(rate * 1000),
	}, nil
}

func CpuRate(duration time.Duration) (float64, error) {
	prework, presum, err := checkCpuRun()
	if err != nil {
		return 0, err
	}
	time.Sleep(duration)

	sufwork, sufsum, err := checkCpuRun()
	if err != nil {
		return 0, err
	}
	return float64(sufwork-prework) / float64(sufsum-presum), nil
}

func checkCpuRun() (int, int, error) {
	contents, err := ioutil.ReadFile("/proc/stat")
	if err != nil {
		return 0, 0, errors.New("invalid proc stat file")
	}

	lines := strings.SplitN(string(contents), "\n", 2)
	if len(lines) == 0 {
		return 0, 0, errors.New("invalid statistic file")
	}

	fields := strings.Fields(lines[0])
	if len(fields) != 11 || fields[0] != "cpu" {
		return 0, 0, errors.New("invalid statistic file")
	}

	v := make([]int, len(fields))
	for i := 1; i < len(fields); i++ {
		if v[i], err = strconv.Atoi(fields[i]); err != nil {
			return 0, 0, fmt.Errorf("invalid proc stat file: %v", err)
		}
	}

	work := v[CpuStatUser] + v[CpuStatNice] + v[CpuStatSystem] + v[CpuStatIrq] + +v[CpuStatSoftirq] + v[CpuStatSteal]
	return work, work + v[CpuStatIdle] + v[CpuStatIowait], nil
}
