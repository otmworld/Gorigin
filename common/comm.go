package common

import (
	"bytes"
	"os/exec"
	"strings"
)

// Cmd Run
func CmdRunOutput(str string) ([]byte, error) {
	return exec.Command("bash", "-c", str).CombinedOutput()
}

func CmdRunEnd(str string) error {
	return exec.Command("bash", "-c", str).Run()
}

func CmdSendRun(str string) {
	exec.Command("bash", "-c", str).Run()
}

func ByteSplitContain(bts, split, contain []byte) [][]byte {
	rows := bytes.Split(bts, split)
	var result [][]byte
	for _, row := range rows {
		if len(row) == 0 {
			continue
		}
		if bytes.Contains(row, contain) {
			result = append(result, row)
		}
	}
	return result
}

func StrSplitContain(str, split, contain string) []string {
	rows := strings.Split(str, split)
	var result []string
	for _, row := range rows {
		if row == "" {
			continue
		}
		if strings.Contains(row, contain) {
			result = append(result, row)
		}
	}
	return result
}

func StrSplitUnContain(str, split, contain string) []string {
	rows := strings.Split(str, split)
	var result []string
	for _, row := range rows {
		if len(row) == 0 {
			continue
		}
		if !strings.Contains(row, contain) {
			result = append(result, row)
		}
	}
	return result
}

func StrSplitUnNumber(str, split string) []string {
	rows := strings.Split(str, split)
	var result []string
	for _, row := range rows {
		if len(row) == 0 {
			continue
		}
		for _, r := range row {
			if r >= 48 && r <= 57 {
				goto NextRow
			}
		}
		result = append(result, row)
	NextRow:
	}
	return result
}
