package watch

import (
	"errors"
	"sync"
	"time"

	"micro/network/pb"
)

type WatchElect struct {
	center bool   // 是否为主节点，中心节点
	begin  int64  // notify send timestamp
	exprie int64  // end main elect timestamp
	nodeId string // 发起的节点编号
	list   []*ElectCampRsp
}

type ElectCampRsp struct {
	NodeBase  *pb.NodeInfo
	SystemMsg *pb.SystemStatus
}

func (w *WatchDetail) Center() bool {
	return w.elect.center
}

type ElectCampReq struct {
	Base   *pb.NodeInfo
	Uuids  []string
	Begin  int64
	Exprie int64
}

func (w *WatchDetail) Elect() {
	now := time.Now().UnixMilli()
	if w.elect.exprie > now {
		return
	}

	w.elect.center, w.elect.nodeId = false, w.base.Uuid
	w.elect.begin, w.elect.exprie = now, now+10_000
	var tmp = &ElectCampReq{
		Base: w.base, Begin: now, Exprie: w.elect.exprie}

	var wg sync.WaitGroup
	for _, n := range w.watch {
		if n.status {
			tmp.Uuids = append(tmp.Uuids, n.Uuid)
			wg.Add(1)
			go func() {
				defer wg.Done()
				var rsp = &ElectCampRsp{}
				if w.call.CallByUuid(n.Uuid, "WatchApi.ElectNotify", tmp, rsp) == nil {
					w.elect.list = append(w.elect.list, rsp)
				}
			}()
		}
	}
	wg.Wait()
}

// 接收主节点推荐竞选通知，并返回服务器状态
func (w WatchApi) ElectNotify(req *ElectCampReq, rsp *ElectCampRsp) error {
	now := time.Now().UnixMilli()
	msg := w.msg.elect
	if msg.exprie > now && msg.begin < req.Begin {
		return errors.New("have other campaign")
	}

	msg.center, msg.nodeId = false, req.Base.Uuid
	msg.begin, msg.exprie = req.Begin, req.Exprie
	rsp.NodeBase = w.msg.base
	rsp.SystemMsg = w.msg.SystemState()
	return nil
}

// 更新本地发现节点数据
func (w WatchApi) ElectResult(req *pb.NodeInfo) error {

	return nil
}
