package informer

import (
	"flag"
	"fmt"
	"testing"

	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	// Define global args flags.
	kubeConfig = flag.String("kubeConfig", "", "The kube config path, default: \"\"")
	nodeName   = flag.String("nodeName", "", "The nodeName to filter")
)

func initContextByConfig(kubeConfig string) (*Context, error) {
	var ctx Context
	ctx.kubeConfig = kubeConfig

	err := ctx.ContextInit()
	if err != nil {
		return nil, fmt.Errorf("ContextInit failed %s", err.Error())
	}

	return &ctx, nil
}

func TestInitContext(t *testing.T) {

	flag.Parse()

	if *kubeConfig == "" {
		t.Logf("kubeConfig is empty, skip the test")
	}

	ctx, err := initContextByConfig(*kubeConfig)
	if err != nil {
		t.Fatalf("TestInitContext failed %s", err.Error())
	}

	nodeList, err := ctx.GetKubeClient().CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("TestInitContext get node list failed,err %s", err.Error())
	}

	t.Logf("node items %+v", nodeList.Items)

	t.Logf("TestInitContext succeed")
}

func TestNodeFactory(t *testing.T) {
	flag.Parse()

	if *kubeConfig == "" {
		t.Logf("kubeConfig is empty, skip the test")
	}

	if *nodeName == "" {
		t.Logf("nodeName is empty, skip the test")
	}

	ctx, err := initContextByConfig(*kubeConfig)
	if err != nil {
		t.Fatalf("TestInitContext failed %s", err.Error())
	}

	nodeInformer := ctx.GetNodeFactory().Core().V1().Nodes().Informer()

	stop := make(chan struct{})
	ctx.Run(stop)

	node, err := GetNodeFromInformer(nodeInformer, *nodeName)
	if err != nil {
		t.Fatalf("TestNodeFactory get node failed %s", err.Error())
	}

	t.Logf("node %+v", *node)

	t.Logf("TestNodeFactory succeed")
}

func TestPodFactory(t *testing.T) {
	flag.Parse()

	if *kubeConfig == "" {
		t.Logf("kubeConfig is empty, skip the test")
	}

	ctx, err := initContextByConfig(*kubeConfig)
	if err != nil {
		t.Fatalf("TestInitContext failed %s", err.Error())
	}

	podInformer := ctx.GetPodFactory().Core().V1().Pods().Informer()

	stop := make(chan struct{})
	ctx.Run(stop)

	podList := GetAllPodFromInformer(podInformer)
	if err != nil {
		t.Fatalf("TestPodFactory get node failed %s", err.Error())
	}

	//t.Logf("podList %+v",podList)

	t.Logf("TestPodFactory succeed len(%d)", len(podList))
}

func TestPodFactoryFilterByNodeName(t *testing.T) {

	flag.Parse()

	if *kubeConfig == "" {
		t.Logf("kubeConfig is empty, skip the test")
	}

	if *nodeName == "" {
		t.Logf("nodeName is empty, skip the test")
	}

	ctx, err := initContextByConfig(*kubeConfig)
	if err != nil {
		t.Fatalf("TestInitContext failed %s", err.Error())
	}
	ctx.nodeName = *nodeName

	podInformer := ctx.GetPodFactory().Core().V1().Pods().Informer()

	stop := make(chan struct{})
	ctx.Run(stop)

	podList := GetAllPodFromInformer(podInformer)
	if err != nil {
		t.Fatalf("TestPodFactoryFilterByNodeName get node failed %s", err.Error())
	}

	//t.Logf("podList %+v",podList)

	t.Logf("TestPodFactoryFilterByNodeName succeed len(%d)", len(podList))

}
