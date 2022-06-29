package rpc

import (
	"errors"
	"fmt"
	"reflect"

	"micro/network/pb"

	"google.golang.org/protobuf/proto"
)

var typeOfError = reflect.TypeOf((*error)(nil)).Elem()

type Server struct {
	rt    reflect.Type
	rv    reflect.Value
	proto pb.Compiler
	sname string                 // 结构名
	funcs map[string]*ServerFunc // 结构方法
}

type ServerFunc struct {
	method reflect.Method
	fname  string         // function name
	api    pb.ApiType     // send request, not response return
	req    reflect.Type   // request data type
	rsp    reflect.Type   // response data type
	args   []reflect.Type // multi request data types
}

// Register server struct with function call
func (n *NodeDetail) Register(argv interface{}) error {
	server := &Server{
		proto: pb.Compiler_PROTO,
		rt:    reflect.TypeOf(argv),
		rv:    reflect.ValueOf(argv),
		funcs: make(map[string]*ServerFunc),
	}
	server.sname = reflect.Indirect(server.rv).Type().Name()

	// Parse the methods
	if err := server.parseFunc(); err != nil {
		return err
	}
	n.funcs[server.sname] = server

	for _, s := range server.funcs {
		n.Funcs = append(n.Funcs, &pb.FuncApi{
			Name: server.sname + "." + s.fname,
			Type: s.api,
			Kind: server.proto,
		})
		if n.Hport != 0 {
			n.httpCall(server.sname+"."+s.fname, server, s)
		}
	}
	return nil
}

func (n *NodeDetail) findCall(fmsg *pb.FuncMsg, bts []byte) ([]byte, error) {
	s, ok := n.funcs[fmsg.ServName]
	if !ok {
		return nil, fmt.Errorf("not found server: %s by local", fmsg.ServName)
	}
	if f, ok := s.funcs[fmsg.FuncName]; ok {
		switch f.api {
		case pb.ApiType_Send:
			if req, err := UnmarshalValue(s.proto, f.req, bts); err != nil {
				return nil, err
			} else {
				return nil, f.ValueSend(s.rv, req)
			}

		case pb.ApiType_Call:
			rsp := reflect.New(f.rsp.Elem())
			if req, err := UnmarshalValue(s.proto, f.req, bts); err != nil {
				return nil, err
			} else if err = f.ValueCall(s.rv, req, rsp); err != nil {
				return nil, err
			} else {
				return MarshalValue(s.proto, rsp)
			}

		case pb.ApiType_Multi:
			var req = &pb.MultiBody{}
			if err := proto.Unmarshal(bts, req); err != nil {
				return nil, err
			} else if req.Count != uint32(len(f.args)) {
				return nil, errors.New("request args number were wrong")
			} else {
				var reqdata = []reflect.Value{s.rv}
				for i, arg := range f.args {
					if v, err := UnmarshalValue(s.proto, arg, req.Data[i]); err != nil {
						return nil, err
					} else {
						reqdata = append(reqdata, v)
					}
				}
				rsp := reflect.New(f.rsp.Elem())
				reqdata = append(reqdata, rsp)
				if err = f.ValueMulti(reqdata); err == nil {
					return MarshalValue(s.proto, rsp)
				} else {
					return nil, err
				}
			}
		}
	}
	return nil, errors.New("not found server function: " + fmsg.FuncName)
}

// parse function method type
func (s *Server) parseFunc() error {
	for i := 0; i < s.rt.NumMethod(); i++ {
		method := s.rt.Method(i)

		// Method must be exported.
		// If function is protocal set, update
		if s.checkCompilerProtocal(method.Name) {
			continue
		}

		var tmp = &ServerFunc{fname: method.Name, method: method}
		// Parse function num in : request and response interfase
		if err := tmp.parseMethodsNumIn(); err != nil {
			return err
		}
		s.funcs[tmp.fname] = tmp
	}
	return nil
}

// check function struct use compiler protocal
func (s *Server) checkCompilerProtocal(protocal string) bool {
	if protocal, ok := CheckProtocal(protocal); ok {
		s.proto = protocal
		return true
	}
	return false
}

// LocalSend : local server call
func (server *ServerFunc) LocalSend(rv reflect.Value, req interface{}) error {
	result := server.method.Func.Call([]reflect.Value{
		rv, reflect.ValueOf(req)})
	if len(result) != 1 {
		return errors.New("call result value length wrong")
	}
	if result[0].Interface() != nil {
		return result[0].Interface().(error)
	}
	return nil
}

// RawCall : local server call
func (server *ServerFunc) LocalCall(rv reflect.Value, req, rsp interface{}) error {
	result := server.method.Func.Call([]reflect.Value{
		rv, reflect.ValueOf(req), reflect.ValueOf(rsp)})
	if len(result) != 1 {
		return errors.New("call result value length wrong")
	}
	if result[0].Interface() != nil {
		return result[0].Interface().(error)
	}
	return nil
}

// RawCall : local server call
func (server *ServerFunc) LocalMulti(rv reflect.Value, args ...interface{}) error {
	var rows = []reflect.Value{rv}
	for _, arg := range rows {
		rows = append(rows, reflect.Value(arg))
	}
	result := server.method.Func.Call(rows)
	if len(result) != 1 {
		return errors.New("call result value length wrong")
	}
	if result[0].Interface() != nil {
		return result[0].Interface().(error)
	}
	return nil
}

// ValueSend : remote server send
func (server *ServerFunc) ValueSend(rv, req reflect.Value) error {
	return server.ValueMulti([]reflect.Value{rv, req})
}

// ValueCall : remote server call
func (server *ServerFunc) ValueCall(rv reflect.Value, req, rsp reflect.Value) error {
	return server.ValueMulti([]reflect.Value{rv, req, rsp})
}

// ValueCall : remote server call
func (server *ServerFunc) ValueMulti(args []reflect.Value) error {
	result := server.method.Func.Call(args)
	if len(result) != 1 {
		return errors.New("call result value length wrong")
	}
	if result[0].Interface() != nil {
		return result[0].Interface().(error)
	}
	return nil
}

// parse function in value with request and response interfase
func (s *ServerFunc) parseMethodsNumIn() error {
	method := s.method.Type

	// Check function run in values
	switch method.NumIn() {
	case 0, 1:
		return fmt.Errorf("method NumIn: %d parse was wrong", method.NumIn())
	case 2:
		s.req, s.api = method.In(1), pb.ApiType_Send
		if s.req.Kind() != reflect.Ptr && s.req.Kind() != reflect.Interface {
			return fmt.Errorf("method request in Kind: %v not ptr or interface", s.req.Kind())
		}
	case 3:
		s.req, s.rsp, s.api = method.In(1), method.In(2), pb.ApiType_Call
		if s.req.Kind() != reflect.Ptr && s.req.Kind() != reflect.Interface {
			return fmt.Errorf("method request in Kind: %v not ptr or interface", s.req.Kind())
		}
		if s.rsp.Kind() != reflect.Ptr && s.rsp.Kind() != reflect.Interface {
			return fmt.Errorf("method response in Kind: %v not ptr or interface", s.rsp.Kind())
		}
	default:
		for i := 1; i < method.NumIn()-1; i++ {
			arg := method.In(i)
			if arg.Kind() != reflect.Ptr && arg.Kind() != reflect.Interface {
				return fmt.Errorf("method request in Kind: %v not ptr or interface", s.req.Kind())
			}
			s.args = append(s.args, arg)
		}
		s.rsp, s.api = method.In(method.NumIn()-1), pb.ApiType_Multi
		if s.rsp.Kind() != reflect.Ptr && s.rsp.Kind() != reflect.Interface {
			return fmt.Errorf("method response in Kind: %v not ptr or interface", s.rsp.Kind())
		}
	}

	// Check function return values
	if method.NumOut() == 1 {
		if method.Out(0) != typeOfError {
			return fmt.Errorf("rpc.Register: return type of method %s is %v, must be error", s.fname, method.Out(0))
		}
	} else {
		return fmt.Errorf("rpc.Register: method %s has %d output parameters; needs exactly one", s.fname, method.NumOut())
	}
	return nil
}
