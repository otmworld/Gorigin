package watch

import (
	"errors"
	"log"

	"micro/network"
	"micro/network/pb"
)

type WatchApi struct {
	// function mapping
	fmsg *funcmap
	// server node mapping
	node *nodemap

	msg *WatchDetail
	api network.NodeApi
}

func (w *WatchApi) Registered(req *pb.NodeInfo, rsp *pb.RegisteredRsp) error {
	defer log.Println("Registered: ", rsp)

	if req.Uuid == "" {
		return errors.New("node server uuid cannot be null")
	}

	// TODO:
	// if w.Center() || len(w.watch) <= 1 {
	for _, v := range req.Funcs {
		if w.fmsg.PutApi(req.Uuid, v) != nil {
			// TODO: 通知所有节点
			// w.api.SendAutoAll(pb.WatchFmsg.GetFuncMsg())
		}
		rsp.Funcs = append(rsp.Funcs, v)
	}
	// rsp.MainUuid = w.base.Uuid
	// }
	w.node.PutNodeDetail(req)

	// go w.msg.RangeNodes(func(node *NodeMsg) bool {
	// 	if !node.state || node.base.Uuid == req.Uuid {
	// 		return true
	// 	}
	// 	if node.base.Host == req.Host {
	// 		if time.Now().Add(comm.NodeConnTimeOut).UnixMilli() > node.stamp {
	// 			node.state = false
	// 		} else if err := w.msg.call.WatchSend(time.Second, node.base.Uuid,
	// 			comm.PingNetwork, []byte("PING")); err != nil {
	// 			node.state = false
	// 		}
	// 	}
	// 	return true
	// })

	for _, row := range w.msg.watch {
		rsp.Watch = append(rsp.Watch, &pb.NodeInfo{
			Uuid: row.Uuid, Main: row.Main,
			Host: row.Host, Tport: row.Tport,
			Uport: row.Uport, Hport: row.Hport,
		})
	}
	return nil
}

// // send data to master watcher node
// func (w WatchApi) NewStrcut(req *pb.NodeInfo) error {
// 	var servname = make(map[string]bool)
// 	for _, fmsg := range req.Funcs {
// 		w.fmsg.AddOrUpdate(fmsg)
// 		servname[fmsg.ServName] = true
// 	}
// 	// TODO: notify other watcher server node
// 	return nil
// }

// server node timer send local message to watcher
func (w *WatchApi) Heartbeats(req *pb.SystemStatus) error {
	log.Println("Heartbeats: ", req)
	if w.node.UpHeartbeat(req) {
		return nil
	}
	return errors.New("no found this node")
}

// get server node message
func (w *WatchApi) GetFuncMsg(req *pb.GetFuncMsgReq, rsp *pb.GetFuncMsgRsp) error {
	log.Println("GetFuncMsg: ", req, rsp)

	if req.GetFuncID() != 0 {
		rsp.Func = w.fmsg.GetIds(req.FuncID).GetApi()
	} else if req.GetApiName() != "" {
		rsp.Func = w.fmsg.GetStr(req.ApiName).GetApi()
	} else {
		w.fmsg.RangeApi(func(arg *pb.FuncApi) bool {
			rsp.List = append(rsp.List, arg)
			return true
		})
	}
	return nil
}

// get server node message
func (w *WatchApi) GetNodeMsg(req *pb.GetNodeMsgReq, rsp *pb.GetNodeMsgRsp) error {
	log.Println("GetNodeMsg: ", req, rsp)

	if req.GetUuid() != "" {
		rsp.Data = w.node.GetNodeByUuid(req.Uuid)
		return nil
	} else if req.GetName() != "" {
		rsp.List = w.node.GetNodeByName(req.Name)
		return nil
	}
	return errors.New("not found this server node")
}

// get server address array by name
func (w *WatchApi) GetApiConn(req *pb.GetApiConnReq, rsp *pb.GetApiConnRsp) error {
	defer log.Println("GetApiConn: ", req, rsp)

	if req.GetFuncID() != 0 {
		rsp.Func = w.fmsg.GetIds(req.FuncID).GetMsg()
	} else if req.GetApiName() != "" {
		rsp.Func = w.fmsg.GetStr(req.ApiName).GetMsg()
	}
	if data := w.fmsg.GetIds(rsp.GetFunc().GetFuncID()); data != nil {
		var ids []string
		for _, id := range data.node {
			if tmp := w.node.GetNodeByUuid(id); tmp != nil {
				rsp.List = append(rsp.List, tmp)
				ids = append(ids, id)
			}
		}
		data.node = ids
	}
	return nil
}
