package rpc

import (
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"micro/network/pb"
)

type ConnType int

var (
	ConnWithTCP  ConnType = 1
	ConnWithUDP  ConnType = 2
	ConnWithHTTP ConnType = 3
)

type NodeConn struct {
	pb.NodeInfo

	tconn *net.TCPConn
	uconn *net.UDPConn
	types ConnType
	wrong bool
	stamp int64

	// safe lock rc to read and wirte
	mut sync.RWMutex
	// struct function (api name), send boolean
	fc map[string]bool
	// recv response
	rc map[int]*RecvChan
	// recv response body
	list map[int]*ReadLink
}

type RecvChan struct {
	body  chan []byte
	err   chan error
	stamp int64
}

type ReadLink struct {
	funcid int      // server api id
	buck   int      // body slice split lenght
	uuid   int      // event id
	body   [][]byte // request body
	recv   uint32   // recv count
}

func (n *NodeConn) NewChan(num int) *RecvChan {
	var tmp = &RecvChan{
		body:  make(chan []byte, 1),
		err:   make(chan error, 1),
		stamp: time.Now().UnixMilli(),
	}
	n.mut.Lock()
	n.rc[num] = tmp
	n.mut.Unlock()
	return tmp
}

func (n *NodeConn) DelChan(num int) {
	n.mut.Lock()
	delete(n.rc, num)
	n.mut.Unlock()
}

func (n *NodeConn) Close() {
	n.mut.Lock()
	defer n.mut.Unlock()
	if n.tconn != nil {
		n.tconn.SetReadDeadline(time.Now())
		n.tconn.Close()
	}
	if n.uconn != nil {
		n.uconn.SetReadDeadline(time.Now())
		n.uconn.Close()
	}
	n.fc = make(map[string]bool)
	n.rc = make(map[int]*RecvChan)
}

func (n *NodeDetail) RefreshConn(nc *NodeConn) error {
	var err error
	if nc.Tport != 0 {
		if nc.tconn, err = n.DialTCP(&net.TCPAddr{
			IP: net.ParseIP(nc.Host), Port: int(nc.Tport)}); err != nil {
			return err
		}
		if err = nc.TestTcpConn(); err != nil {
			nc.tconn.Close()
			return err
		}
		nc.tconn.SetKeepAlive(true)
		nc.types = ConnWithTCP
		go nc.ReadRespTCP()
	}
	if nc.Uport != 0 {
		if nc.uconn, err = n.DialUDP(&net.UDPAddr{
			IP: net.ParseIP(nc.Host), Port: int(nc.Uport)}); err != nil {
			return err
		}
		if err = nc.TestUdpConn(); err != nil {
			nc.uconn.Close()
			return err
		}
		nc.types = ConnWithUDP
		go nc.ReadRespUDP()
		return nil
	}
	if nc.Hport != 0 {
		if HttpPingTest(nc.Host, nc.Hport) == nil {
			nc.types = ConnWithHTTP
		}
	}
	return err
}

func (n *NodeDetail) TimerCheck(nc *NodeConn) {
	n.ticker.AddDurationBoolean(time.Second*10, func() bool {
		if !nc.wrong {
			return false
		}
		switch nc.types {
		case ConnWithTCP:
			if nc.TestTcpConn() != nil {
				nc.tconn.SetReadDeadline(time.Now())
				nc.tconn.Close()
				n.RefreshConn(nc)
			}
		case ConnWithUDP:
			if nc.TestUdpConn() != nil {
				nc.tconn.SetReadDeadline(time.Now())
				nc.uconn.Close()
				n.RefreshConn(nc)
			}
		case ConnWithHTTP:
			if HttpPingTest(n.Host, nc.Hport) != nil {
				n.RefreshConn(nc)
			}
		default:
			if n.RefreshConn(nc) != nil {
				return false
			}
		}
		if !nc.wrong {
			return false
		}

		var stamp = time.Now().Add(time.Second * -10).UnixMilli()
		nc.mut.Lock()
		for num, v := range nc.rc {
			if v.stamp < stamp {
				delete(nc.rc, num)
			}
		}
		nc.mut.Unlock()
		return true
	})
}

// parse network request body
// return request event id, function_id, request body, error
func (n *NodeConn) ParseResp(nt NetworkBuffer, cb *ConnBody) (int, int, []byte, error) {
	body, err := nt.parse(cb)
	if err != nil || body == nil || body.Sort > body.Buck {
		return 0, 0, nil, err
	}
	if body.Buck == 1 {
		return body.Uuid, body.Func, body.Data, err

	} else if body.Buck > 1 {
		n.mut.RLock()
		tmp, ok := n.list[body.Uuid]
		n.mut.RUnlock()
		if ok {
			if body.Buck != tmp.buck || body.Func != tmp.funcid {
				n.mut.Lock()
				delete(n.list, body.Uuid)
				n.mut.Unlock()
				return 0, 0, nil, errors.New("recv wrong")
			}
			if body.Sort-1 >= len(tmp.body) {
				n.mut.Lock()
				delete(n.list, body.Uuid)
				n.mut.Unlock()
				return 0, 0, nil, errors.New("recv uuid body buck wrong")
			}
			if len(tmp.body[body.Sort-1]) == 0 {
				atomic.AddUint32(&tmp.recv, 1)
				tmp.body[body.Sort-1] = body.Data

				if int(tmp.recv) >= tmp.buck {
					var result []byte
					for i := range tmp.body {
						if len(tmp.body[i]) == 0 {
							return 0, 0, nil, nil
						}
						result = append(result, tmp.body[i]...)
					}
					n.mut.Lock()
					delete(n.list, body.Uuid)
					n.mut.Unlock()
					return body.Uuid, body.Func, result, nil
				}
			}
		} else if !ok {
			tmp = &ReadLink{
				funcid: body.Func,
				buck:   body.Buck,
				uuid:   body.Uuid,
				body:   make([][]byte, body.Buck),
				recv:   1,
			}
			if body.Sort <= 0 || body.Sort-1 >= len(tmp.body) {
				n.mut.Lock()
				delete(n.list, body.Uuid)
				n.mut.Unlock()
				return 0, 0, nil, errors.New("recv uuid body buck wrong")
			}
			tmp.body[body.Sort-1] = body.Data
			n.mut.Lock()
			n.list[body.Uuid] = tmp
			n.mut.Unlock()
		}
	}
	return 0, 0, nil, nil
}

// 用于接收处理自己请求出去的返回数据
func (n *NodeConn) ReadRespTCP() {
	var num int
	var err error
	var bts []byte

	for {
		buff := NewTcpBuffer()
		num, err = n.tconn.Read(buff.Data)
		if err != nil {
			PutTcpBuffer(buff)
			return
		} else if num < tcpsplit.TotalSize {
			sum := num
			for sum < tcpsplit.TotalSize {
				if num, err = n.tconn.Read(buff.Data[sum:]); err != nil {
					PutTcpBuffer(buff)
					return
				}
				sum += num
			}
		}

		num, _, bts, err = n.ParseResp(tcpsplit, buff)
		if num > 0 {
			n.mut.Lock()
			c, ok := n.rc[num]
			n.mut.Unlock()
			if ok {
				if err != nil {
					c.err <- err
				} else {
					c.body <- bts
				}
			}
		}
		PutTcpBuffer(buff)
	}
}

// 用于接收处理自己请求出去的返回数据
func (n *NodeConn) ReadRespUDP() {
	var num int
	var err error
	var bts []byte

	for {
		buff := NewUdpBuffer()
		num, _, err = n.uconn.ReadFromUDP(buff.Data)
		if err != nil {
			PutUdpBuffer(buff)
			return
		} else if num != udpsplit.TotalSize {
			PutUdpBuffer(buff)
			continue
		}
		num, _, bts, err = n.ParseResp(udpsplit, buff)
		if num > 0 {
			n.mut.Lock()
			c, ok := n.rc[num]
			n.mut.Unlock()
			if ok {
				if err != nil {
					c.err <- err
				} else {
					c.body <- bts
				}
			}
		}
		PutUdpBuffer(buff)
	}
}

func (n *NodeConn) TestConn() error {
	if n == nil {
		return errors.New("connection was null")
	}
	switch n.types {
	case ConnWithTCP:
		return n.TestTcpConn()
	case ConnWithUDP:
		return n.TestUdpConn()
	case ConnWithHTTP:
		return HttpPingTest(n.Host, n.Hport)
	}
	return errors.New("no tcp or udp connection")
}

// 测试节点连接是否可用
func (n *NodeConn) TestTcpConn() error {
	if n.Tport != 0 {
		if n.tconn != nil {
			if _, err := n.tconn.Write(tcpsplit.PingBytes); err == nil {
				return nil
			}
		}
		var err error
		if n.tconn, err = net.DialTCP("tcp", nil, &net.TCPAddr{
			IP: net.ParseIP(n.Host), Port: int(n.Tport)}); err != nil {
			return err
		} else {
			_, err = n.tconn.Write(tcpsplit.PingBytes)
			return err
		}
	}
	return errors.New("this node not use tcp")
}

// 测试节点连接是否可用
func (n *NodeConn) TestUdpConn() error {
	if n.Uport != 0 {
		if n.uconn != nil {
			if _, err := n.uconn.Write(udpsplit.PingBytes); err == nil {
				return nil
			}
		}
		var err error
		if n.uconn, err = net.DialUDP("udp", nil, &net.UDPAddr{
			IP: net.ParseIP(n.Host), Port: int(n.Uport)}); err != nil {
			return err
		} else {
			_, err = n.uconn.Write(udpsplit.PingBytes)
			return err
		}
	}
	return errors.New("this node not use udp")
}
