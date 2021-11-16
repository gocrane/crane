package grpcc

import (
	"flag"
	"testing"
)

var (
	endpoint = flag.String("endpoint", "unix:///var/run/dockershim.sock", "The runtime endpoint, default: unix:///var/run/dockershim.sock.")
)

func TestGrpcConnection(t *testing.T) {
	flag.Parse()

	t.Logf("TestGrpcConnection endpoint %v,", endpoint)

	runtimeConn, err := InitGrpcConnection([]string{*endpoint})
	if err != nil {
		t.Fatalf("InitGrpcConnection failed %s", err.Error())
	}
	defer CloseGrpcConnection(runtimeConn)

	t.Logf("TestGrpcConnection succeed")
}
