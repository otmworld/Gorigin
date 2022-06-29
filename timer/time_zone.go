package timer

import (
	"errors"
	"strconv"
	"strings"
)

// SetTimeZone : verify time eg:[-08:00, +08:00]
// If want use UTC to time zone : '+00:00'
func (t *TimerData) SetTimeZone(v string) error {
	if v == "" {
		return errors.New("time zone string is null")
	} else if v[0] != 43 && v[0] != 45 {
		return errors.New("time zone string first byte not ['-' or '+']")
	} else if v == "+00:00" {
		t.zone = 0
		return nil
	}

	ts := strings.Split(v[1:], ":")
	if len(ts) != 2 {
		return errors.New("time zone string format wrong")
	}

	h, err := strconv.ParseUint(ts[0], 10, 64)
	if err != nil {
		return errors.New("parse time zone hour error: " + err.Error())
	} else if h > 23 {
		return errors.New("time zone hour more than 23")
	}

	m, err := strconv.ParseUint(ts[1], 10, 64)
	if err != nil {
		return errors.New("parse time Zone minter error: " + err.Error())
	} else if m > 59 {
		return errors.New("time zone minter more than 59")
	}

	if v[0] == 43 {
		t.zone = int64(h*3600 + m*60)
	} else {
		t.zone = -int64(h*3600 + m*60)
	}
	return nil
}
