package rpc

import (
	"log"
	"net"
	"sync"
	"time"

	"micro/common"
	"micro/network/comm"
)

var (
	tcpCleanBuf = make([]byte, tcpsplit.TotalSize)
	tcpBodyPool = &sync.Pool{New: func() interface{} {
		return &ConnBody{Data: make([]byte, tcpsplit.TotalSize)}
	}}
)

func NewTcpBuffer() *ConnBody {
	return tcpBodyPool.Get().(*ConnBody)
}
func PutTcpBuffer(c *ConnBody) {
	copy(c.Data[0:], tcpCleanBuf)
	tcpBodyPool.Put(c)
}

var tcpsplit = NetworkBuffer{
	TotalSize: 1444,
	BodyStart: 12,
	BodySplit: 1430,
	FinalByte: 1442,
	PingBytes: func() []byte {
		var d = make([]byte, 1444)
		var pre = buffPrefix(1, comm.PingNetwork, 1, 1, 0)
		copy(d[0:], pre)
		d[len(d)-1] = FirstByte
		return d
	}(),
}

// 测试Tcp地址是否能用
func TcpDialTest(host string, port int) error {
	if conn, err := net.DialTCP("tcp", nil, &net.TCPAddr{
		IP: net.ParseIP(host), Port: port,
	}); err != nil {
		return err
	} else {
		defer conn.Close()
		_, err = conn.Write(tcpsplit.PingBytes)
		return err
	}
}

func (n *NodeDetail) DialTCP(addr *net.TCPAddr) (*net.TCPConn, error) {
	conn, err := net.DialTCP("tcp", nil, addr)
	if err == nil {
		for _, row := range n.tmps.tcpreg {
			if _, err = conn.Write(row); err != nil {
				conn.Close()
			}
		}
	}
	return conn, err
}

// Tcp端口监听连接请求
func (n *NodeDetail) TcpListen(port int) error {
	if listen, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP: net.ParseIP("0.0.0.0"), Port: int(n.Tport)},
	); err != nil {
		return err
	} else {
		go func(s *NodeDetail, t *net.TCPListener) {
			for {
				if conn, err := t.Accept(); err != nil {
					log.Printf("rpc.Serve: accept: %v\n", err)
				} else {
					go s.tcpAccept(conn.(*net.TCPConn))
				}
			}
		}(n, listen)
	}
	return nil
}

// ServeCodec is like ServeConn but uses the specified codec to
// decode requests and encode responses.
func (n *NodeDetail) tcpAccept(conn *net.TCPConn) {
	defer common.Recover()
	defer conn.Close()

	r := &NodeConn{
		tconn: conn,
		types: ConnWithTCP,
		fc:    make(map[string]bool),
		rc:    make(map[int]*RecvChan),
		list:  make(map[int]*ReadLink),
	}

	var num int
	var err error
	for {
		buff := NewTcpBuffer()
		num, err = conn.Read(buff.Data)
		if err != nil {
			PutTcpBuffer(buff)
			return
		} else if num < tcpsplit.TotalSize {
			sum := num
			for sum < tcpsplit.TotalSize {
				num, err = conn.Read(buff.Data[sum:])
				if err != nil {
					PutTcpBuffer(buff)
					return
				}
				sum += num
			}
		}

		go func(rc *NodeConn, b *ConnBody) {
			if rows := n.ParseRspByte(tcpsplit, r, b); len(rows) >= 0 {
				for _, row := range rows {
					if _, err := rc.tconn.Write(row); err != nil {
						rc.tconn.SetReadDeadline(time.Now())
						rc.tconn.Close()
						break
					}
				}
			}
			PutTcpBuffer(buff)
		}(r, buff)
	}
}
