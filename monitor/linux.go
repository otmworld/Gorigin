package monitor

import (
	"errors"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Server top process running detail
type TopProc struct {
	Pid  int    `json:"p"`
	Cpu  string `json:"c"`
	Mem  string `json:"m"`
	Cmd  string `json:"d"`
	Stat string `json:"s"`
}

// Server Status with: [menory, storage and top process]
type ServerState struct {
	Time int64          `json:"ts"` // second
	Top  []*TopProc     `json:"top"`
	Cpu  *SystemCpuInfo `json:"cpu"`
	Mem  *SystemMemInfo `json:"mem"`
}

// Check Linux Server Status
func SystemStat() (*ServerState, error) {
	if runtime.GOOS != "linux" {
		return &ServerState{}, errors.New("not linux system")
	}
	var err error
	var tmp = &ServerState{
		Time: time.Now().UTC().Unix(),
	}
	if tmp.Top, err = ShellTop(); err != nil {
		return tmp, err
	}
	if tmp.Mem, err = MemStat(); err != nil {
		return tmp, err
	}
	if tmp.Cpu, err = CpuStat(); err != nil {
		return tmp, err
	}
	return tmp, nil
}

const topCmdStr = `top -bn 1 | head -n 17 | tail -10 | awk '{print $1,$8,$9,$10,$12}'`

// ShellTop Find Top process running
func ShellTop() ([]*TopProc, error) {
	out, err := exec.Command("sh", "-c", topCmdStr).CombinedOutput()
	if err != nil {
		return nil, err
	}
	rows := strings.Split(string(out), "\n")

	var result = make([]*TopProc, 0, len(rows))
	for _, row := range rows {
		v := strings.Split(row, " ")
		if len(v) != 5 {
			continue
		}
		if v[0] == "" || v[4] == "top" {
			continue
		}
		if num, err := strconv.Atoi(v[0]); err != nil || num <= 100 {
			continue
		} else {
			result = append(result, &TopProc{
				Pid:  num,
				Cpu:  v[2],
				Mem:  v[3],
				Cmd:  v[4],
				Stat: v[1],
			})
		}
	}
	return result, nil
}
