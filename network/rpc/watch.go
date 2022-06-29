package rpc

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"micro/common"
	"micro/monitor"
	"micro/network/comm"
	"micro/network/pb"

	"google.golang.org/protobuf/proto"
)

type WatchBuiltApi interface {
	Heartbeats() error
	GetFuncMsg(ctx context.Context, fid uint32, name string) (*pb.GetFuncMsgRsp, error)
	GetApiConn(ctx context.Context, fid uint32, name string) (*pb.GetApiConnRsp, error)
	GetNodeMsg(ctx context.Context, uuid string, name string) (*pb.GetNodeMsgRsp, error)
}

func (n *NodeDetail) WatchApi() WatchBuiltApi { return n.wser }

type WatchNode struct {
	uuid   string // server node uuid
	count  uint32 // request watcher number
	procid uint64 // proc id

	config []*pb.NodeInfo // watcher node config
	master *NodeConn      // master watcher node
	slaves []*NodeConn    // slave watcher nodes
}

func (n *NodeDetail) InitWatchConfig() error {
	if n.wser.master != nil {
		n.wser.master.Close()
		n.wser.master = nil
	}
	for _, row := range n.wser.slaves {
		row.Close()
	}

	var err error
	if len(n.wser.config) == 1 {
		n.wser.master, err = n.NodeBaseToConn(n.wser.config[0])
	} else if len(n.wser.config) > 1 {
		n.wser.slaves = make([]*NodeConn, 0)
		for _, node := range n.wser.config {
			conn, err := n.NodeBaseToConn(node)
			if err == nil && conn.TestTcpConn() == nil {
				if node.Main && n.wser.master == nil {
					n.wser.master = conn
				} else {
					n.wser.slaves = append(n.wser.slaves, conn)
				}
			}
		}
		if n.wser.master == nil && len(n.wser.slaves) > 0 {
			n.wser.master = n.wser.slaves[0]
			n.wser.slaves = n.wser.slaves[1:]
		}
	}
	if err != nil || n.wser.master == nil {
		return errors.New("no watcher node can connection: " + fmt.Sprint(err))
	}
	return nil
}

func (n *NodeDetail) WatchRegister(ctx context.Context) error {
	var result = &pb.RegisteredRsp{}
	if bts, err := proto.Marshal(&n.NodeInfo); err != nil {
		return err
	} else {
		err = n.wser.MasterCall(ctx, comm.Registered, bts, result)
		if err != nil || len(result.GetWatch()) <= 0 {
			return errors.New("register watcher wrong: " + fmt.Sprint(err))
		}
	}
	log.Println(result)

	// Init function-id mapping by response data
	for _, fmsg := range result.Funcs {
		sname, fname := comm.SplitApiName(fmsg.Name)
		var tmp = &funcdata{one: true, msg: &pb.FuncMsg{
			FuncID: fmsg.ID, FuncName: fname,
			ServName: sname, ApiName: fmsg.Name,
			ApiType: fmsg.Type, Protocal: fmsg.Kind,
		}}
		n.fmsg.ids.Store(fmsg.ID, tmp)
		n.fmsg.str.Store(fmsg.Name, tmp)
	}

	// Init watchers server list
	for _, node := range result.Watch {
		if conn, err := n.NodeBaseToConn(node); err == nil {
			if node.Main {
				n.wser.master = conn
			} else {
				n.wser.slaves = append(n.wser.slaves, conn)
			}
		}
	}
	if n.wser.master == nil && len(n.wser.slaves) > 0 {
		n.wser.master = n.wser.slaves[0]
		n.wser.slaves = n.wser.slaves[1:]
	}
	n.wser.uuid = n.Uuid
	n.wser.procid = n.Pid
	return nil
}

// send hearbeats to watcher node
func (w *WatchNode) Heartbeats() error {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()
	if bts, err := proto.Marshal(w.SystemState()); err == nil {
		return w.MasterCall(ctx, comm.Heartbeats, bts, nil)
	}
	return w.MasterCall(ctx, comm.Heartbeats, []byte("PING"), nil)
}

// query function message by function-id or api-name
func (w *WatchNode) GetFuncMsg(ctx context.Context, fid uint32, name string) (*pb.GetFuncMsgRsp, error) {
	var result = &pb.GetFuncMsgRsp{}
	bts, err := proto.Marshal(&pb.GetFuncMsgReq{FuncID: fid, ApiName: name})
	if err != nil {
		return result, err
	}
	return result, w.MasterCall(ctx, comm.GetFuncMsg, bts, result)
}

// query node message by function-id or api-name
func (w *WatchNode) GetNodeMsg(ctx context.Context, uuid string, name string) (*pb.GetNodeMsgRsp, error) {
	var result = &pb.GetNodeMsgRsp{Data: &pb.NodeInfo{}}
	bts, err := proto.Marshal(&pb.GetNodeMsgReq{Uuid: uuid, Name: name})
	if err != nil {
		return result, err
	}
	return result, w.MasterCall(ctx, comm.GetNodeMsg, bts, result)
}

// request watcher server api
func (w *WatchNode) GetApiConn(ctx context.Context, fid uint32, name string) (*pb.GetApiConnRsp, error) {
	var result = &pb.GetApiConnRsp{}
	bts, err := proto.Marshal(&pb.GetApiConnReq{FuncID: fid, ApiName: name})
	if err != nil {
		return result, err
	}
	return result, w.MasterCall(ctx, comm.GetApiConn, bts, result)
}

// request watcher by master node
func (w *WatchNode) MasterCall(ctx context.Context, fid int, bts []byte, rsp interface{}) error {
	if w.master == nil || (w.master.tconn == nil && w.master.uconn == nil) {
		return w.SlavesCall(ctx, fid, bts, rsp)
	}
	var num = int(atomic.AddUint32(&w.count, 1) % 65535)
	switch w.master.types {
	case ConnWithTCP:
		var tcpRow = tcpsplit.MakeReqBody(bts, num, fid)
		for _, row := range tcpRow {
			if _, err := w.master.tconn.Write(row); err != nil {
				return w.SlavesCall(ctx, fid, bts, rsp)
			}
		}
		return w.master.ProtoWaitRsp(ctx, num, rsp)

	case ConnWithUDP:
		var udpRow = udpsplit.MakeReqBody(bts, num, fid)
		for _, row := range udpRow {
			if _, err := w.master.uconn.Write(row); err != nil {
				return w.SlavesCall(ctx, fid, bts, rsp)
			}
		}
		return w.master.ProtoWaitRsp(ctx, num, rsp)
	}
	return w.SlavesCall(ctx, fid, bts, rsp)
}

// request watcher by slave node
func (w *WatchNode) SlavesCall(ctx context.Context, fid int, bts []byte, rsp interface{}) error {
	var num = int(atomic.AddUint32(&w.count, 1) % 65535)
	for _, wser := range w.slaves {
		switch wser.types {
		case ConnWithTCP:
			var tcpRow = tcpsplit.MakeReqBody(bts, num, fid)
			for _, row := range tcpRow {
				if _, err := wser.tconn.Write(row); err != nil {
					goto NextNode
				}
			}
			return wser.ProtoWaitRsp(ctx, num, rsp)

		case ConnWithUDP:
			var udpRow = udpsplit.MakeReqBody(bts, num, fid)
			for _, row := range udpRow {
				if _, err := wser.uconn.Write(row); err != nil {
					goto NextNode
				}
			}
			return wser.ProtoWaitRsp(ctx, num, rsp)
		}
	NextNode:
	}
	return errors.New("request watchers wrong")
}

// wait proto protocal response
func (n *NodeConn) ProtoWaitRsp(ctx context.Context, num int, rsp interface{}) error {
	var c = n.NewChan(num)
	defer n.DelChan(num)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case buff := <-c.body:
		if len(buff) == 0 || rsp == nil {
			return nil
		}
		if v, ok := rsp.(proto.Message); ok {
			return proto.Unmarshal(buff, v)
		} else {
			return errors.New("not implement proto.Message")
		}
	case err := <-c.err:
		return err
	}
}

// gen system env detail
func (w *WatchNode) SystemState() *pb.SystemStatus {
	var result = &pb.SystemStatus{Uuid: w.uuid}
	// if cpu, err := monitor.CpuStat(); err == nil && cpu != nil {
	// 	result.CpuRate = cpu.Rate
	// }
	if mem, err := monitor.MemStat(); err == nil && mem != nil {
		result.MemFree = mem.Free
		result.MemUsed = mem.Used
	}
	cmdStr := fmt.Sprintf("ls /proc/%d/fd | wc -l", w.procid)
	bts, err := common.CmdRunOutput(cmdStr)
	if err == nil && len(bts) > 0 {
		result.ConnNum = uint64(monitor.ByteSplitOnNumber(bts))
	}
	return result
}
