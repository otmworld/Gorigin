package rpc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"

	"micro/network/comm"
	"micro/network/pb"
)

const (
	HttpReqSuccessBody = "000" // request success and parse respose body
	HttpReqSuccessNull = "001" // request success and no response data
	HttpReqFailMessage = "002" // request failed and return error message
	HttpReqFailNetwork = "003" // network error, maybe no this api
)

func HttpPingTest(host string, port uint64) error {
	var addr = fmt.Sprintf("http://%s:%d/Ping", host, port)
	_, err := http.Post(addr, "application/octet-stream", nil)
	return err
}

func (n *NodeDetail) httpBuiltIn(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	bts, err := ioutil.ReadAll(r.Body)
	if err != nil {
		makeHttpResp(w, nil, err)
		return
	}
	var req = &comm.CommReq{}
	if err = json.Unmarshal(bts, req); err != nil {
		makeHttpResp(w, nil, err)
		return
	}
	n.builtin(req.Func, &NodeConn{}, req.Data.([]byte))
}

// http: apiname to call
func (n *NodeDetail) httpCall(name string, s *Server, f *ServerFunc) {
	if f.api == pb.ApiType_Multi {
		return
	}
	var function = func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		bts, err := ioutil.ReadAll(r.Body)
		if err != nil {
			makeHttpResp(w, nil, err)
			return
		}
		req, err := UnmarshalValue(s.proto, f.req, bts)
		if err != nil {
			makeHttpResp(w, nil, err)
			return
		}

		if f.api == pb.ApiType_Send {
			err = f.ValueSend(s.rv, req)
			makeHttpResp(w, nil, err)
		} else if f.api == pb.ApiType_Call {
			rsp := reflect.New(f.rsp.Elem())
			if err := f.ValueCall(s.rv, req, rsp); err != nil {
				makeHttpResp(w, nil, err)
			}
			bts, err = MarshalValue(s.proto, rsp)
			makeHttpResp(w, bts, err)
		}
	}
	n.hlist[name] = function
}

func makeHttpResp(w http.ResponseWriter, bts []byte, err error) {
	if err != nil {
		w.Header().Set("code", HttpReqFailMessage)
		w.Write([]byte(err.Error()))
	} else if len(bts) == 0 {
		w.Header().Set("code", HttpReqSuccessNull)
	} else {
		w.Header().Set("code", HttpReqSuccessBody)
		w.Write(bts)
	}
}

// return api error, post error
func postHttpApi(fmsg *pb.FuncMsg, host string, port uint64, body []byte, rsp interface{}) (error, error) {
	var addr = fmt.Sprintf("http://%s:%d/%s", host, port, fmsg.ApiName)
	resp, err := http.Post(addr, "application/octet-stream",
		bytes.NewBuffer(body))
	if err == nil {
		defer resp.Body.Close()
		code := resp.Header.Get("code")

		if code == HttpReqSuccessBody && rsp != nil {
			if body, err = ioutil.ReadAll(resp.Body); err == nil {
				return UnmarshalInterface(fmsg.Protocal, rsp, body), nil
			}
		} else if code == HttpReqSuccessNull {
			return nil, nil
		} else if code == HttpReqFailMessage {
			body, err = ioutil.ReadAll(resp.Body)
			if err == nil && len(body) > 0 {
				return errors.New(string(body)), nil
			}
			return nil, err
		}
	} else {
		return nil, err
	}
	return nil, errors.New("http post server api wrong")
}
