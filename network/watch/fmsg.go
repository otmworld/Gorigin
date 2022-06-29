package watch

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"micro/network/comm"
	"micro/network/pb"
)

// server api mapping struct
type funcmap struct {
	ids  sync.Map // key: function-id
	str  sync.Map // key: function-api-name
	num  uint32   // max function id
	name string   // env file path name
	file *os.File
}

//
type funcdata struct {
	msg  *pb.FuncMsg
	api  *pb.FuncApi
	node []string
}

func (f *funcmap) PutMsg(uuid string, arg *pb.FuncMsg) error {
	if arg == nil || uuid == "" || arg.ApiName == "" {
		return errors.New("request data was wrong")
	}
	if data := f.GetStr(arg.ApiName); data != nil {
		arg.FuncID = data.msg.FuncID
		if uuid != "" {
			for _, str := range data.node {
				if str == uuid {
					return nil
				}
			}
			data.node = append(data.node, uuid)
		}
		return nil
	}

	// TODO:  Check master watcher node
	arg.FuncID = atomic.AddUint32(&f.num, 1)
	var tmp = &funcdata{
		msg: arg,
		api: &pb.FuncApi{
			ID: arg.FuncID, Name: arg.ApiName,
			Type: arg.ApiType, Kind: arg.Protocal,
		}, node: []string{uuid}}
	f.ids.Store(arg.FuncID, tmp)
	f.str.Store(arg.ApiName, tmp)
	f.addEnvFile(tmp.msg)
	return nil
}

func (f *funcmap) PutApi(uuid string, arg *pb.FuncApi) error {
	if arg == nil || uuid == "" || arg.Name == "" {
		return errors.New("request data was wrong")
	}
	if data := f.GetStr(arg.Name); data != nil {
		arg.ID = data.msg.FuncID
		if uuid != "" {
			for _, str := range data.node {
				if str == uuid {
					return nil
				}
			}
			data.node = append(data.node, uuid)
		}
		return nil
	}

	// TODO:  Check master watcher node
	arg.ID = atomic.AddUint32(&f.num, 1)
	sname, fname := comm.SplitApiName(arg.Name)
	var tmp = &funcdata{
		msg: &pb.FuncMsg{
			FuncID: arg.ID, ApiName: arg.Name,
			ServName: sname, FuncName: fname,
			ApiType: arg.Type, Protocal: arg.Kind,
		}, api: arg, node: []string{uuid}}
	f.ids.Store(arg.ID, tmp)
	f.str.Store(arg.Name, tmp)
	f.addEnvFile(tmp.msg)
	return nil
}

func (f *funcmap) GetIds(fid uint32) *funcdata {
	if v, ok := f.ids.Load(fid); ok && v != nil {
		return v.(*funcdata)
	}
	return nil
}

func (f *funcmap) GetStr(name string) *funcdata {
	if v, ok := f.str.Load(name); ok && v != nil {
		return v.(*funcdata)
	}
	return nil
}

func (f *funcmap) RangeMsg(function func(*pb.FuncMsg) bool) {
	f.ids.Range(func(key, value interface{}) bool {
		if v, ok := value.(*funcdata); ok && v.msg != nil {
			return function(v.msg)
		}
		return true
	})
}

func (f *funcmap) RangeApi(function func(*pb.FuncApi) bool) {
	f.ids.Range(func(key, value interface{}) bool {
		if v, ok := value.(*funcdata); ok && v.api != nil {
			return function(v.api)
		}
		return true
	})
}

func (f *funcdata) GetMsg() *pb.FuncMsg {
	if f != nil {
		return f.msg
	}
	return nil
}
func (f *funcdata) GetApi() *pb.FuncApi {
	if f != nil {
		return f.api
	}
	return nil
}
func (f *funcdata) GetUuid() []string {
	if f != nil {
		return f.node
	}
	return nil
}

// func (f *funcmap) QuerySerConn(id uint32, name string) []*pb.NodeBaseMsg {
// 	var ids []string
// 	if id != 0 {
// 		ids = f.GetIds(id).GetUuid()
// 	} else if name != "" {
// 		ids = f.GetStr(name).GetUuid()
// 	}

// 	var result = make([]*pb.NodeBaseMsg, 0, len(ids))
// 	for _, id := range ids {
// 		if v, ok := w.uuid.Load(id); ok && v != nil {
// 			if base, ok := v.(*pb.NodeBaseMsg); ok && base != nil {
// 				result = append(result, &pb.NodeBaseMsg{
// 					Pid: base.Pid,
// 					Ver: base.Ver, Uuid: base.Uuid,
// 					Name: base.Name, Main: base.Main,
// 					Host: base.Host, Tport: base.Tport,
// 					Uport: base.Uport, Hport: base.Hport,
// 				})
// 			}
// 		}
// 	}
// 	log.Println(ids, "Get Api Conn Ids", result)
// 	return result
// }

func (f *funcmap) InitEnvFile(path string) error {
	if path == "" {
		f.name = "./funclist.conf"
	} else {
		f.name = filepath.Join(path, "funclist.conf")
	}
	bts, err := ioutil.ReadFile(f.name)
	if err != nil && !os.IsNotExist(err) {
		panic(err)
	} else {
		rows := bytes.Split(bts, []byte("\n"))
		for _, row := range rows {
			if len(row) == 0 {
				continue
			}
			var msg = &pb.FuncApi{}
			if json.Unmarshal(row, msg) == nil && msg.ID > comm.BUILT_IN_MAX {
				f.PutApi("", msg)
				if msg.ID > f.num {
					f.num = uint32(msg.ID)
				}
			}
		}
		if f.num < comm.BUILT_IN_MAX {
			f.num = comm.BUILT_IN_MAX
		}
	}

	os.Remove(f.name)
	f.file, err = os.OpenFile(f.name, os.O_CREATE|os.O_RDWR|os.O_APPEND, os.ModePerm)
	if err == nil {
		f.ids.Range(func(key, value interface{}) bool {
			if bts, err := json.Marshal(value); err == nil {
				f.file.Write(bts)
				f.file.WriteString("\n")
			}
			return true
		})
	}
	return err
}

func (f *funcmap) addEnvFile(fmsg *pb.FuncMsg) {
	if f.file == nil {
		f.upEnvFile()
	}
	if bts, err := json.Marshal(fmsg); err == nil {
		_, err = f.file.Write(append(bts, []byte("\n")...))
		if err != nil {
			f.upEnvFile()
		}
	}
}

func (f *funcmap) upEnvFile() {
	var err error
	f.file, err = os.OpenFile(f.name, os.O_CREATE|os.O_RDWR|os.O_APPEND, os.ModePerm)
	if err != nil {
		log.Panic(err)
	}
	if f.file != nil {
		f.RangeApi(func(arg *pb.FuncApi) bool {
			if bts, err := json.Marshal(arg); err == nil {
				if _, err = f.file.Write(bts); err != nil {
					f.name = "./funclist.bak"
					f.upEnvFile()
					return false
				}
				if _, err = f.file.Write([]byte("\n")); err != nil {
					f.name = "./funclist.bak"
					f.upEnvFile()
					return false
				}
			}
			return true
		})
	}
}
