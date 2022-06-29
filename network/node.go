package network

import (
	"context"
	"time"

	"micro/network/pb"
	"micro/timer"
)

type NodeConfig struct {
	// system version
	Version string
	// system node name
	NodeName string

	// aatcher node config
	Watchers []*WatcherConfig

	// localhost ip
	// default ip: 0.0.0.0, port: free port
	IPAddr string
	// heart-beat of test watcher with connect
	HeartBeatTime time.Duration

	// localhost tcp listen port
	// default switch on to use, free port
	TcpPort uint64
	// default first use tcp listen and send
	TcpListenOff bool

	// localhost tcp listen port
	// when watcher tcp switch on, default use
	UdpPort uint64
	// default not use udp listen
	UdpListenOn bool

	// localhost tcp listen port
	// default switch on to use, free port
	HttpPort uint64
	// default use http listen
	HttpListenOff bool
}

type WatcherConfig struct {
	// watcher ip eg: [127.0.0.1]
	Host string
	// watcher tcp listen port to connect
	TcpPort uint64
	// watcher udp listen port to connect
	UdpPort uint64
	// watcher http listen port to connect
	HttpPort uint64
	// config file path
	ConfigPath string
}

type Node interface {
	// register struct functions
	Register(interface{}) error
	// running server
	RunServer() (NodeApi, error)

	GetNodeMsg() pb.NodeInfo
	GetTimeTicker() timer.TimerStruct
}

type NodeApi interface {
	GetTimeTicker() timer.TimerStruct

	GetNodeDetail() pb.NodeInfo
	GetNodeByUuid(string) (pb.NodeInfo, error)
	GetNodeByName(string) ([]*pb.NodeInfo, error)

	// call rpc interface to make response
	CallAuto(name string, req, rsp interface{}) CallResp
	CallByUuid(uuid, name string, req, rsp interface{}) CallResp
	CallAutoContext(ctx context.Context, name string, req, rsp interface{}) CallResp
	CallAutoTimeout(duration time.Duration, name string, req, rsp interface{}) CallResp
	CallRemoteByte(duration time.Duration, uuid, name string, req []byte) CallByte

	// call by multi request args
	CallMultiAuto(name string, args ...interface{}) CallResp
	CallMultiByUuid(uuid, name string, args ...interface{}) CallResp
	CallMultiByByte(duration time.Duration, uuid, name string, args ...[]byte) CallByte

	// send data to other server
	SendAuto(name string, req interface{}) error
	SendByUuid(uuid, name string, req interface{}) error
	SendAutoAll(name string, req interface{}, rsp *pb.SendAllRsp) error
	SendAutoContext(ctx context.Context, name string, req interface{}) error
	SendAutoTimeout(duration time.Duration, name string, req interface{}) error
}

type CallResp interface {
	Err() error
	NodeUuid() string
	NodeBase() pb.NodeInfo
	Network() string // TCP, UDP, HTTP, Local, Remote
}

type CallByte interface {
	CallResp
	RespBody() []byte
}
