package runtime

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	criapis "k8s.io/cri-api/pkg/apis"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	"k8s.io/klog/v2"
	criremote "k8s.io/kubernetes/pkg/kubelet/cri/remote"
	"k8s.io/kubernetes/pkg/kubelet/util"

	gprcconnection "github.com/gocrane/crane/pkg/ensurance/grpc"
)

// runtimeEndpoint is CRI server runtime endpoint
func getRuntimeClientConnection(runtimeEndpoint string) (*grpc.ClientConn, error) {
	var runtimeEndpoints []string
	if runtimeEndpoint != "" {
		runtimeEndpoints = append(runtimeEndpoints, runtimeEndpoint)
	}
	runtimeEndpoints = append(runtimeEndpoints, defaultRuntimeEndpoints...)
	klog.V(2).Infof("Runtime connect using endpoints: %v. You can set the endpoint instead.", defaultRuntimeEndpoints)
	return gprcconnection.InitGrpcConnection(runtimeEndpoints)
}

// imageEndpoint is CRI server image endpoint, default same as runtime endpoint
func getImageClientConnection(imageEndpoint string) (*grpc.ClientConn, error) {
	var imageEndpoints []string
	if imageEndpoint != "" {
		imageEndpoints = append(imageEndpoints, imageEndpoint)
	}
	imageEndpoints = append(imageEndpoints, defaultRuntimeEndpoints...)
	klog.V(2).Infof(fmt.Sprintf("Image connect using endpoints: %v. You should set the endpoint instead.", imageEndpoints))
	return gprcconnection.InitGrpcConnection(imageEndpoints)
}

// GetRuntimeClient get the runtime client
func GetRuntimeClient(runtimeEndpoint string) (pb.RuntimeServiceClient, *grpc.ClientConn, error) {
	// Set up a connection to the server.
	conn, err := getRuntimeClientConnection(runtimeEndpoint)
	if err != nil {
		return nil, nil, err
	}
	runtimeClient := pb.NewRuntimeServiceClient(conn)
	return runtimeClient, conn, nil
}

// GetImageClient get the runtime client
func GetImageClient(imageEndpoint string) (pb.ImageServiceClient, *grpc.ClientConn, error) {
	// Set up a connection to the server.
	conn, err := getImageClientConnection(imageEndpoint)
	if err != nil {
		return nil, nil, err
	}
	imageClient := pb.NewImageServiceClient(conn)
	return imageClient, conn, nil
}

// GetCRIRuntimeService get the CRI runtime service.
func GetCRIRuntimeService(runtimeEndpoint string) (criapis.RuntimeService, error) {
	var runtimeEndpoints = defaultRuntimeEndpoints
	if runtimeEndpoint != "" {
		runtimeEndpoints = []string{runtimeEndpoint}
	}
	klog.V(2).Infof("Runtime connect using endpoints: %v", runtimeEndpoints)

	for _, endpoint := range runtimeEndpoints {
		err := dialRemoteRuntime(endpoint, 3*time.Second)
		if err != nil {
			klog.Warningf("Failed to connect to remote runtime service: %v", err)
			continue
		}
		klog.V(2).Infof("Runtime connect using endpoint: %v", endpoint)
		containerRuntime, err := criremote.NewRemoteRuntimeService(endpoint, 3*time.Second)
		if err != nil {
			klog.Fatalf("Failed to connect to remote runtime service: %v", err)
		}
		return containerRuntime, nil
	}
	return nil, fmt.Errorf("failed to connect to remote runtime service")
}

// copy from `criremote.NewRemoteRuntimeService` but dial endpoint in block mode
func dialRemoteRuntime(endpoint string, connectionTimeout time.Duration) error {
	klog.V(3).InfoS("Connecting to runtime service", "endpoint", endpoint)
	addr, dialer, err := util.GetAddressAndDialer(endpoint)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), connectionTimeout)
	defer cancel()

	// use block mode to wait for connection
	conn, err := grpc.DialContext(ctx, addr, grpc.WithInsecure(), grpc.WithContextDialer(dialer), grpc.WithBlock())
	if err != nil {
		klog.ErrorS(err, "Connect remote runtime failed", "address", addr)
		return err
	}
	defer conn.Close()
	return nil
}
