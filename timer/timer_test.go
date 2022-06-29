package timer

import (
	"fmt"
	"log"
	"testing"
	"time"
)

func TestTimer(t *testing.T) {
	timer := NewTimer(time.Millisecond * 500)
	timer.SetTimeZone("+00:01")
	// time.Sleep(time.Second)

	fmt.Println(time.Now(), time.Now().UTC().Unix())

	err, num := timer.AddTimerRun("00", 4, false, "time [:00] one", display)
	if err != nil {
		t.Error(err)
	} else {
		log.Println(num)
	}

	timer.AddDurationFunction(time.Second*2, 3, func() {})

	time.Sleep(time.Minute * 1)
	timer.CloseAndExit()
	time.Sleep(time.Second * 66)
}

func display(data interface{}) {
	fmt.Println(time.Now(), data)
}

func eachprint(data interface{}) {
	fmt.Println(time.Now(), "each second to display")
}

func oneprint() {
	fmt.Println(time.Now(), "one second last display")
}

func twoprint(data interface{}) {
	fmt.Println(time.Now(), "two second last display")
}

func TestSort(t *testing.T) {
	var rows = make([]*TimerFunc, 0, 3)
	rows = append(rows, &TimerFunc{next: 1})

	fmt.Println(len(rows), rows)
}
