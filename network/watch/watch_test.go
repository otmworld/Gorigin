package watch

import (
	"micro/network"
	"micro/network/comm"
	"micro/network/pb"
	"micro/network/rpc"
	"testing"
	"time"
)

func TestWatch(t *testing.T) {
	wd := &WatchDetail{base: &pb.NodeBaseMsg{
		Name: comm.WatchNodeName,
		Host: "192.168.0.175", Tport: 8080, Hport: 8082,
	}, node: &NodeMsgList{
		name: make(map[string][]*NodeMsg),
		uuid: make(map[string]*NodeMsg),
	}, fmsg: &FuncMsgList{
		num: comm.WATCH_IN_MAX},
		elect: &WatchElect{}}
	for _, row := range comm.WatchFmsg.Fmsg {
		wd.base.Funcs = append(wd.base.Funcs, row)
	}
	wd.watch = append(wd.watch, &WatchNode{
		NodeBaseMsg: *wd.base,
		center:      true, stamp: time.Now().UnixMilli(),
	})

	node, err := rpc.NewClient(&network.NodeConfig{
		NodeName: comm.WatchNodeName,
		IPAddr:   wd.base.Host,
		TcpPort:  uint64(wd.base.Tport),
		HttpPort: uint64(wd.base.Hport),
	})
	if err != nil {
		t.Error(err)
	}
	wd.fmsg.InitEnvFile("./")

	if err = node.Register(&WatchApi{msg: wd}); err != nil {
		t.Error(err)
	}
	api, err := node.RunServer()
	if err != nil {
		t.Error(err)
	}
	wd.call = api

	time.Sleep(time.Minute * 10)
}
