package rpc

import (
	"errors"
	"log"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestEncode(t *testing.T) {
	v1 := "dda测试@#)(&#*$!_))&$#*%_\ndfe大赛的发"
	v2 := "asdf&"
	a := ""
	b := ""
	for i := 0; i < 250; i++ {
		a += v1
		b += v2
	}
	log.Println(len(a), len(b))

	r := &NodeConn{
		fc:   make(map[string]bool),
		rc:   make(map[int]*RecvChan),
		list: make(map[int]*ReadLink),
	}

	var fid = 321
	now := time.Now()
	var sum uint32

	var wg sync.WaitGroup
	for n := 100; n < 1100; n++ {
		wg.Add(1)
		go func(g *sync.WaitGroup, rc *NodeConn, num int) {
			defer g.Done()

			body := []byte(a)[num:]
			// rows := udpsplit.MakeReqBody([]byte(a), num, fid)
			rows := udpsplit.MakeRspBody(body, num, fid, nil)
			for _, row := range rows {
				var buff = &ConnBody{Data: row}
				num1, fid1, bts, err := rc.ParseResp(udpsplit, buff)
				if err == nil && num1 > 0 && fid1 > 0 {
					if num != num1 || fid != fid1 || string(body) != string(bts) {
						log.Println(num1, fid1, len(bts))
						log.Println("make body not equal parse", num)
					} else {
						atomic.AddUint32(&sum, 1)
						// log.Println("Success One", num, len(bts))
					}
				} else if err != nil {
					log.Println("Error: ", num, err)
				}
			}
		}(&wg, r, n)
	}
	wg.Wait()
	log.Println(time.Since(now), sum)

	num, fid := 567, 765
	// rows = udpsplit.MakeReqBody([]byte(b), num, fid)
	rows := udpsplit.MakeRspBody([]byte(b), num, fid, nil)

	for _, row := range rows {
		var buff = &ConnBody{Data: row}
		num2, fid2, bts, err := r.ParseResp(udpsplit, buff)
		if err == nil && num2 > 0 && fid2 > 0 {
			if num != num2 || fid != fid2 || b != string(bts) {
				t.Error("make body not equal parse")
			} else {
				log.Println("Success Two")
			}
		}
	}

	num, fid = 789, 987
	now = time.Now()
	rows = udpsplit.MakeReqBody([]byte(v1), num, fid)
	// rows = udpsplit.MakeRspBody([]byte(v1), num, fid, nil)

	for _, row := range rows {
		var buff = &ConnBody{Data: row}
		num3, fid3, bts, err := r.ParseResp(udpsplit, buff)
		if err == nil && num3 > 0 && fid3 > 0 {
			if num != num3 || fid != fid3 || v1 != string(bts) {
				t.Error("make body not equal parse")
			} else {
				log.Println("Success Three")
			}
		}
	}
	log.Println(time.Since(now))

	num, fid = 789, 987
	now = time.Now()
	rows = udpsplit.MakeRspBody(nil, num, fid, errors.New(v1))

	for _, row := range rows {
		var buff = &ConnBody{Data: row}
		num4, fid4, _, err := r.ParseResp(udpsplit, buff)
		if num != num4 || fid != fid4 || v1 != err.Error() {
			t.Error("make body not equal parse")
		} else {
			log.Println("Success Four")
		}
	}
	log.Println(time.Since(now))

	log.Println(len(r.fc), len(r.rc), len(r.list))
}
