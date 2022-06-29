package rpc

import (
	"encoding/json"
	"errors"
	"reflect"

	"micro/network/pb"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func CheckProtocal(str string) (pb.Compiler, bool) {
	switch str {
	case "Compiler_JSON":
		return pb.Compiler_JSON, true
	case "Compiler_JSONPB":
		return pb.Compiler_JSONPB, true
	case "Compiler_PROTO":
		return pb.Compiler_PROTO, true
	}
	return pb.Compiler_PROTO, false
}

func UnmarshalValue(protocal pb.Compiler, argv reflect.Type, data []byte) (reflect.Value, error) {
	tmp := reflect.New(argv.Elem())
	var err error
	switch protocal {
	case pb.Compiler_PROTO:
		if reqType, ok := tmp.Interface().(proto.Message); ok {
			err = proto.Unmarshal(data, reqType)
		} else {
			err = errors.New("not implement proto.Message")
		}
	case pb.Compiler_JSON:
		err = json.Unmarshal(data, tmp.Interface())
	case pb.Compiler_JSONPB:
		if reqType, ok := tmp.Interface().(proto.Message); ok {
			err = protojson.Unmarshal(data, reqType)
		} else {
			err = errors.New("not implement proto.Message")
		}
	default:
		err = errors.New("undefind compiler protocal")
	}
	return tmp, err
}

func MarshalValue(protocal pb.Compiler, data reflect.Value) ([]byte, error) {
	switch protocal {
	case pb.Compiler_PROTO, pb.Compiler_JSONPB:
		if rspType, ok := data.Interface().(proto.Message); ok {
			return proto.Marshal(rspType)
		}
	case pb.Compiler_JSON:
		return json.Marshal(data.Interface())
	default:
		return nil, errors.New("undefind compiler protocal")
	}
	return nil, errors.New("marsha1 response value wrong")
}

func UnmarshalInterface(protocal pb.Compiler, argv interface{}, data []byte) error {
	if reflect.TypeOf(argv).Kind() != reflect.Ptr {
		return errors.New("rsp interface not ptr")
	}
	switch protocal {
	case pb.Compiler_PROTO:
		if v, ok := argv.(proto.Message); ok {
			return proto.Unmarshal(data, v)
		} else {
			return errors.New("not implement proto.Message")
		}
	case pb.Compiler_JSON:
		return json.Unmarshal(data, argv)
	case pb.Compiler_JSONPB:
		if reqType, ok := argv.(proto.Message); ok {
			return protojson.Unmarshal(data, reqType)
		} else {
			return errors.New("not implement proto.Message")
		}
	default:
		return errors.New("undefind compiler protocal")
	}
}

func MarshalInterface(protocal pb.Compiler, data interface{}) ([]byte, error) {
	switch protocal {
	case pb.Compiler_PROTO, pb.Compiler_JSONPB:
		if v, ok := data.(proto.Message); ok {
			return proto.Marshal(v)
		}
	case pb.Compiler_JSON:
		return json.Marshal(data)
	default:
		return nil, errors.New("undefind compiler protocal")
	}
	return nil, errors.New("marsha1 response value wrong")
}
