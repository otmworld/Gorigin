package rpc

import (
	"errors"
	"net"
	"strings"
)

// get not used port
func GetFreePort() (uint64, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return uint64(l.Addr().(*net.TCPAddr).Port), nil
}

// check input string
func IsExportedName(name string) error {
	for _, n := range name {
		switch {
		case n >= 48 && n <= 58: // '0-9', ':'
		case n == 45, n == 95: // '-', '_'
		case n >= 65 && n <= 90: // 'A-Z'
		case n >= 97 && n <= 122: // 'a-z'
		default:
			return errors.New("have illegal characters")
		}
	}
	return nil
}

// split server name to find api
// return nodename, servername, funcname
func SplitServerName(name string) (string, string, string) {
	rows := strings.Split(name, ".")
	if len(rows) == 3 {
		return rows[0], rows[1], rows[2]
	} else if len(rows) == 2 {
		return "", rows[0], rows[1]
	}
	return "", "", ""
}

// return nodename, apiname
func SplitApiName(name string) (string, string) {
	rows := strings.Split(name, ".")
	if len(rows) == 3 {
		return rows[0], rows[1] + "." + rows[2]
	} else if len(rows) == 2 {
		return "", name
	}
	return "", ""
}

// return nodename, apiname
func SplitServName(name string) string {
	rows := strings.Split(name, ".")
	if len(rows) == 2 {
		return rows[0]
	} else if len(rows) == 3 {
		return rows[1]
	}
	return name
}
