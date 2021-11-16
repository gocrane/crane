package informer

import (
	"flag"
	"testing"
)

func TestEvictPod(t *testing.T) {
	flag.Parse()

	if *kubeConfig == "" {
		t.Logf("kubeConfig is empty, skip the test")
	}

	var podName = "low"
	var namespace = "default"

	ctx, err := initContextByConfig(*kubeConfig)
	if err != nil {
		t.Fatalf("TestEvictPod failed %s", err.Error())
	}

	podInformer := ctx.GetPodFactory().Core().V1().Pods().Informer()

	stop := make(chan struct{})
	ctx.Run(stop)

	pod, err := GetPodFromInformer(podInformer, generateKey(namespace, podName))
	if err != nil {
		t.Fatalf("TestEvictPod failed %s", err.Error())
	}

	err = EvictPodWithGracePeriod(ctx.GetKubeClient(), pod, 30)
	if err != nil {
		t.Fatalf("TestEvictPod failed %s", err.Error())
	}

	t.Logf("TestEvictPod succeed")
}
