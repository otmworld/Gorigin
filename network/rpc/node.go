package rpc

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"micro/common"
	"micro/network"
	"micro/network/comm"
	"micro/network/pb"
	"micro/timer"

	"google.golang.org/protobuf/proto"
)

// micro server node base struct message
type NodeDetail struct {
	// server node base message
	pb.NodeInfo

	// Local server list
	// string: function name
	funcs map[string]*Server
	// heartbeat send to watcher
	heart time.Duration
	// convert to http request
	hlist map[string]func(http.ResponseWriter, *http.Request)
	// timer to run
	ticker timer.TimerStruct
	// request timers
	reqnum uint32

	// function message and connection
	fmsg *funcmap

	// watcher node detail to connect
	wser *WatchNode

	// dail network to register link
	tmps struct {
		tcpreg [][]byte
		udpreg [][]byte
	}
	run sync.Once
}

func (n *NodeDetail) QueryFunc(fid uint32, name string) *pb.FuncMsg {
	if value := n.fmsg.Query(fid, name); value != nil {
		return value.GetMsg()
	}

	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*3)
	defer cancel()
	rsp, err := n.WatchApi().GetFuncMsg(ctx, fid, name)
	if err == nil && rsp.GetFunc() != nil {
		sname, fname := comm.SplitApiName(rsp.Func.Name)
		var tmp = &pb.FuncMsg{
			FuncID: rsp.Func.ID, FuncName: fname,
			ServName: sname, ApiName: rsp.Func.Name,
			ApiType: rsp.Func.Type, Protocal: rsp.Func.Kind,
		}
		n.fmsg.PutMsg(tmp)
		return tmp
	}
	return nil
}

// Get server function name by id
func (n *NodeDetail) GetFuncName(id uint32) string { return n.QueryFunc(id, "").ApiName }

// Get server function id by name
func (n *NodeDetail) GetFuncID(name string) uint32 { return n.QueryFunc(0, name).FuncID }

func (n *NodeDetail) GetNodeMsg() pb.NodeInfo { return n.NodeInfo }

func WatchClient(config *network.NodeConfig) (*NodeDetail, error) { return newClient(config) }

func newClient(config *network.NodeConfig) (*NodeDetail, error) {
	var result = &NodeDetail{
		NodeInfo: pb.NodeInfo{
			Pid:   uint64(syscall.Getpid()),
			Ver:   config.Version,
			Name:  config.NodeName,
			Tport: config.TcpPort,
			Uport: config.UdpPort,
			Hport: config.HttpPort,
		},
		ticker: timer.NewTimer(time.Millisecond * 200),
		wser:   &WatchNode{},
		fmsg:   &funcmap{},
		funcs:  make(map[string]*Server),
		hlist:  make(map[string]func(http.ResponseWriter, *http.Request)),
	}
	for _, row := range config.Watchers {
		result.wser.config = append(result.wser.config, &pb.NodeInfo{
			Host: row.Host, Tport: row.TcpPort, Uport: row.UdpPort})
	}

	var err error
	if result.Host, err = common.GetLocalIp(); err != nil {
		result.Host = "0.0.0.0"
	}
	pid := result.Host + ":" + strconv.FormatUint(result.Pid, 10)
	h := md5.New()
	h.Write([]byte(pid))
	result.Uuid = fmt.Sprintf("%x", h.Sum(nil))[8:24]

	// make free port to listen
	if !config.TcpListenOff && config.TcpPort == 0 {
		if result.Tport, err = GetFreePort(); err != nil {
			return nil, err
		}
	} else if config.TcpListenOff {
		result.Tport = 0
	}
	if config.UdpListenOn && config.UdpPort == 0 {
		if result.Uport, err = GetFreePort(); err != nil {
			return nil, err
		}
	} else if !config.UdpListenOn {
		result.Uport = 0
	}
	if !config.HttpListenOff && config.HttpPort == 0 {
		if result.Hport, err = GetFreePort(); err != nil {
			return nil, err
		}
	} else if config.HttpListenOff {
		result.Hport = 0
	}

	if config.HeartBeatTime > time.Second &&
		config.HeartBeatTime < comm.NodeConnTimeOut {
		result.heart = config.HeartBeatTime
	} else {
		result.heart = comm.HeartbeatInterval
	}
	if err = result.initMsgByte(); err != nil {
		return result, err
	}

	// make watcher server node connection message to local cache
	if config.NodeName != comm.WatchNodeName {
		if len(config.Watchers) == 0 {
			return nil, errors.New("watcher node cannot be null")
		}
		return result, result.InitWatchConfig()
	}
	return result, nil
}

func (n *NodeDetail) initMsgByte() error {
	if bts, err := proto.Marshal(&n.NodeInfo); err != nil {
		return err
	} else {
		rows := tcpsplit.MakeReqBody(bts, 1, comm.DialRegister)
		if len(rows) < 1 {
			return errors.New("node message have mistake")
		}
		n.tmps.tcpreg = rows

		rows = udpsplit.MakeReqBody(bts, 1, comm.DialRegister)
		if len(rows) < 1 {
			return errors.New("node message have mistake")
		}
		n.tmps.udpreg = rows
	}
	return nil
}

// arge: name, port
func (n *NodeDetail) runServer() error {
	// make tcp listen
	if n.Tport > 0 {
		if err := n.TcpListen(int(n.Tport)); err != nil {
			return fmt.Errorf("tcp listen: %v", err)
		}
	}
	// make udp listen server
	if n.Uport > 0 {
		if err := n.UdpListen(int(n.Uport)); err != nil {
			return fmt.Errorf("udp listen: %v", err)
		}
	}
	// make http listen server
	if n.Hport > 0 {
		go func(nd *NodeDetail) {
			mux := http.NewServeMux()
			for key, function := range nd.hlist {
				mux.HandleFunc("/"+key, function)
			}
			// build-in
			mux.HandleFunc("/"+comm.BUILT_IN_NAME, nd.httpBuiltIn)
			err := http.ListenAndServe(":"+strconv.FormatUint(nd.Hport, 10), mux)
			if err != nil {
				panic(err)
			}
			nd.hlist = nil
		}(n)
	}

	if n.NodeInfo.Name != comm.WatchNodeName {
		ctx, cancel := context.WithTimeout(context.TODO(), time.Second*10)
		defer cancel()
		if err := n.WatchRegister(ctx); err != nil {
			return err
		}
	} else {
		// init watcher built function map
		for _, row := range comm.WatchFmsg.Fmsg {
			n.fmsg.PutMsg(row)
		}
	}
	// init timer and register server node
	n.InitTimer()
	return nil
}

var heartbeatsWrong uint32 = 0

func (n *NodeDetail) InitTimer() {
	n.ticker.AddDurationFunction(time.Second*10, -1, func() {
		n.fmsg.ClearConn()
	})

	// timer make heartbeat to watcher
	if n.Name != comm.WatchNodeName {
		n.ticker.AddDurationFunction(comm.HeartbeatInterval, -1, func() {
			if err := n.WatchApi().Heartbeats(); err != nil {
				if atomic.AddUint32(&heartbeatsWrong, 1)%3 != 0 {
					return
				}
				ctx, cancel := context.WithTimeout(context.TODO(), time.Second*3)
				defer cancel()
				if n.WatchRegister(ctx) != nil {
					if n.InitWatchConfig() == nil {
						if n.WatchRegister(context.TODO()) == nil {
							atomic.SwapUint32(&heartbeatsWrong, 0)
						}
					}
				} else {
					atomic.SwapUint32(&heartbeatsWrong, 0)
				}
			}
		})
	}
}

func (n *NodeDetail) NodeBaseToConn(node *pb.NodeInfo) (*NodeConn, error) {
	// n.ClearConnWithUuid()
	if conn := n.fmsg.GetNodeConn(node.Uuid); conn != nil {
		if !conn.wrong && conn.Host != node.Host {
			return nil, errors.New("same uuid but host not same")
		} else if conn.TestConn() == nil {
			conn.wrong = false
			return conn, nil
		} else if conn != nil {
			conn.wrong = true
			conn.Close()
			n.fmsg.DelConn(node.Uuid)
		}
	}

	var conn = &NodeConn{
		NodeInfo: *node,
		list:     make(map[int]*ReadLink),
		fc:       make(map[string]bool),
		rc:       make(map[int]*RecvChan),
	}
	err := n.RefreshConn(conn)
	if err == nil {
		n.TimerCheck(conn)
		n.fmsg.PutConn(conn)
	}
	return conn, err
}
