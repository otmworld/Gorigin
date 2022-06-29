# Golang Micro Frame

## 
```sh
- Node  # Server Node (1:1)
- Proc  # Server-Process (1:m)  (Address:Host -> Server)
```

## Heartbeat

```sh
- 定时请求保持连接通畅


// TODO:
- 异常断开，从高频到低频测试连接 (纠正端口动态变化问题)，并进行负载调节
- 主动断开，关闭节点或进程连接心跳，并进行负载调节
- 慢恢复启动，进行有效的负载均衡
```

## Network (RPC)

```go
// Rpc采用TCP协议
// Rpc网络连接可复用并发请求

// send request, default timeout : one minute
func RPCSend(name string, req interface{}) error

// request to call response, default timeout : one minute
func RPCCall(name string, req, rsp interface{}) error

// can use context to cancel call
// eg: Timeout, but cannot reback event
func RPCContextCall(ctx context.Context, name string, req, rsp interface{}) error

// make rpc call with timeout
func RPCTimeOutCall(duration time.Duration, name string, req, rsp interface{}) error
```

## Examples

```go
// Watcher Server
func main() {
    s := watcher.NewWatchServer()
	if err := s.RunServer(8091); err != nil {
		panic(err)
	}
}

// Node Client Server
func main() {
    client := watcher.NewWatchClient("Test", "127.0.0.1:8091")
	rs, err := client.InitRpcServer()
	if err != nil {
		panic(err)
	}
    rs.SetBuffSize(64)
    rs.Register(&Struct{})
    rs.SignalExit()
    rs.SetListenAddr("127.0.0.1:9091")
    client.RunServer()
}
```