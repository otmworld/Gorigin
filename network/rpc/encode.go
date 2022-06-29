package rpc

import (
	"errors"

	"micro/network/comm"
)

const (
	FirstByte  = byte(1)
	SecondByte = byte(0)

	// Request Parse
	BodyWholeData  = byte(0) // 完整请求体
	BodyBodyStart  = byte(1) // 请求体第一块
	BodyReqMiddle  = byte(2) // 请求体中间部分
	BodyReqFinaly  = byte(3) // 最后一块
	BodyReqDataNil = byte(4) // 请求体为空

	// Response Parse
	BodyRespSuccess = byte(11) // 请求成功,返回数据
	BodyRespFailed  = byte(12) // 请求失败,返回错误信息
	BodyRespNoFound = byte(13) // 请求接口未找到
	BodyRespDataNil = byte(14) // 请求成功,返回体为空
	BodyRespStart   = byte(15) // 请求体第一块
	BodyRespMiddle  = byte(16) // 请求体中间部分
	BodyRespFinaly  = byte(17) // 最后一块
)

type NetworkBuffer struct {
	TotalSize int
	BodySplit int
	BodyStart int
	FinalByte int
	PingBytes []byte
}

type JoinBodyData struct {
	Uuid int // request no
	Func int // server api
	Buck int // bucker number
	Sort int // sort number
	Data []byte
}

type ConnBody struct{ Data []byte }

func (n *NodeDetail) ParseRspByte(nb NetworkBuffer, nc *NodeConn, cb *ConnBody) [][]byte {
	num, fid, bts, err := nc.ParseResp(nb, cb)
	if err != nil || fid == 0 || num == 0 {
		return nil
	} else if fid < comm.BUILT_IN_MAX {
		bts, err = n.builtin(fid, nc, bts)
	} else if fmsg := n.QueryFunc(uint32(fid), ""); fmsg != nil {
		bts, err = n.findCall(fmsg, bts)
	} else {
		err = errors.New("not found server api mapping in server: " + nc.Uuid)
	}
	return nb.MakeRspBody(bts, num, fid, err)
}

// gen request body split
func (n NetworkBuffer) MakeReqBody(bts []byte, num, fid int) [][]byte {
	if len(bts) == 0 {
		var b = make([]byte, n.TotalSize)
		copy(b[0:], []byte{1, 0, byte(num >> 16 % 256), byte(num >> 8 % 256),
			byte(num % 256), BodyReqDataNil, byte(fid / 256), byte(fid % 256)})
		b[n.TotalSize-1] = FirstByte
		return [][]byte{b}

	} else if len(bts) <= n.BodySplit {
		var b = make([]byte, 0, n.TotalSize)
		b = append(b, buffPrefix(num, fid, 1, len(bts), BodyWholeData)...)
		b = append(b, bts...)
		b = append(b, make([]byte, n.TotalSize-len(b)-2)...)
		b = append(b, 0, 1)
		return [][]byte{b}

	} else {
		count := len(bts) / n.BodySplit
		last := len(bts) % n.BodySplit
		if last > 0 {
			count += 1
		} else {
			last = n.BodySplit
		}
		var result = make([][]byte, 0, count)
		var tmp = []byte{1, 0, byte(num >> 16 % 256), byte(num >> 8 % 256), byte(num % 256),
			BodyReqMiddle, byte(fid / 256), byte(fid % 256), byte(count / 256), byte(count % 256)}

		for i := 1; i < count; i++ {
			b := make([]byte, 10, n.TotalSize)
			copy(b[0:], tmp)
			b = append(b, byte(i>>8%256), byte(i%256))
			b = append(b, bts[n.BodySplit*(i-1):n.BodySplit*i]...)
			b = append(b, 0, 1)
			result = append(result, b)
		}
		result[0][5] = BodyBodyStart

		// Last split bytes check
		b := make([]byte, 10, n.TotalSize)
		copy(b[0:], tmp)
		b = append(b, byte(last>>8%256), byte(last%256))
		b = append(b, bts[n.BodySplit*(count-1):]...)
		b = append(b, make([]byte, n.BodySplit-last)...)
		b = append(b, 0, 1)
		b[5] = BodyReqFinaly
		result = append(result, b)
		return result
	}
}

func buffPrefix(num, fid, buck, last int, model byte) []byte {
	return []byte{
		FirstByte, SecondByte, // 前缀检查是否是正确开头
		byte(num >> 16 % 256), byte(num >> 8 % 256), byte(num % 256), // 事件编号
		model,                            // 分块类型
		byte(fid / 256), byte(fid % 256), // 服务接口
		byte(buck / 256), byte(buck % 256), // 块数
		byte(last / 256), byte(last % 256), // 序号 或 最后一块的长度
	}
}

func (n NetworkBuffer) MakeRspBody(bts []byte, num int, fid int, err error) [][]byte {
	if err != nil {
		msg := []byte(err.Error())
		if len(msg) > n.BodySplit {
			msg = msg[:n.BodySplit]
		}
		var b = make([]byte, 0, n.TotalSize)
		b = append(b, buffPrefix(num, fid, 1, len(msg), BodyRespFailed)...)
		b = append(b, msg...)
		b = append(b, make([]byte, n.BodySplit-len(msg))...)
		b = append(b, 0, 1)
		return [][]byte{b}
	} else if fid == 0 {
		msg := []byte("not found this server api by function id")
		var b = make([]byte, 0, n.TotalSize)
		b = append(b, buffPrefix(num, fid, 1, len(msg), BodyRespNoFound)...)
		b = append(b, msg...)
		b = append(b, make([]byte, n.BodySplit-len(msg))...)
		b = append(b, 0, 1)
		return [][]byte{b}
	} else {
		return n.MakeReqBody(bts, num, fid)
	}
}

func (n NetworkBuffer) parse(b *ConnBody) (*JoinBodyData, error) {
	if len(b.Data) != n.TotalSize {
		return nil, errors.New("request body lenght was wrong")
	} else if b.Data[0] != FirstByte || b.Data[1] != SecondByte ||
		b.Data[len(b.Data)-1] != FirstByte || b.Data[len(b.Data)-2] != SecondByte {
		return nil, errors.New("request body rule was wrong")
	}
	var tmp = &JoinBodyData{
		Uuid: int(b.Data[2])<<16 + int(b.Data[3])<<8 + int(b.Data[4]),
		Func: int(b.Data[6])<<8 + int(b.Data[7]),
	}

	switch b.Data[5] {
	case BodyReqDataNil, BodyRespDataNil:
		tmp.Buck, tmp.Sort = 1, 1
	case BodyWholeData, BodyRespSuccess:
		tmp.Buck, tmp.Sort = 1, 1
		lenght := int(b.Data[10])<<8 + int(b.Data[11])
		tmp.Data = append([]byte{}, b.Data[n.BodyStart:n.BodyStart+lenght]...)

	case BodyRespFailed:
		tmp.Buck, tmp.Sort = 1, 1
		lenght := int(b.Data[10])<<8 + int(b.Data[11])
		msg := string(b.Data[n.BodyStart : n.BodyStart+lenght])
		return tmp, errors.New(msg)

	case BodyBodyStart, BodyRespStart:
		tmp.Buck = int(b.Data[8])<<8 + int(b.Data[9])
		tmp.Sort = 1
		tmp.Data = append([]byte{}, b.Data[n.BodyStart:n.FinalByte]...)
	case BodyReqMiddle, BodyRespMiddle:
		tmp.Buck = int(b.Data[8])<<8 + int(b.Data[9])
		tmp.Sort = int(b.Data[10])<<8 + int(b.Data[11])
		tmp.Data = append([]byte{}, b.Data[n.BodyStart:n.FinalByte]...)
	case BodyReqFinaly, BodyRespFinaly:
		tmp.Buck = int(b.Data[8])<<8 + int(b.Data[9])
		tmp.Sort = tmp.Buck
		lenght := int(b.Data[10])<<8 + int(b.Data[11])
		tmp.Data = append([]byte{}, b.Data[n.BodyStart:n.BodyStart+lenght]...)
	}
	return tmp, nil
}
