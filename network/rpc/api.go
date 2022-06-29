package rpc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"micro/timer"
	"net/http"
	"time"

	"micro/network"
	"micro/network/pb"
)

const (
	CtxKeyNodeUuid = "Node_Uuid"
	CtxKeyNodeBase = "Node_Base"
)

// 当请求的接口是以 One.Two.Three 模式请求全部，依旧查询读取 One.Two.Three
// 当请求的接口是以 Two.Three 模式请求全部，将读取全部 *.Two.Three

func NewClient(config *network.NodeConfig) (network.Node, error) { return newClient(config) }
func (n *NodeDetail) GetTimeTicker() timer.TimerStruct           { return n.ticker }
func (n *NodeDetail) GetNodeDetail() pb.NodeInfo                 { return n.NodeInfo }

func (n *NodeDetail) RunServer() (network.NodeApi, error) {
	var err error
	n.run.Do(func() {
		err = n.runServer()
	})
	return network.NodeApi(n), err
}

func (n *NodeDetail) GetNodeByUuid(uuid string) (pb.NodeInfo, error) {
	// check local uuid
	if n.Uuid == uuid {
		return n.NodeInfo, nil
	}
	// check local remote connection cache
	if v := n.fmsg.GetNodeConn(uuid); v != nil {
		return v.NodeInfo, nil
	}
	// query by watcher
	rsp, err := n.WatchApi().GetNodeMsg(context.TODO(), uuid, "")
	return *rsp.GetData(), err
}

// query server node name list by watcher
func (n *NodeDetail) GetNodeByName(name string) ([]*pb.NodeInfo, error) {
	rsp, err := n.WatchApi().GetNodeMsg(context.TODO(), "", name)
	return rsp.GetList(), err
}

func (n *NodeDetail) CallAuto(name string, req, rsp interface{}) network.CallResp {
	return n.call(context.TODO(), "", name, req, rsp)
}
func (n *NodeDetail) CallByUuid(uuid, name string, req, rsp interface{}) network.CallResp {
	return n.call(context.TODO(), uuid, name, req, rsp)
}
func (n *NodeDetail) CallAutoContext(ctx context.Context, name string, req, rsp interface{}) network.CallResp {
	return n.call(ctx, "", name, req, rsp)
}
func (n *NodeDetail) CallAutoTimeout(duration time.Duration, name string, req, rsp interface{}) network.CallResp {
	ctx, cancel := context.WithTimeout(context.TODO(), duration)
	defer cancel()
	return n.call(ctx, "", name, req, rsp)
}

func (n *NodeDetail) SendAuto(name string, req interface{}) error {
	return n.send(context.TODO(), "", name, req).Err()
}
func (n *NodeDetail) SendByUuid(uuid, name string, req interface{}) error {
	return n.send(context.TODO(), uuid, name, req).Err()
}
func (n *NodeDetail) SendAutoContext(ctx context.Context, name string, req interface{}) error {
	return n.send(ctx, "", name, req).Err()
}
func (n *NodeDetail) SendAutoTimeout(duration time.Duration, name string, req interface{}) error {
	ctx, cancel := context.WithTimeout(context.TODO(), duration)
	defer cancel()
	return n.send(ctx, "", name, req).Err()
}
func (n *NodeDetail) SendAutoAll(name string, req interface{}, rsp *pb.SendAllRsp) error {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*10)
	defer cancel()
	return n.sendall(ctx, name, req, rsp)
}

// last arg is response struct
func (n *NodeDetail) CallMultiAuto(name string, args ...interface{}) network.CallResp {
	return n.multi(context.TODO(), "", name, args...)
}
func (n *NodeDetail) CallMultiByUuid(uuid, name string, args ...interface{}) network.CallResp {
	return n.multi(context.TODO(), uuid, name, args...)
}

//
func (n *NodeDetail) CallRemoteByte(duration time.Duration, uuid, name string, req []byte) network.CallByte {
	_, apiname := SplitApiName(name)
	if apiname == "" {
		return &CallResp{msg: &pb.NodeInfo{}, err: errors.New("name cannot be null")}
	}

	ctx, cancel := context.WithTimeout(context.TODO(), duration)
	defer cancel()

	tmp := n.fmsg.Query(0, apiname)
	if tmp == nil {
		return &CallResp{msg: &pb.NodeInfo{}, con: "Remote",
			err: errors.New("not found function to request: " + apiname)}
	}
	fmsg := tmp.GetMsg()
	conns := n.fmsg.GetFuncConn(0, apiname)

	var remote bool
TryRemote:
	if len(conns) == 0 {
		rsp, err := n.WatchApi().GetApiConn(ctx, fmsg.FuncID, fmsg.ApiName)
		log.Println("1111111111111111111", err, rsp)
		if err == nil && len(rsp.List) > 0 {
			var ids []string
			for _, node := range rsp.List {
				if conn := n.fmsg.GetNodeConn(node.Uuid); conn != nil {
					if conn.TestConn() == nil {
						ids = append(ids, node.Uuid)
						conns = append(conns, conn)
						continue
					}
				}
				if conn, err := n.NodeBaseToConn(node); err == nil {
					n.fmsg.PutConn(conn)
					ids = append(ids, node.Uuid)
					conns = append(conns, conn)
				}
			}
			n.fmsg.UpFuncNode(fmsg.FuncID, ids)
		}
		if len(conns) == 0 {
			return &CallResp{msg: &pb.NodeInfo{}, con: "Remote",
				err: errors.New("not found node to request: " + fmsg.ApiName)}
		}
		remote = true
	}

	var num = n.serial(fmsg.FuncID)
	var tcpRow, udpRow [][]byte
	for _, conn := range conns {
		if conn.Uuid == n.Uuid || conn.wrong {
			continue
		}
		switch conn.types {
		case ConnWithTCP:
			if len(tcpRow) == 0 {
				tcpRow = tcpsplit.MakeReqBody(req, num, int(fmsg.FuncID))
			}
			for _, row := range tcpRow {
				if _, err := conn.tconn.Write(row); err != nil {
					goto NextConn
				}
			}
			body, err := conn.WaitRspByte(ctx, num, fmsg)
			return &CallResp{msg: &conn.NodeInfo, con: "TCP", rsp: body, err: err}

		case ConnWithUDP:
			if len(udpRow) == 0 {
				udpRow = udpsplit.MakeReqBody(req, num, int(fmsg.FuncID))
			}
			for _, row := range udpRow {
				if _, err := conn.uconn.Write(row); err != nil {
					goto NextConn
				}
			}
			body, err := conn.WaitRspByte(ctx, num, fmsg)
			return &CallResp{msg: &conn.NodeInfo, con: "UDP", rsp: body, err: err}

		case ConnWithHTTP:
			var addr = fmt.Sprintf("http://%s:%d/%s", conn.Host, conn.Hport, fmsg.ApiName)
			if resp, err := http.Post(addr, "application/octet-stream",
				bytes.NewBuffer(req)); err == nil {
				defer resp.Body.Close()
				code := resp.Header.Get("code")

				if code == HttpReqSuccessBody {
					body, err := ioutil.ReadAll(resp.Body)
					return &CallResp{msg: &conn.NodeInfo, con: "UDP", rsp: body, err: err}
				} else if code == HttpReqSuccessNull {
					return &CallResp{msg: &conn.NodeInfo, con: "UDP"}
				} else if code == HttpReqFailMessage {
					body, _ := ioutil.ReadAll(resp.Body)
					return &CallResp{msg: &conn.NodeInfo, con: "UDP", err: errors.New(string(body))}
				}
			}
		}
	NextConn:
	}

	if !remote {
		conns = nil
		goto TryRemote
	}
	return &CallResp{err: errors.New("call all server node with api, but all wrong"),
		msg: &pb.NodeInfo{}, con: "Remote"}
}

func (n *NodeConn) WaitRspByte(ctx context.Context, num int, fmsg *pb.FuncMsg) ([]byte, error) {
	var c = n.NewChan(num)
	defer n.DelChan(num)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case buff := <-c.body:
		return buff, nil
	case err := <-c.err:
		return nil, err
	}
}

//
func (n *NodeDetail) CallMultiByByte(duration time.Duration, uuid, name string, args ...[]byte) network.CallByte {
	if len(args) == 1 {
		return n.CallRemoteByte(duration, uuid, name, args[0])
	} else if len(args) > 1 {
		var req = &pb.MultiBody{Count: uint32(len(args)), Data: args}
		if bts, err := MarshalInterface(pb.Compiler_PROTO, req); err == nil {
			return n.CallRemoteByte(duration, uuid, name, bts)
		} else {
			return &CallResp{err: errors.New("combined request body error: " + err.Error()),
				msg: &pb.NodeInfo{}, con: "Local"}
		}
	} else {
		return &CallResp{err: errors.New("call args cannot null"), msg: &pb.NodeInfo{}, con: "Local"}
	}
}

func (n *NodeDetail) WatchSend(dur time.Duration, uuid string, fid uint32, req []byte) error {
	if uuid == "" || fid == 0 {
		return errors.New("uuid and function id cannot be null")
	}
	if conn := n.fmsg.GetNodeConn(uuid); conn == nil {
		return errors.New("not found this uuid")
	} else {
		ctx, cancel := context.WithTimeout(context.TODO(), dur)
		defer cancel()
		resp := n.connsCall(ctx, []*NodeConn{conn}, &pb.FuncMsg{ApiType: pb.ApiType_Send,
			FuncID: fid, Protocal: pb.Compiler_PROTO}, req, nil)
		return resp.Err()
	}
}

func (n *NodeDetail) WatchCall(dur time.Duration, uuid string, fid uint32, req []byte, rsp interface{}) error {
	if uuid == "" || fid == 0 {
		return errors.New("uuid and function id cannot be null")
	}
	if conn := n.fmsg.GetNodeConn(uuid); conn == nil {
		return errors.New("not found this uuid")
	} else {
		ctx, cancel := context.WithTimeout(context.TODO(), dur)
		defer cancel()
		resp := n.connsCall(ctx, []*NodeConn{conn}, &pb.FuncMsg{ApiType: pb.ApiType_Call,
			FuncID: fid, Protocal: pb.Compiler_PROTO}, req, rsp)
		return resp.Err()
	}
}
