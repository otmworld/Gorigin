package rpc

import (
	"errors"
	"strings"

	"micro/network/comm"
	"micro/network/pb"

	"google.golang.org/protobuf/proto"
)

// make built in request
func (n *NodeDetail) builtin(fid int, nc *NodeConn, bts []byte) ([]byte, error) {
	switch fid {
	case comm.PingNetwork:
		return []byte("PONG"), nil

	case comm.DialRegister:
		var rsp = &pb.NodeInfo{}
		if err := proto.Unmarshal(bts, rsp); err != nil {
			return nil, err
		} else {
			if rsp.Host == "" || rsp.Host == "127.0.0.1" || rsp.Host == "0.0.0.0" {
				if nc.types == ConnWithTCP {
					rsp.Host = strings.Split(nc.tconn.RemoteAddr().String(), ":")[0]
				} else if nc.types == ConnWithUDP {
					rsp.Host = strings.Split(nc.uconn.RemoteAddr().String(), ":")[0]
				}
			}
			n.NodeBaseToConn(rsp)
			return nil, nil
		}

	case comm.UpFuncMapList:
		var req = &pb.UpFuncList{}
		if err := proto.Unmarshal(bts, req); err != nil {
			return nil, err
		}
		for _, row := range req.Data {
			n.fmsg.PutMsg(row)
		}
		return nil, nil

	case comm.UpNodeConnMsg:
		var req = &pb.GetApiConnRsp{}
		if err := proto.Unmarshal(bts, req); err != nil {
			return nil, err
		}
		n.UpServNodeConn(req)
		return nil, nil

	case comm.UpServerState:

	case comm.UpWatcherList:
		// var req = &comm.UpWatcherListReq{}

	}
	return nil, errors.New("parse build-in request byte wrong")
}

func (n *NodeDetail) UpServNodeConn(data *pb.GetApiConnRsp) {
	var result = make([]*NodeConn, 0, len(data.List))
	for _, row := range data.List {
		if conn := n.fmsg.GetNodeConn(row.Uuid); conn != nil {
			conn.Host, conn.Hport = row.Host, row.Hport
			conn.Tport, conn.Uport = row.Tport, row.Uport
			result = append(result, conn)
		} else if conn, err := n.NodeBaseToConn(row); err == nil {
			n.fmsg.PutConn(conn)
			result = append(result, conn)
		}
	}
	if len(result) > 0 {
		// n.StoreConnByName(data.Func.ServName, result)
	}
}
