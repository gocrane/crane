package runtime

import (
	"flag"
	"testing"

	"github.com/gocrane-io/crane/pkg/ensurance/grpcc"
)

var (
	// Define global args flags.
	runtimeEndpoint      = flag.String("runtimeEndpoint", "unix:///var/run/dockershim.sock", "The runtime endpoint, default: unix:///var/run/dockershim.sock.")
	runtimeEndpointIsSet = flag.Bool("runtimeEndpointIsSet", true, "The runtime endpoint set, default: true")
	imageEndpoint        = flag.String("imageEndpoint", "unix:///var/run/dockershim.sock", "The image endpoint, default: unix:///var/run/dockershim.sock.")
	imageEndpointIsSet   = flag.Bool("imageEndpointIsSet", true, "The image endpoint set, default: true")
	containerId          = flag.String("containerId", "", "The container id, default: \"\"")
)

type UpdateResourceInput struct {
	UpdateOptions
}

type ListContainersInput struct {
	ListOptions
}

func TestInitRuntimeClient(t *testing.T) {
	flag.Parse()

	t.Logf("TestInitRuntimeClient runtimeEndpoint %v, runtimeEndpointIsSet %v, containerId %v",
		runtimeEndpoint, runtimeEndpointIsSet, containerId)

	runtimeClient, runtimeConn, err := GetRuntimeClient(*runtimeEndpoint, *runtimeEndpointIsSet)
	if err != nil {
		t.Fatalf("TestInitRuntimeClient GetRuntimeClient failed %s", err.Error())
	}
	defer grpcc.CloseGrpcConnection(runtimeConn)

	t.Logf("runtimeClient %v", runtimeClient)

	t.Logf("TestUpdateResource succeed")
}

func TestInitImageClient(t *testing.T) {
	flag.Parse()

	t.Logf("TestInitImageClient imageEndpoint %v, imageEndpointIsSet %v, containerId %v",
		imageEndpoint, imageEndpointIsSet, containerId)

	imageClient, imageConn, err := GetImageClient(*imageEndpoint, *imageEndpointIsSet)
	if err != nil {
		t.Fatalf("TestInitImageClient GetRuntimeClient failed %s", err.Error())
	}
	defer grpcc.CloseGrpcConnection(imageConn)

	t.Logf("imageClient %v", imageClient)
	t.Logf("TestInitImageClient succeed")
}

func TestUpdateResource(t *testing.T) {
	flag.Parse()

	t.Logf("TestUpdateResource runtimeEndpoint %v, runtimeEndpointIsSet %v, containerId %v",
		runtimeEndpoint, runtimeEndpointIsSet, containerId)

	if *containerId == "" {
		t.Logf("TestUpdateResource containerId is empty")
		return
	}

	runtimeClient, runtimeConn, err := GetRuntimeClient(*runtimeEndpoint, *runtimeEndpointIsSet)
	if err != nil {
		t.Fatalf("TestUpdateResource GetRuntimeClient failed %s", err.Error())
	}
	defer grpcc.CloseGrpcConnection(runtimeConn)

	var cases = []UpdateResourceInput{
		{
			UpdateOptions: UpdateOptions{
				CPUQuota: int64(100000),
			},
		},
	}

	for _, c := range cases {
		err := UpdateContainerResources(runtimeClient, *containerId, c.UpdateOptions)
		if err != nil {
			t.Fatalf("TestUpdateResource failed %s", err.Error())
		}
	}

	t.Logf("TestUpdateResource succeed")
}

func TestListContainers(t *testing.T) {
	flag.Parse()

	t.Logf("TestListContainers runtimeEndpoint %v, runtimeEndpointIsSet %v",
		runtimeEndpoint, runtimeEndpointIsSet)

	runtimeClient, runtimeConn, err := GetRuntimeClient(*runtimeEndpoint, *runtimeEndpointIsSet)
	if err != nil {
		t.Fatalf("TestUpdateResource GetRuntimeClient failed %s", err.Error())
	}
	defer grpcc.CloseGrpcConnection(runtimeConn)

	var cases = []ListContainersInput{
		{
			ListOptions: ListOptions{},
		},
	}

	for _, c := range cases {
		containers, err := ListContainers(runtimeClient, c.ListOptions)
		if err != nil {
			t.Fatalf("TestListContainers failed %s", err.Error())
		}

		t.Logf("containers %v", containers)
	}

	t.Logf("TestListContainers succeed")
}
