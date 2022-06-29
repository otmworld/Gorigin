package watch

import (
	"errors"
	"time"

	"micro/common"
	"micro/network"
	"micro/network/comm"
	"micro/network/pb"
	"micro/network/rpc"
	"micro/timer"
)

type WatchDetail struct {
	base  *pb.NodeInfo
	watch []*WatchNode
	elect *WatchElect
	call  *rpc.NodeDetail
	timer timer.TimerStruct
}

type WatchNode struct {
	pb.NodeInfo
	center bool  // 是否为主节点
	status bool  // 状态
	stamp  int64 // 时间戳
}

// func (w *WatchDetail) RangeNodes(function func(*NodeMsg) bool) {
// 	w.uuid.Range(func(key, value interface{}) bool {
// 		if key == nil || value == nil {
// 			w.uuid.Delete(key)
// 		} else if v, ok := value.(*NodeMsg); ok {
// 			return function(v)
// 		} else {
// 			w.uuid.Delete(key)
// 		}
// 		return true
// 	})
// }

func NewWatcher(conf *network.WatcherConfig) error {
	var err error
	if conf.TcpPort == 0 && conf.UdpPort == 0 && conf.HttpPort == 0 {
		return errors.New("cannot all listen port is null")
	} else if conf.Host == "" || conf.Host == "127.0.0.1" {
		if conf.Host, err = common.GetLocalIp(); err != nil {
			return err
		}
	}

	if node, err := rpc.WatchClient(&network.NodeConfig{
		NodeName: comm.WatchNodeName,
		IPAddr:   conf.Host,
		TcpPort:  uint64(conf.TcpPort),
		UdpPort:  uint64(conf.UdpPort),
		HttpPort: uint64(conf.HttpPort),
	}); err != nil || node == nil {
		return err
	} else {
		var nodedata = &WatchApi{
			fmsg: &funcmap{num: comm.WATCH_IN_MAX},
			msg: &WatchDetail{
				base:  &node.NodeInfo,
				elect: &WatchElect{center: true},
				timer: node.GetTimeTicker(),
				watch: []*WatchNode{&WatchNode{
					NodeInfo: node.GetNodeMsg(),
					center:   true,
					stamp:    time.Now().UnixMilli(),
				}},
			},
			node: &nodemap{},
		}
		nodedata.fmsg.InitEnvFile(conf.ConfigPath)

		if err = node.Register(nodedata); err != nil {
			return err
		} else if _, err = node.RunServer(); err != nil {
			return err
		} else {
			nodedata.msg.call = node
		}

		nodedata.msg.timer.AddDurationFunction(time.Second*5, -1, func() {
			nodedata.node.ClearNodeExprie()
		})
	}
	return nil
}
