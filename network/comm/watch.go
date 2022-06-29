package comm

import (
	"micro/network/pb"
	"time"
)

// 心跳超时时间
var HeartbeatInterval = time.Second * 2
var NodeConnTimeOut = HeartbeatInterval * (-2)

type WatchFmsgData struct {
	Name string
	Fmsg map[uint32]*pb.FuncMsg
}

const WatchNodeName = "WATCHER"

const (
	Registered = 80
	Heartbeats = 81
	GetNodeMsg = 82
	GetFuncMsg = 83
	GetWatcher = 84
	GetApiConn = 85
	NewServers = 86
)

var WatchFmsg = &WatchFmsgData{Name: "WatchApi", Fmsg: map[uint32]*pb.FuncMsg{
	Heartbeats: &pb.FuncMsg{ApiType: pb.ApiType_Send, FuncName: "Heartbeats"},
	Registered: &pb.FuncMsg{ApiType: pb.ApiType_Call, FuncName: "Registered"},
	GetNodeMsg: &pb.FuncMsg{ApiType: pb.ApiType_Call, FuncName: "GetNodeMsg"},
	GetFuncMsg: &pb.FuncMsg{ApiType: pb.ApiType_Call, FuncName: "GetFuncMsg"},
	GetWatcher: &pb.FuncMsg{ApiType: pb.ApiType_Call, FuncName: "GetWatcher"},
	GetApiConn: &pb.FuncMsg{ApiType: pb.ApiType_Call, FuncName: "GetApiConn"},
	NewServers: &pb.FuncMsg{ApiType: pb.ApiType_Call, FuncName: "NewServers"},
}}

func init() {
	for id, row := range WatchFmsg.Fmsg {
		row.FuncID = id
		row.ServName = WatchFmsg.Name
		row.Protocal = pb.Compiler_PROTO
		row.ApiName = WatchFmsg.Name + "." + row.FuncName
	}
}

// Watch server name
func (w *WatchFmsgData) ServName() string { return w.Name }

// 服务注册
func (w *WatchFmsgData) Register() *pb.FuncMsg { return w.Fmsg[Registered] }

// 向主发现节点推送新服务接口
func (w *WatchFmsgData) NewStruct() *pb.FuncMsg { return w.Fmsg[NewServers] }

// 发送心跳到发现节点
func (w *WatchFmsgData) Heartbeat() *pb.FuncMsg { return w.Fmsg[Heartbeats] }

// 获取节点信息
func (w *WatchFmsgData) GetNodeMsg() *pb.FuncMsg { return w.Fmsg[GetNodeMsg] }

// 获取某个接口基础信息
func (w *WatchFmsgData) GetFuncMsg() *pb.FuncMsg { return w.Fmsg[GetFuncMsg] }

// 获取接口节点连接信息
func (w *WatchFmsgData) GetSerConn() *pb.FuncMsg { return w.Fmsg[GetApiConn] }

// 获取发现节点列表
func (w *WatchFmsgData) GetWatcher() *pb.FuncMsg { return w.Fmsg[GetWatcher] }
