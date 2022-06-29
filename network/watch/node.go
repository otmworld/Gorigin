package watch

import (
	"fmt"
	"sync"
	"time"

	"micro/common"
	"micro/monitor"
	"micro/network/comm"
	"micro/network/pb"
)

type nodemap struct {
	// server name
	// map[string][]uuid
	name sync.Map

	// node uuid
	// map[string]*NodeMsg
	uuid sync.Map
}

type NodeMsg struct {
	base  *pb.NodeInfo
	syst  *pb.SystemStatus
	stamp int64 // heartbeat ping last timestamp
	state bool  // node status, false is cannot connection
}

// update connect status with heartbeat request
func (n *nodemap) UpHeartbeat(sys *pb.SystemStatus) bool {
	if v, ok := n.uuid.Load(sys.GetUuid()); ok && v != nil {
		if msg, ok := v.(*NodeMsg); ok && msg != nil {
			msg.stamp = time.Now().UnixMilli()
			msg.state, msg.syst = true, sys
			return true
		}
	}
	return false
}

// Get node list by uuid
func (n *nodemap) GetNodeByUuid(uuid string) *pb.NodeInfo {
	if uuid != "" {
		if v, ok := n.uuid.Load(uuid); ok && v != nil {
			if msg, ok := v.(*NodeMsg); ok && msg != nil {
				return msg.base
			}
		}
	}
	return nil
}

// Get node list by server name
func (n *nodemap) GetUuidByName(name string) []string {
	if v, ok := n.name.Load(name); ok && v != nil {
		if ids, ok := v.([]string); ok {
			return ids
		}
	}
	return nil
}

// Get node list by server name
func (n *nodemap) GetNodeByName(name string) []*pb.NodeInfo {
	var result []*pb.NodeInfo
	if ids := n.GetUuidByName(name); len(ids) > 0 {
		for _, id := range ids {
			if tmp := n.GetNodeByUuid(id); tmp != nil {
				result = append(result, tmp)
			}
		}
	}
	return result
}

// func (n *WatchDetail) GetFuncNode(fid uint32, sname string) []*pb.NodeInfo {
// 	var result []*pb.NodeInfo
// 	var now = time.Now().Add(comm.NodeConnTimeOut).UnixMilli()
// 	if node, ok := n.serv.Load(sname); ok {
// 		tmp := node.(*ServNode)
// 		for _, row := range tmp.list {
// 			if !row.state || row.stamp < now {
// 				continue
// 			}
// 			if fid != 0 {
// 				for _, f := range row.base.Funcs {
// 					if f.ID == fid {
// 						result = append(result, row.base)
// 						break
// 					}
// 				}
// 			} else {
// 				result = append(result, row.base)
// 			}
// 		}
// 	}
// 	return result
// }

// // Get node list by server name
// func (n *WatchDetail) GetNodeByFuncName(sid, sname string, fid uint32) []*pb.NodeInfo {
// 	var result []*pb.NodeInfo
// 	var now = time.Now().Add(comm.NodeConnTimeOut).UnixMilli()
// 	if sid != "" {
// 		if node, ok := n.uuid.Load(sid); ok {
// 			tmp, ok := node.(*NodeMsg)
// 			if ok && tmp.state && tmp.stamp > now {
// 				if fid != 0 {
// 					for _, f := range tmp.base.Funcs {
// 						if f.ID == fid {
// 							return []*pb.NodeInfo{tmp.base}
// 						}
// 					}
// 				} else {
// 					return []*pb.NodeInfo{tmp.base}
// 				}
// 			}
// 		}
// 	} else if node, ok := n.serv.Load(sname); ok {
// 		tmp := node.(*ServNode)
// 		for _, row := range tmp.list {
// 			if !row.state || row.stamp < now {
// 				continue
// 			}
// 			if fid != 0 {
// 				for _, f := range row.base.Funcs {
// 					if f.ID == fid {
// 						result = append(result, row.base)
// 						break
// 					}
// 				}
// 			} else {
// 				result = append(result, row.base)
// 			}
// 		}
// 	}
// 	return result
// }

// if node exsit, return false
func (n *nodemap) PutNodeDetail(msg *pb.NodeInfo) {
	if v, ok := n.uuid.Load(msg.Uuid); ok && v != nil {
		if arg, ok := v.(*NodeMsg); ok && arg.base.Host == msg.Host {
			arg.stamp, arg.state = time.Now().UnixMilli(), true
		}
		return
	}

	msg.Funcs = nil
	var tmp = &NodeMsg{
		base: msg, state: true,
		stamp: time.Now().UnixMilli(),
	}
	n.uuid.Store(msg.Uuid, tmp)

	// var keys = make(map[string]bool)
	// for _, row := range msg.Funcs {
	// 	sname, _ := comm.SplitApiName(row.Name)
	// 	keys[sname] = true
	// }

	// now := time.Now().Add(comm.NodeConnTimeOut).UnixMilli()

	// for k := range keys {
	// 	go func(key string, tmp *NodeMsg) {
	// 		if v, ok := n.serv.Load(key); ok {
	// 			arg := v.(*ServNode)

	// 			arg.lock.Lock()
	// 			var result = []*NodeMsg{tmp}
	// 			var notify = &pb.GetApiConnRsp{
	// 				List: []*pb.NodeInfo{tmp.base},
	// 				Func: &pb.FuncMsg{ServName: key}}
	// 			for _, row := range arg.list {
	// 				if row.state && row.stamp > now {
	// 					notify.List = append(notify.List, row.base)
	// 					result = append(result, row)
	// 				}
	// 			}
	// 			arg.list = result
	// 			arg.lock.Unlock()

	// 			if bts, err := proto.Marshal(notify); err == nil {
	// 				var used []*NodeMsg
	// 				for _, row := range arg.used {
	// 					if row.state && row.stamp > now {
	// 						used = append(used, row)
	// 						go n.call.WatchSend(time.Second*3,
	// 							row.base.Uuid, comm.UpNodeConnMsg, bts)
	// 					}
	// 				}
	// 				arg.used = used
	// 			}
	// 		} else {
	// 			var data = &ServNode{list: []*NodeMsg{tmp}}
	// 			n.serv.Store(key, data)
	// 		}
	// 	}(k, tmp)
	// }
}

func (w *nodemap) ClearNodeExprie() {
	var v *NodeMsg
	exp := time.Now().Add(comm.NodeConnTimeOut).UnixMilli()
	w.uuid.Range(func(key, value interface{}) bool {
		v = value.(*NodeMsg)
		if !v.state || v.stamp < exp {
			v.state = false
			w.uuid.Delete(key)
		}
		return true
	})

	// w.serv.Range(func(key, value interface{}) bool {
	// 	tmp, ok := value.(*ServNode)
	// 	if ok {
	// 		for _, v = range tmp.list {
	// 			if !v.state || v.stamp < exp {
	// 				goto DealWith
	// 			}
	// 		}
	// 	} else {
	// 		w.serv.Delete(key)
	// 	}
	// 	return true

	// DealWith:
	// 	go func(n *WatchDetail, data *ServNode) {
	// 		var dur = time.Now().Add(comm.NodeConnTimeOut).UnixMilli()
	// 		var nodes []*NodeMsg
	// 		data.lock.Lock()
	// 		var notify = &pb.GetApiConnRsp{Func: &pb.FuncMsg{
	// 			ServName: key.(string), Protocal: pb.Compiler_PROTO}}
	// 		for _, v = range data.list {
	// 			if v.state && v.stamp > dur {
	// 				notify.List = append(notify.List, v.base)
	// 				nodes = append(nodes, v)
	// 			}
	// 		}
	// 		data.list = nodes
	// 		data.lock.Unlock()

	// 		if len(nodes) == 0 {
	// 			n.serv.Delete(key)
	// 			return
	// 		}

	// 		nodes = make([]*NodeMsg, 0)
	// 		if bts, err := proto.Marshal(notify); err == nil {
	// 			for _, row := range data.used {
	// 				if row.state && row.stamp > dur {
	// 					if n.call.WatchSend(time.Second, row.base.Uuid,
	// 						comm.UpNodeConnMsg, bts) == nil {
	// 						nodes = append(nodes, row)
	// 					}
	// 				}
	// 			}
	// 		}
	// 		data.used = nodes
	// 	}(w, tmp)
	// 	return true
	// })
}

func (n *WatchDetail) SystemState() *pb.SystemStatus {
	var result = &pb.SystemStatus{Uuid: n.base.Uuid}
	if cpu, err := monitor.CpuStat(); err == nil {
		result.CpuRate = cpu.Rate
	}
	if mem, err := monitor.MemStat(); err == nil {
		result.MemFree = mem.Free
		result.MemUsed = mem.Used
	}
	cmdStr := fmt.Sprintf("ls /proc/%d/fd | wc -l", n.base.Pid)
	bts, err := common.CmdRunOutput(cmdStr)
	if err == nil && len(bts) > 0 {
		result.ConnNum = uint64(monitor.ByteSplitOnNumber(bts))
	}
	return result
}
