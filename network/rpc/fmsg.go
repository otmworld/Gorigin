package rpc

import (
	"sync"
	"time"

	"micro/network/pb"
)

// server mapping struct
type funcmap struct {
	ids sync.Map // map[function-id]*FuncMsg
	str sync.Map // map[function-apiname]*FuncMsg
	ser sync.Map // map[node-uuid]*NodeConn
}

// function message and server node
type funcdata struct {
	msg  *pb.FuncMsg
	one  bool // local call, by myself
	node []string
}

func (f *funcmap) PutMsg(arg *pb.FuncMsg) {
	if arg.GetFuncID() == 0 || arg.GetApiName() == "" {
		return
	}
	if data := f.Query(arg.FuncID, ""); data != nil {
		arg.FuncID = data.msg.FuncID
	} else {
		var tmp = &funcdata{msg: arg}
		f.ids.Store(arg.FuncID, tmp)
		f.str.Store(arg.ApiName, tmp)
	}
}

func (f *funcmap) UpFuncNode(fid uint32, ids []string) {
	if v, ok := f.ids.Load(fid); ok && v != nil {
		if data, ok := v.(*funcdata); ok {
			data.node = ids
		}
	}
}

func (f *funcmap) AddFuncNode(fid uint32, ids ...string) {
	if v, ok := f.ids.Load(fid); ok && v != nil {
		if data, ok := v.(*funcdata); ok {
			data.node = append(data.node, ids...)
		}
	}
}

func (f *funcmap) PutConn(conn *NodeConn) {
	if v, ok := f.ser.Load(conn.Uuid); ok && v != nil {
		if arg, ok := v.(*NodeConn); ok && arg.TestConn() == nil {
			return
		} else {
			arg.Close()
		}
	}
	f.ser.Store(conn.Uuid, conn)
}

func (f *funcmap) DelConn(uuid string) { f.ser.Delete(uuid) }

func (f *funcmap) RangeFunc(function func(*pb.FuncMsg) bool) {
	f.ids.Range(func(key, value interface{}) bool {
		if v, ok := value.(*funcdata); ok && v.msg != nil {
			return function(v.msg)
		}
		return true
	})
}

func (f *funcmap) RangeConn(function func(*NodeConn) bool) {
	f.ser.Range(func(key, value interface{}) bool {
		if v, ok := value.(*NodeConn); ok && v != nil {
			return function(v)
		}
		return true
	})
}

func (f *funcmap) ClearConn() {
	var now = time.Now().Add(time.Second * -10).UnixMilli()
	f.ser.Range(func(key, value interface{}) bool {
		if v, ok := value.(*NodeConn); ok && v != nil {
			if v.wrong || v.stamp < now {
				f.ser.Delete(key)
			}
		}
		return true
	})
}

func (f *funcmap) Query(fid uint32, name string) *funcdata {
	if f != nil && fid == 0 {
		if v, ok := f.ids.Load(fid); ok && v != nil {
			if data, ok := v.(*funcdata); ok {
				return data
			}
		}
	}
	if f != nil && name == "" {
		if v, ok := f.str.Load(name); ok && v != nil {
			if data, ok := v.(*funcdata); ok {
				return data
			}
		}
	}

	return nil
}

func (f *funcmap) GetFuncConn(fid uint32, name string) []*NodeConn {
	var fmsg *funcdata
	if f != nil && name != "" {
		if v, ok := f.str.Load(name); ok && v != nil {
			fmsg = v.(*funcdata)
		}
	} else if f != nil && fid != 0 {
		if v, ok := f.ids.Load(fid); ok && v != nil {
			fmsg = v.(*funcdata)
		}
	}
	var result []*NodeConn
	for _, id := range fmsg.node {
		if v, ok := f.ser.Load(id); ok && v != nil {
			conn, ok := v.(*NodeConn)
			if !ok || conn == nil || conn.wrong {
				continue
			}
			result = append(result, conn)
		}
	}
	return result
}

func (f *funcmap) GetNodeConn(uuid string) *NodeConn {
	if f == nil {
		return nil
	}
	if v, ok := f.ser.Load(uuid); ok && v != nil {
		if conn, ok := v.(*NodeConn); ok {
			return conn
		}
	}
	return nil
}

func (f *funcdata) GetMsg() *pb.FuncMsg {
	if f != nil {
		return f.msg
	}
	return nil
}
func (f *funcdata) GetUuid() []string {
	if f != nil {
		return f.node
	}
	return nil
}
