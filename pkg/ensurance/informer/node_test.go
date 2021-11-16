package informer

import (
	"flag"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"
)

type nodeConditionStruct struct {
	inputCondition  v1.NodeCondition
	outputCondition v1.NodeCondition
}

func TestNodeConditionsUpdate(t *testing.T) {
	flag.Parse()

	if *kubeConfig == "" {
		t.Logf("kubeConfig is empty, skip the test")
	}

	if *nodeName == "" {
		t.Logf("nodeName is empty, skip the test")
	}

	ctx, err := initContextByConfig(*kubeConfig)
	if err != nil {
		t.Fatalf("TestNodeConditionsUpdateTrue failed %s", err.Error())
	}

	nodeInformer := ctx.GetNodeFactory().Core().V1().Nodes().Informer()

	stop := make(chan struct{})
	ctx.Run(stop)

	var now = metav1.Now()
	var cases = []nodeConditionStruct{
		{
			inputCondition:  v1.NodeCondition{Type: NodeUnscheduledLow, Status: v1.ConditionTrue, LastTransitionTime: now},
			outputCondition: v1.NodeCondition{Type: NodeUnscheduledLow, Status: v1.ConditionTrue, LastTransitionTime: now},
		},
		{
			inputCondition:  v1.NodeCondition{Type: NodeUnscheduledLow, Status: v1.ConditionFalse, LastTransitionTime: now},
			outputCondition: v1.NodeCondition{Type: NodeUnscheduledLow, Status: v1.ConditionFalse, LastTransitionTime: now},
		},
	}

	for idx, c := range cases {
		node, err := GetNodeFromInformer(nodeInformer, *nodeName)
		if err != nil {
			t.Fatalf("TestNodeConditionsUpdate get node failed %s", err.Error())
		}

		updateNode, err := updateNodeConditions(node, c.inputCondition)
		if err != nil {
			t.Fatalf("TestNodeConditionsUpdate updateNodeConditions failed %s", err.Error())
		}

		err = updateNodeStatus(ctx.kubeClient, updateNode, nil)
		if err != nil {
			t.Fatalf("TestNodeConditionsUpdate updateNodeStatus failed %s", err.Error())
		}

		time.Sleep(time.Second)

		nodeRefreshed, err := GetNodeFromInformer(nodeInformer, *nodeName)
		if err != nil {
			t.Fatalf("TestNodeConditionsUpdate get nodeRefreshed failed %s", err.Error())
		}

		conditions, err := GetNodeConditions(nodeRefreshed)
		if err != nil {
			t.Fatalf("TestNodeConditionsUpdate GetNodeConditions from nodeRefreshed failed %s", err.Error())
		}

		cond, err := FilterNodeConditionByType(conditions, string(c.outputCondition.Type))
		if err != nil {
			t.Fatalf("TestNodeConditionsUpdate FilterNodeConditionByType from nodeRefreshed failed %s", err.Error())
		}

		if c.outputCondition.Status != cond.Status {
			t.Fatalf("the outputCondition status is %v, the nodeRefreshed condition status is %v, not equal",
				c.outputCondition.Status, cond.Status)
		}

		t.Logf("NodeConditionsUpdate[%d] succeed", idx)
	}

	t.Logf("TestNodeConditionsUpdate succeed")
}
