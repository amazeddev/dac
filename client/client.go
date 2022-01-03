package client

import (
	"log"
	"net"
	"net/rpc"

	"github.com/spiral/goridge"
)

type Rpc_client struct {
	client *rpc.Client
}

type PidArgs struct{Name string; Pid string}

type DelArgs struct{Key string}

type Info struct{Name string; Pid string}

func (c *Rpc_client) Connect() error {
	conn, err := net.Dial("tcp",  "0.0.0.0:42586")
	if err != nil {
		return err
	}
	c.client = rpc.NewClientWithCodec(goridge.NewClientCodec(conn))
	return nil
}

func (c *Rpc_client) Ping() error {
	_, err := net.Dial("tcp",  "0.0.0.0:42586")
	if err != nil {
		log.Fatal("dialing:", err)
		return err
	}
	return nil
}

func (c Rpc_client) SetPid(args PidArgs) (string, error) {
	var reply string
	err := c.client.Call(
		"KVStore.SetPID", 
		args, 
		&reply,
	)
	if err != nil {
		log.Fatal(err)
		return "", err
	}
	return reply, nil
}

func (c Rpc_client) List(args struct{}) ([]string, error) {
	var reply []string
	err := c.client.Call(
		"KVStore.List", 
		args, 
		&reply,
	)
	if err != nil {
		log.Fatal(err)
		return []string{}, err
	}
	return reply, nil
}

func (c Rpc_client) Delete(args DelArgs) (bool, error) {
	var reply bool
	err := c.client.Call("KVStore.Delete", args, &reply)
	if err != nil {
		log.Fatal(err)
		return false, err
	}
	return reply, nil
}

func (c Rpc_client) Info() (Info, error) {
	var reply Info
	err := c.client.Call("KVStore.Info", struct{}{}, &reply)
	if err != nil {
		panic(err)
	}
	return reply, nil
}