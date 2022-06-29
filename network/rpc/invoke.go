package rpc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"micro/network/comm"
	"micro/network/pb"
)

// 内部发出RPC请求时，先检查一遍本身的服务是否有提供相应的接口
// Gate: 通过网关发现调度服务的地址，转发请求到对应的节点接口，并考虑负载均衡和节点情况
// Register : 结构方法注册后，由 watcher 推送需要用到的rpc接口的节点地址，如启用了负载均衡则
// 		按规则一定调用频率通过和发现节点通信接口状况

// 注册服务连接watcher时，会将节点请求到的RPC接口记录并推送相关节点接口的地址和信息

type CallResp struct {
	err error
	msg *pb.NodeInfo
	con string
	rsp []byte
}

func (c *CallResp) Err() error {
	return c.err
}
func (c *CallResp) NodeUuid() string {
	return c.msg.Uuid
}
func (c *CallResp) NodeBase() pb.NodeInfo {
	return *c.msg
}
func (c *CallResp) Network() string {
	return c.con
}
func (c *CallResp) RespBody() []byte {
	return c.rsp
}

func (n *NodeDetail) send(ctx context.Context, uuid, name string, req interface{}) *CallResp {
	if name == "" {
		return &CallResp{err: errors.New("send server api name cannot be null"), msg: &n.NodeInfo}
	}
	nodename, apiname := SplitApiName(name)
	fmsg := n.QueryFunc(0, apiname)
	if fmsg == nil || fmsg.ApiType != pb.ApiType_Send {
		return &CallResp{err: errors.New("call not found this server api: " + name), msg: &n.NodeInfo}
	}

	// check local server function
	if uuid != "" && n.Uuid == uuid {
		if s, ok := n.funcs[fmsg.ServName]; ok {
			f, ok := s.funcs[fmsg.FuncName]
			if !ok || f.api != pb.ApiType_Call {
				return &CallResp{err: errors.New("local not found this api or input values wrong"),
					msg: &n.NodeInfo, con: "Local"}
			}
			return &CallResp{err: f.LocalSend(s.rv, req), msg: &n.NodeInfo, con: "Local"}
		}
		return &CallResp{err: errors.New("local server node not found struct service"), msg: &n.NodeInfo}
	} else if nodename == "" || n.Name == nodename {
		if s, ok := n.funcs[fmsg.ServName]; ok {
			if f, ok := s.funcs[fmsg.FuncName]; ok || f.api != pb.ApiType_Call {
				return &CallResp{err: f.LocalSend(s.rv, req), msg: &n.NodeInfo, con: "Local"}
			}
		}
	}
	// check remote server api to request
	return n.remoteCall(ctx, uuid, nodename, fmsg, req, nil)
}

type WaitDone struct {
	wg  sync.WaitGroup
	mut sync.Mutex
	msg []string
}

func (n *NodeDetail) sendall(ctx context.Context, name string, req interface{}, rsp *pb.SendAllRsp) error {
	if name == "" {
		return errors.New("send server api name cannot be null")
	}
	_, apiname := SplitApiName(name)
	fmsg := n.QueryFunc(0, apiname)
	if fmsg == nil {
		return errors.New("not found this server api: " + name)
	}
	conns := n.GetRemoteConn(ctx, fmsg)
	if len(conns) <= 0 {
		return fmt.Errorf("not found remote node")
	}

	bts, err := MarshalInterface(fmsg.Protocal, req)
	if err != nil {
		return errors.New("remoteCall marshal error: " + err.Error())
	}
	var num = n.serial(fmsg.FuncID)
	var tcpRow = tcpsplit.MakeReqBody(bts, num, int(fmsg.FuncID))
	var udpRow [][]byte

	var wait = &WaitDone{}
	for _, conn := range conns {
		wait.wg.Add(1)
		go func(nc *NodeConn) {
			defer wait.wg.Done()

			var result = &pb.SendRsp{Uuid: nc.Uuid}
			if nc.tconn != nil { // try request by tcp
				result.Network, result.Success = "TCP", true
				for _, row := range tcpRow {
					if _, err = nc.tconn.Write(row); err != nil {
						result.Success = false
						break
					}
				}
			}
			if !result.Success && nc.uconn != nil { // try request by udp
				if len(udpRow) == 0 {
					udpRow = udpsplit.MakeReqBody(bts, num, int(fmsg.FuncID))
				}
				result.Network, result.Success = "UDP", true
				for _, row := range udpRow {
					if _, err = nc.uconn.Write(row); err != nil {
						result.Success = false
						break
					}
				}
			}
			if !result.Success && nc.Hport != 0 { // try request by http
				aerr, perr := postHttpApi(fmsg, nc.Host, nc.Hport, bts, nil)
				if aerr == nil && perr == nil {
					result.Network, result.Success = "HTTP", true
				}
			}
			wait.mut.Lock()
			rsp.Result = append(rsp.Result, result)
			wait.mut.Unlock()
		}(conn)
	}

	wait.wg.Wait()
	if len(wait.msg) == 0 {
		return nil
	} else {
		return fmt.Errorf("list server uuid send error: [%s]", strings.Join(wait.msg, ", "))
	}
}

func (n *NodeDetail) call(ctx context.Context, uuid, name string, req, rsp interface{}) *CallResp {
	if name == "" {
		return &CallResp{err: errors.New("call server api name cannot be null"), msg: &n.NodeInfo}
	}
	nodename, apiname := SplitApiName(name)
	fmsg := n.QueryFunc(0, apiname)
	if fmsg == nil || fmsg.ApiType != pb.ApiType_Call {
		return &CallResp{err: errors.New("call not found this server api: " + name), msg: &n.NodeInfo}
	}
	// check local server function
	if uuid != "" && n.Uuid == uuid {
		if s, ok := n.funcs[fmsg.ServName]; ok {
			f, ok := s.funcs[fmsg.FuncName]
			if !ok || f.api != pb.ApiType_Call {
				return &CallResp{err: errors.New("local not found this api or input values wrong"),
					msg: &n.NodeInfo, con: "Local"}
			}
			return &CallResp{err: f.LocalCall(s.rv, req, rsp), msg: &n.NodeInfo, con: "Local"}
		}
		return &CallResp{err: errors.New("local server node not found struct service"), msg: &n.NodeInfo}
	} else if nodename == "" || n.Name == nodename {
		if s, ok := n.funcs[fmsg.ServName]; ok {
			if f, ok := s.funcs[fmsg.FuncName]; ok || f.api != pb.ApiType_Call {
				return &CallResp{err: f.LocalCall(s.rv, req, rsp), msg: &n.NodeInfo, con: "Local"}
			}
		}
	}
	// check remote server api to request
	return n.remoteCall(ctx, uuid, nodename, fmsg, req, rsp)
}

func (n *NodeDetail) multi(ctx context.Context, uuid, name string, args ...interface{}) *CallResp {
	nodename, apiname := SplitApiName(name)
	fmsg := n.QueryFunc(0, apiname)
	if fmsg == nil || fmsg.ApiType != pb.ApiType_Multi {
		return &CallResp{err: errors.New("call not found this server api: " + name), msg: &n.NodeInfo}
	}
	// check local server function
	if uuid != "" && n.Uuid == uuid {
		if s, ok := n.funcs[fmsg.ServName]; ok {
			f, ok := s.funcs[fmsg.FuncName]
			if !ok {
				return &CallResp{err: errors.New("local not found this api or input values wrong"),
					msg: &n.NodeInfo, con: "Local"}
			}
			return &CallResp{err: f.LocalMulti(s.rv, args), msg: &n.NodeInfo, con: "Local"}
		}
		return &CallResp{err: errors.New("local server node not found struct service"), msg: &n.NodeInfo}
	} else if nodename == "" || n.Name == nodename {
		if s, ok := n.funcs[fmsg.ServName]; ok {
			if f, ok := s.funcs[fmsg.FuncName]; ok {
				return &CallResp{err: f.LocalMulti(s.rv, args), msg: &n.NodeInfo, con: "Local"}
			}
		}
	}
	// check remote server api to request
	return n.remoteMulti(ctx, uuid, nodename, fmsg, args)
}

// request remote server api
func (n *NodeDetail) remoteCall(ctx context.Context, uuid, nodename string,
	fmsg *pb.FuncMsg, req, rsp interface{}) *CallResp {
	bts, err := MarshalInterface(fmsg.Protocal, req)
	if err != nil {
		return &CallResp{err: errors.New("remoteCall marshal error: " + err.Error()),
			msg: &pb.NodeInfo{}, con: "Local"}
	}

	var conns []*NodeConn
	if uuid != "" || nodename != "" {
		conns = n.GetAssignConn(ctx, uuid, nodename)
	} else if fmsg != nil {
		conns = n.GetRemoteConn(ctx, fmsg)
	}
	if len(conns) <= 0 {
		return &CallResp{err: fmt.Errorf("no multi remote or get remote error"),
			msg: &pb.NodeInfo{}, con: "Comm"}
	}
	return n.connsCall(ctx, conns, fmsg, bts, rsp)
}

// request remote server api
func (n *NodeDetail) remoteMulti(ctx context.Context, uuid, nodename string,
	fmsg *pb.FuncMsg, args []interface{}) *CallResp {
	var req = &pb.MultiBody{Count: uint32(len(args) - 1)}
	for i := 0; i < len(args)-1; i++ {
		if bts, err := MarshalInterface(fmsg.Protocal, args[i]); err == nil {
			req.Data = append(req.Data, bts)
		} else {
			return &CallResp{err: errors.New("remoteCall marshal error: " + err.Error()),
				msg: &pb.NodeInfo{}, con: "Local"}
		}
	}

	var conns []*NodeConn
	if uuid != "" || nodename != "" {
		conns = n.GetAssignConn(ctx, uuid, nodename)
	} else if fmsg != nil {
		conns = n.GetRemoteConn(ctx, fmsg)
	}
	if len(conns) <= 0 || len(args) < 2 {
		return &CallResp{err: fmt.Errorf("no multi remote or get remote error"),
			msg: &pb.NodeInfo{}, con: "Comm"}
	}

	if bts, err := MarshalInterface(pb.Compiler_PROTO, req); err == nil {
		return n.connsCall(ctx, conns, fmsg, bts, args[len(args)-1])
	} else {
		return &CallResp{err: errors.New("remoteCall marshal error: " + err.Error()),
			msg: &pb.NodeInfo{}, con: "Local"}
	}
}

func (n *NodeDetail) serial(fid uint32) int {
	if int(fid) < comm.WATCH_IN_MAX {
		return int(fid)
	} else {
		var num = int(atomic.AddUint32(&n.reqnum, 1))
		if num < comm.WATCH_IN_MAX {
			atomic.SwapUint32(&n.reqnum, uint32(comm.WATCH_IN_MAX))
			num = int(atomic.AddUint32(&n.reqnum, 1))
		}
		return num % 65535
	}
}

func (n *NodeDetail) connsCall(ctx context.Context, conns []*NodeConn,
	fmsg *pb.FuncMsg, bts []byte, rsp interface{}) *CallResp {
	var num = n.serial(fmsg.FuncID)
	var tcpRow, udpRow [][]byte
	for _, conn := range conns {
		if conn.Uuid == n.Uuid || conn.wrong {
			continue
		} else {
			switch conn.types {
			case ConnWithTCP:
				if len(tcpRow) == 0 {
					tcpRow = tcpsplit.MakeReqBody(bts, num, int(fmsg.FuncID))
				}
				for _, row := range tcpRow {
					if _, err := conn.tconn.Write(row); err != nil {
						goto NextConn
					}
				}
				return &CallResp{err: conn.WaitRsp(ctx, num, fmsg, rsp), msg: &conn.NodeInfo, con: "TCP"}

			case ConnWithUDP:
				if len(udpRow) == 0 {
					udpRow = udpsplit.MakeReqBody(bts, num, int(fmsg.FuncID))
				}
				for _, row := range udpRow {
					if _, err := conn.uconn.Write(row); err != nil {
						goto NextConn
					}
				}
				return &CallResp{err: conn.WaitRsp(ctx, num, fmsg, rsp), msg: &conn.NodeInfo, con: "UDP"}

			case ConnWithHTTP:
				apierr, err := postHttpApi(fmsg, conn.Host, conn.Hport, bts, rsp)
				if err == nil {
					return &CallResp{err: apierr, msg: &conn.NodeInfo, con: "HTTP"}
				}
			}
		}
	NextConn:
	}
	return &CallResp{err: errors.New("call all server node with api, but all wrong"),
		msg: &pb.NodeInfo{}, con: "Remote"}
}

func (n *NodeConn) WaitRsp(ctx context.Context, num int, fmsg *pb.FuncMsg, rsp interface{}) error {
	var c = n.NewChan(num)
	defer n.DelChan(num)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case buff := <-c.body:
		if len(buff) == 0 || rsp == nil {
			return nil
		}
		return UnmarshalInterface(fmsg.Protocal, rsp, buff)
	case err := <-c.err:
		return err
	}
}

// Get server api remote connect list
func (n *NodeDetail) GetRemoteConn(ctx context.Context, fmsg *pb.FuncMsg) []*NodeConn {
	var rows = n.fmsg.GetFuncConn(fmsg.FuncID, fmsg.ApiName)
	if len(rows) > 0 {
		return rows
	}
	rsp, err := n.WatchApi().GetApiConn(ctx, fmsg.FuncID, fmsg.ApiName)
	if err == nil && len(rsp.List) > 0 {
		var ids []string
		for _, node := range rsp.List {
			if conn := n.fmsg.GetNodeConn(node.Uuid); conn != nil {
				if conn.TestConn() == nil {
					ids = append(ids, node.Uuid)
					rows = append(rows, conn)
					continue
				}
			}
			if conn, err := n.NodeBaseToConn(node); err == nil {
				n.fmsg.PutConn(conn)
				ids = append(ids, node.Uuid)
				rows = append(rows, conn)
			}
		}
		if len(ids) < len(rsp.List) {
			n.fmsg.UpFuncNode(fmsg.FuncID, ids)
		}
	}
	return rows
}

// Get server remote connect list by node uuid or node name
func (n *NodeDetail) GetAssignConn(ctx context.Context, uuid, name string) []*NodeConn {
	if uuid != "" {
		if conn := n.fmsg.GetNodeConn(uuid); conn != nil {
			return []*NodeConn{conn}
		}
	}
	if name != "" {
		return []*NodeConn{}
	}

	rsp, err := n.WatchApi().GetNodeMsg(ctx, uuid, name)
	if err != nil || rsp == nil || (rsp.Data == nil && len(rsp.List) == 0) {
		return nil
	}
	if rsp.GetData() != nil {
		if conn, err := n.NodeBaseToConn(rsp.Data); err == nil {
			n.fmsg.PutConn(conn)
			return []*NodeConn{conn}
		}
	}
	var conns []*NodeConn
	for _, node := range rsp.List {
		if conn := n.fmsg.GetNodeConn(node.Uuid); conn != nil {
			if conn.TestConn() == nil {
				n.fmsg.PutConn(conn)
				conns = append(conns, conn)
				continue
			}
		}
		if conn, err := n.NodeBaseToConn(node); err == nil {
			n.fmsg.PutConn(conn)
			conns = append(conns, conn)
		}
	}
	return conns
}
