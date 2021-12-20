package grpc

import (
	"fmt"
	"time"

	"google.golang.org/grpc"

	"github.com/gocrane/crane/pkg/log"
	"github.com/gocrane/crane/pkg/utils"
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
		log.Logger().V(5).Info(fmt.Sprintf("Connect using endpoint '%s' with '%s' timeout", v, defaultTimeout))
		addr, dialer, err := utils.GetAddressAndDialer(v)
		if err != nil {
			if idx == (len - 1) {
				return nil, err
			}
			log.Logger().V(5).Info(fmt.Sprintf("Waring: connect using endpoint '%s' failed, err: %s", v, err.Error()))
			continue
		}

		conn, err = grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(defaultTimeout), grpc.WithContextDialer(dialer))
		if err != nil {
			errMsg := fmt.Errorf("connect endpoint '%s', make sure you are running as root and the endpoint has been started", v)
			if idx == (len - 1) {
				return nil, errMsg
			}
			log.Logger().V(5).Info(fmt.Sprintf("Waring: %s", errMsg))
		} else {
			log.Logger().V(5).Info(fmt.Sprintf("Connected successfully using endpoint: %s", v))
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
