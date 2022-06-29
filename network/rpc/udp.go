package rpc

import (
	"net"
	"sync"

	"micro/common"
	"micro/network/comm"
)

var (
	udpCleanBuf = make([]byte, udpsplit.TotalSize)
	udpBodyPool = &sync.Pool{New: func() interface{} {
		return &ConnBody{Data: make([]byte, udpsplit.TotalSize)}
	}}
)

func NewUdpBuffer() *ConnBody {
	return udpBodyPool.Get().(*ConnBody)
}
func PutUdpBuffer(c *ConnBody) {
	copy(c.Data[0:], udpCleanBuf)
	udpBodyPool.Put(c)
}

var udpsplit = NetworkBuffer{
	TotalSize: 534,
	BodyStart: 12,
	BodySplit: 520,
	FinalByte: 532,
	PingBytes: func() []byte {
		var d = make([]byte, 534)
		var pre = buffPrefix(1, comm.PingNetwork, 1, 1, 0)
		copy(d[0:], pre)
		d[len(d)-1] = FirstByte
		return d
	}(),
}

func UdpDialTest(addr *net.UDPAddr) error {
	if conn, err := net.DialUDP("udp", nil, addr); err != nil {
		return err
	} else {
		defer conn.Close()
		_, err = conn.Write(udpsplit.PingBytes)
		return err
	}
}

func (n *NodeDetail) DialUDP(addr *net.UDPAddr) (*net.UDPConn, error) {
	conn, err := net.DialUDP("udp", nil, addr)
	if err == nil {
		for _, row := range n.tmps.udpreg {
			if _, err = conn.Write(row); err != nil {
				conn.Close()
			}
		}
	}
	return conn, err
}

// ServeCodec is like ServeConn but uses the specified codec to
// decode requests and encode responses.
// 接收请求数据
func (n *NodeDetail) UdpListen(port int) error {
	if udpconn, err := net.ListenUDP("udp", &net.UDPAddr{
		Port: port, IP: net.ParseIP("0.0.0.0"),
	}); err != nil {
		return err
	} else {
		go func(nd *NodeDetail, conn *net.UDPConn) {
			defer common.Recover()
			defer conn.Close()

			r := &NodeConn{
				uconn: conn,
				types: ConnWithUDP,
				fc:    make(map[string]bool),
				rc:    make(map[int]*RecvChan),
				list:  make(map[int]*ReadLink),
			}

			for {
				buff := NewUdpBuffer()
				if _, addr, err := conn.ReadFromUDP(buff.Data); err != nil {
					PutUdpBuffer(buff)
					break
				} else {
					go func(rc *NodeConn, addr *net.UDPAddr, b *ConnBody) {
						if rows := n.ParseRspByte(udpsplit, r, b); len(rows) >= 0 {
							for _, row := range rows {
								if _, err := r.uconn.WriteToUDP(row, addr); err != nil {
									return
								}
							}
						}
						PutUdpBuffer(buff)
					}(r, addr, buff)
				}
			}
			conn.Close()
		}(n, udpconn)
	}
	return nil
}

var udpconns = sync.Pool{
	New: func() interface{} {
		uconn, _ := net.DialUDP("udp", nil, &net.UDPAddr{})
		return uconn
	},
}
