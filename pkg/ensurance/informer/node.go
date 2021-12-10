package informer

import (
	"fmt"

	"golang.org/x/net/context"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

const (
	NodeUnscheduledLow v1.NodeConditionType = "NodeUnscheduledLow"
)

// updateNodeConditions be used to update node condition
func updateNodeConditions(node *v1.Node, condition v1.NodeCondition) (*v1.Node, error) {
	if node == nil {
		return nil, fmt.Errorf("updateNodeConditions node is empty")
	}

	updatedNode := node.DeepCopy()

	conditions, err := GetNodeConditions(updatedNode)
	if err != nil {
		return nil, err
	}

	var bFound = false
	for i, cond := range conditions {
		if cond.Type == condition.Type {
			conditions[i] = condition
			bFound = true
		}
	}

	if !bFound {
		conditions = append(conditions, condition)
	}

	updatedNode.Status.Conditions = conditions

	return updatedNode, nil
}

// updateNodeTaints be used to update node taint
func updateNodeTaints(node *v1.Node, taint v1.Taint) (*v1.Node, error) {
	if node == nil {
		return nil, fmt.Errorf("updateNodeTaints node is empty")
	}

	updatedNode := node.DeepCopy()

	taints, err := GetNodeTaints(updatedNode)
	if err != nil {
		return nil, err
	}

	var bFound = false
	for i, t := range taints {
		if t.Key == taint.Key {
			taints[i] = taint
			bFound = true
		}
	}

	if !bFound {
		taints = append(taints, taint)
	}

	updatedNode.Spec.Taints = taints

	return updatedNode, nil
}

func GetNodeConditions(node *v1.Node) ([]v1.NodeCondition, error) {

	if node == nil {
		return []v1.NodeCondition{}, fmt.Errorf("node resource is empty")
	}

	return node.Status.Conditions, nil
}

func GetNodeTaints(node *v1.Node) ([]v1.Taint, error) {

	if node == nil {
		return []v1.Taint{}, fmt.Errorf("node resource is empty")
	}

	return node.Spec.Taints, nil
}

func FilterNodeConditionByType(conditions []v1.NodeCondition, conditionType string) (v1.NodeCondition, error) {
	for _, cond := range conditions {
		if string(cond.Type) == conditionType {
			return cond, nil
		}
	}

	return v1.NodeCondition{}, fmt.Errorf("condition %s is not found", conditionType)
}

// updateNodeStatus be used to update node status by communication with api-server
func updateNodeStatus(client clientset.Interface, updateNode *v1.Node, retry *uint64) error {

	for i := uint64(0); i < getRetryTimes(retry); i++ {
		_, err := client.CoreV1().Nodes().UpdateStatus(context.Background(), updateNode, metav1.UpdateOptions{})
		if err != nil {
			if errors.IsConflict(err) {
				continue
			} else {
				return err
			}
		}

		return nil

	}

	return fmt.Errorf("update node status failed, conflict too more times")
}

// updateNode be used to update node  by communication with api-server
func updateNode(client clientset.Interface, updateNode *v1.Node, retry *uint64) error {
	for i := uint64(0); i < getRetryTimes(retry); i++ {
		_, err := client.CoreV1().Nodes().Update(context.Background(), updateNode, metav1.UpdateOptions{})
		if err != nil {
			if errors.IsConflict(err) {
				continue
			} else {
				return err
			}
		}

		return nil
	}

	return fmt.Errorf("update node failed, conflict too more times")
}

func getRetryTimes(retry *uint64) uint64 {
	if retry == nil {
		return defaultRetryTimes
	}
	return *retry
}

func GetNodeFromInformer(nodeInformer cache.SharedIndexInformer, nodeName string) (*v1.Node, error) {
	obj, exited, err := nodeInformer.GetStore().GetByKey(nodeName)
	if err != nil {
		return nil, err
	}

	if !exited {
		return nil, fmt.Errorf("node(%s) not found", nodeName)
	}

	// re-assign new node info
	return obj.(*v1.Node), nil
}
