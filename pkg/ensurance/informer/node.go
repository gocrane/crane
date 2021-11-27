package informer

import (
	"fmt"

	"golang.org/x/net/context"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/gocrane/crane/pkg/utils"
	"github.com/gocrane/crane/pkg/utils/clogs"
)

// UpdateNodeConditions be used to update node condition with check whether it needs to update
func UpdateNodeConditions(node *v1.Node, condition v1.NodeCondition) (*v1.Node, bool) {

	updatedNode := node.DeepCopy()

	// loop and found the condition type
	for i, cond := range updatedNode.Status.Conditions {
		if cond.Type == condition.Type {
			if cond.Status == condition.Status {
				return updatedNode, false
			} else {
				updatedNode.Status.Conditions[i] = condition
				return updatedNode, true
			}
		}
	}

	// not found the condition, to add the condition to the end
	updatedNode.Status.Conditions = append(updatedNode.Status.Conditions, condition)
	return updatedNode, true
}

// UpdateNodeTaints be used to update node taint
func UpdateNodeTaints(node *v1.Node, taint v1.Taint) (*v1.Node, bool) {

	updatedNode := node.DeepCopy()

	for i, t := range updatedNode.Spec.Taints {
		if t.Key == taint.Key {
			if (t.Value == taint.Value) && (t.Effect == taint.Effect) {
				return updatedNode, false
			} else {
				updatedNode.Spec.Taints[i] = taint
				return updatedNode, true
			}
		}
	}

	// not found the taint, to add the taint
	updatedNode.Spec.Taints = append(updatedNode.Spec.Taints, taint)
	return updatedNode, true
}

// RemoveNodeTaints be used to update node taint
func RemoveNodeTaints(node *v1.Node, taint v1.Taint) (*v1.Node, bool) {
	clogs.Log().Info(fmt.Sprintf("RemoveNodeTaints %v", taint))

	updatedNode := node.DeepCopy()

	var bFound = false
	var taints []v1.Taint

	for _, t := range updatedNode.Spec.Taints {
		clogs.Log().V(4).Info(fmt.Sprintf("taint %s", t.Key))
		if t.Key == taint.Key {
			bFound = true
		} else {
			taints = append(taints, t)
		}
	}

	// found the taint, remove it
	if bFound {
		updatedNode.Spec.Taints = taints
		return updatedNode, true
	}

	return updatedNode, false
}

func FilterNodeConditionByType(conditions []v1.NodeCondition, conditionType string) (v1.NodeCondition, error) {
	for _, cond := range conditions {
		if string(cond.Type) == conditionType {
			return cond, nil
		}
	}

	return v1.NodeCondition{}, fmt.Errorf("condition %s is not found", conditionType)
}

// UpdateNodeStatus be used to update node status by communication with api-server
func UpdateNodeStatus(client clientset.Interface, updateNode *v1.Node, retry *uint64) error {

	for i := uint64(0); i < utils.GetUint64withDefault(retry, defaultRetryTimes); i++ {
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
func UpdateNode(client clientset.Interface, updateNode *v1.Node, retry *uint64) error {
	for i := uint64(0); i < utils.GetUint64withDefault(retry, defaultRetryTimes); i++ {
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
