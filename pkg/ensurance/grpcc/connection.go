package grpcc

import (
	"fmt"
	"time"

	"google.golang.org/grpc"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/kubelet/util"
)

const (
	defaultTimeout = 2 * time.Second
)

func InitGrpcConnection(endPoints []string) (*grpc.ClientConn, error) {
	if len(endPoints) == 0 {
		return nil, fmt.Errorf("endpoint can not be empty")
	}

	var len = len(endPoints)

	var conn *grpc.ClientConn
	for idx, v := range endPoints {
		klog.V(5).Infof("connect using endpoint '%s' with '%s' timeout", v, defaultTimeout)
		addr, dialer, err := util.GetAddressAndDialer(v)
		if err != nil {
			if idx == (len - 1) {
				return nil, err
			}
			klog.Warningf("connect using endpoint '%s' failed, err: %s", v, err.Error())
			continue
		}

		conn, err = grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(defaultTimeout), grpc.WithContextDialer(dialer))
		if err != nil {
			errMsg := fmt.Errorf("connect endpoint '%s', make sure you are running as root and the endpoint has been started", v)
			if idx == (len - 1) {
				return nil, errMsg
			}
			klog.Warningf("connect using endpoint '%s' failed, err: %s", v, errMsg.Error())
		} else {
			klog.V(5).Infof("connected successfully using endpoint: %s", v)
			break
		}
	}
	return conn, nil
}

func CloseGrpcConnection(conn *grpc.ClientConn) error {
	if conn == nil {
		return nil
	}
	return conn.Close()
}
