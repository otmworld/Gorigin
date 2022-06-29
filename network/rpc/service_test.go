package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"micro/network"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestServerA(t *testing.T) {
	node, err := NewClient(&network.NodeConfig{
		NodeName: "ServerA",
		IPAddr:   "192.168.0.175",
		TcpPort:  8091,
		Watchers: []*network.WatcherConfig{
			{Host: "192.168.0.175", TcpPort: 8080, HttpPort: 8082},
		},
	})
	if err != nil {
		t.Error(err)
	}
	if err = node.Register(&Bsv{}); err != nil {
		t.Error(err)
	}
	api, err := node.RunServer()
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second * 3)

	go func() {
		time.Sleep(time.Second * 2)
		req := &GetNameReq{Name: "Garaad的方式都\n方式士大夫上///n/\\\n看到激发*&(^%&^$%%*&)*)_(df的方式都"}
		rsp := &GetNameRsp{}
		resp := api.CallAuto("Tsv.GetName", req, rsp)
		fmt.Println(resp.Err(), resp.Network(), rsp)

		time.Sleep(time.Second * 1)
		req1 := &GetNameReq{Name: "Lin"}
		rsp1 := &GetNameRsp{}
		resp = api.CallAuto("Tsv.UpName", req1, rsp1)
		fmt.Println(resp.Err(), resp.Network(), rsp1)

		err = api.SendAuto("Tsv.SendName", req1)
		fmt.Println(err)

		msg := "__d&(^%&^$%%*&)*)_(>?<!@#$%$"
		req2 := &GetNameReq{Name: msg}
		rsp2 := &GetNameRsp{}
		resp = api.CallAutoContext(context.TODO(), "Tsv.GetName", req2, rsp2)
		fmt.Println("--1111--", resp.Err(), resp.Network(), rsp2.Name, rsp2.Name == msg)

		req3 := &GetNameReq{Name: "AAA"}
		rsp3 := &GetNameRsp{}
		resp = api.CallMultiAuto("Tsv.MultiName", req3, req3, req3, rsp3)
		fmt.Println("--1111--", resp.Err(), resp.Network(), rsp3.Name)

		req4 := &GetNameReq{Name: "BBB"}
		bts, _ := json.Marshal(req4)

		api.CallMultiByByte(time.Second, "", "Tsv.MultiName", bts, bts, bts)
		time.Sleep(time.Second)

		var now = time.Now()
		var wg sync.WaitGroup
		var wrong uint32
		for i := 0; i < 10000; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				resp1 := api.CallMultiByByte(time.Second, "", "Tsv.MultiName", bts, bts, bts)
				if resp1.Err() != nil {
					fmt.Println(resp1.Err())
					atomic.AddUint32(&wrong, 1)
				}
			}()
		}
		wg.Wait()
		fmt.Println(time.Since(now), wrong)
	}()
	time.Sleep(time.Minute * 5)
}

func TestServerB(t *testing.T) {
	node, err := NewClient(&network.NodeConfig{
		NodeName: "ServerB",
		IPAddr:   "192.168.0.175",
		// TcpPort: 8092,
		// TcpListenOff: true,
		// UdpListenOn:  true,
		// UdpPort:      8093,
		Watchers: []*network.WatcherConfig{
			{Host: "192.168.0.175", TcpPort: 8080, HttpPort: 8082},
		},
	})
	if err != nil {
		t.Error(err)
	}
	if err = node.Register(&Tsv{}); err != nil {
		t.Error(err)
	}
	api, err := node.RunServer()
	if err != nil {
		t.Error(err)
	}

	time.Sleep(time.Second * 3)

	go func() {
		time.Sleep(time.Second * 2)
		req := &GetNameReq{Name: "Garaad的方式都方式士大夫上看到激发*&(^%&^$%%*&)*)_(df的方式都"}
		rsp := &GetNameRsp{}
		resp := api.CallAuto("Bsv.GetName", req, rsp)
		fmt.Println(resp.Err(), resp.Network(), rsp)

		time.Sleep(time.Second * 1)
		req1 := &GetNameReq{Name: "Lin"}
		rsp1 := &GetNameRsp{}
		resp = api.CallAuto("Bsv.UpName", req1, rsp1)
		fmt.Println(resp.Err(), resp.Network(), rsp1)

		err = api.SendAuto("Bsv.SendName", req1)
		fmt.Println(err)

		msg := "__d&(^%&^$%%*&)*)_(>?<!@#$%$"
		req2 := &GetNameReq{Name: msg}
		rsp2 := &GetNameRsp{}
		resp = api.CallAutoContext(context.TODO(), "Bsv.GetName", req2, rsp2)
		fmt.Println("--1111--", resp.Err(), resp.Network(), rsp2.Name, rsp2.Name == msg)

		req3 := &GetNameReq{Name: "AAA"}
		rsp3 := &GetNameRsp{}
		resp = api.CallMultiAuto("Bsv.MultiName", req3, req3, req3, rsp3)
		fmt.Println("--1111--", resp.Err(), resp.Network(), rsp3.Name)

		req4 := &GetNameReq{Name: "BBB"}
		bts, _ := json.Marshal(req4)

		api.CallMultiByByte(time.Second, "", "Bsv.MultiName", bts, bts, bts)
		time.Sleep(time.Second)

		var now = time.Now()
		var wg sync.WaitGroup
		var wrong uint32
		for i := 0; i < 10000; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				resp1 := api.CallMultiByByte(time.Second, "", "Bsv.MultiName", bts, bts, bts)
				if resp1.Err() != nil {
					fmt.Println(resp1.Err())
					atomic.AddUint32(&wrong, 1)
				}
			}()
		}
		wg.Wait()
		fmt.Println(time.Since(now), wrong)
	}()
	time.Sleep(time.Minute * 5)
}

type Tsv struct {
	Name string `json:"name"`
}

type Bsv struct{ Tsv }

type GetNameReq struct {
	Name string `json:"name"`
}

type GetNameRsp struct {
	Name string `json:"name"`
}

func (s *Tsv) GetName(req *GetNameReq, rsp *GetNameRsp) error {
	rsp.Name = "GetName:" + req.Name
	return nil
}

func (s *Tsv) UpName(req *GetNameReq, rsp *GetNameRsp) error {
	rsp.Name = "UpName:" + req.Name
	return nil
}

func (s *Tsv) SendName(req *GetNameReq) error {
	return nil
}

func (s *Tsv) MultiName(req1, req2, req3 *GetNameReq, rsp *GetNameRsp) error {
	rsp.Name = req1.Name + "=====" + req2.Name + "-----" + req3.Name
	return nil
}

func (s *Tsv) Compiler_JSON() {}

func (s *Bsv) Compiler_JSON() {}
