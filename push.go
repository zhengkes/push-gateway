package push

import (
	"bufio"
	"fmt"
	"github.com/ugorji/go/codec"
	"io"
	"math/rand"
	"net"
	"net/rpc"
	"reflect"
	"sync"
	"time"
)

func rpcPush(addrs []string, metricItems []*metricValue) error {
	var err error
	var items []*metricValue
	now := time.Now().Unix()

	for _, item := range metricItems {
		if item.Endpoint == "" {
			item.Endpoint = config.Remote.Ident
		}
		err = item.CheckValidity(now)
		if err != nil {
			fmt.Println("数据有问题:", err)
			continue
		}
		items = append(items, item)
	}

	count := len(addrs)
	retry := 0
	for {
		for _, i := range rand.Perm(count) {
			addr := addrs[i]
			reply, err := rpcCall(addr, items)
			if err != nil {
				continue
			} else {
				if reply.Msg != "ok" {
					err = fmt.Errorf("some item push err: %s", reply.Msg)
				}
				return err
			}
		}

		time.Sleep(time.Millisecond * 500)

		retry += 1
		if retry == 3 {
			break
		}
	}

	return err
}


func rpcCall(addr string, items []*metricValue) (transferResp, error) {
	var reply transferResp
	var err error

	client := rpcClients.Get(addr)
	if client == nil {
		client, err = rpcClient(addr)
		if err != nil {
			return reply, err
		}
		affected := rpcClients.Put(addr, client)
		if !affected {
			defer func() {
				client.Close()
			}()

		}
	}

	timeout := time.Duration(8) * time.Second
	done := make(chan error, 1)

	go func() {
		err := client.Call("Transfer.Push", items, &reply)
		done <- err
	}()

	select {
	case <-time.After(timeout):
		rpcClients.Put(addr, nil)
		client.Close()
		return reply, fmt.Errorf("%s rpc call timeout", addr)
	case err := <-done:
		if err != nil {
			rpcClients.Del(addr)
			client.Close()
			return reply, fmt.Errorf("%s rpc call done, but fail: %v", addr, err)
		}
	}

	return reply, nil
}

func rpcClient(addr string) (*rpc.Client, error) {
	conn, err := net.DialTimeout("tcp", addr, time.Second*3)
	if err != nil {
		err = fmt.Errorf("dial transfer %s fail: %v", addr, err)
		return nil, err
	}

	var bufConn = struct {
		io.Closer
		*bufio.Reader
		*bufio.Writer
	}{conn, bufio.NewReader(conn), bufio.NewWriter(conn)}

	var mh codec.MsgpackHandle
	mh.MapType = reflect.TypeOf(map[string]interface{}(nil))

	rpcCodec := codec.MsgpackSpecRpc.ClientCodec(bufConn, &mh)
	client := rpc.NewClientWithCodec(rpcCodec)
	return client, nil
}

type transferResp struct {
	Msg     string
	Total   int
	Invalid int
	Latency int64
}

func (t *transferResp) String() string {
	s := fmt.Sprintf("TransferResp total=%d, err_invalid=%d, latency=%dms",
		t.Total, t.Invalid, t.Latency)
	if t.Msg != "" {
		s = fmt.Sprintf("%s, msg=%s", s, t.Msg)
	}
	return s
}

type rpcClientContainer struct {
	M map[string]*rpc.Client
	sync.RWMutex
}

var rpcClients *rpcClientContainer

func initRpcClients() {
	rpcClients = &rpcClientContainer{
		M: make(map[string]*rpc.Client),
	}
}

func (rcc *rpcClientContainer) Get(addr string) *rpc.Client {
	rcc.RLock()
	defer rcc.RUnlock()

	client, has := rcc.M[addr]
	if !has {
		return nil
	}

	return client
}

func (rcc *rpcClientContainer) Put(addr string, client *rpc.Client) bool {
	rcc.Lock()
	defer rcc.Unlock()

	oc, has := rcc.M[addr]
	if has && oc != nil {
		return false
	}

	rcc.M[addr] = client
	return true
}

func (rcc *rpcClientContainer) Del(addr string) {
	rcc.Lock()
	defer rcc.Unlock()
	delete(rcc.M, addr)
}
