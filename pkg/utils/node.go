package utils

import (
	"fmt"
	"k8s.io/klog/v2"

	"golang.org/x/net/context"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
)

const defaultRetryTimes = 3

// UpdateNodeConditionsStatues be used to update node condition with check whether it needs to update
func UpdateNodeConditionsStatues(client clientset.Interface, nodeLister corelisters.NodeLister, nodeName string, condition v1.NodeCondition, retry *uint64) (*v1.Node, error) {

	for i := uint64(0); i < GetUint64withDefault(retry, defaultRetryTimes); i++ {
		node, err := nodeLister.Get(nodeName)
		if err != nil {
			return nil, err
		}

		updateNode, needUpdate := updateNodeConditions(node, condition)
		if needUpdate {
			klog.Warningf("Updating node condition %v", condition)
			if updateNode, err = client.CoreV1().Nodes().UpdateStatus(context.Background(), updateNode, metav1.UpdateOptions{}); err != nil {
				if errors.IsConflict(err) {
					continue
				} else {
					return nil, err
				}
			}
		}

		return updateNode, nil
	}

	return nil, fmt.Errorf("update node failed, conflict too more times")
}

func updateNodeConditions(node *v1.Node, condition v1.NodeCondition) (*v1.Node, bool) {
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

// UpdateNodeTaints be used to update node taints with check whether it needs to update
func UpdateNodeTaints(client clientset.Interface, nodeLister corelisters.NodeLister, nodeName string, taint v1.Taint, retry *uint64) (*v1.Node, error) {

	for i := uint64(0); i < GetUint64withDefault(retry, defaultRetryTimes); i++ {
		node, err := nodeLister.Get(nodeName)
		if err != nil {
			return nil, err
		}

		updateNode, needUpdate := updateNodeTaints(node, taint)
		if needUpdate {
			if updateNode, err = client.CoreV1().Nodes().Update(context.Background(), updateNode, metav1.UpdateOptions{}); err != nil {
				if errors.IsConflict(err) {
					continue
				} else {
					return nil, err
				}
			}
		}

		return updateNode, nil
	}

	return nil, fmt.Errorf("failed to update node taints after %d retries", GetUint64withDefault(retry, defaultRetryTimes))
}

func updateNodeTaints(node *v1.Node, taint v1.Taint) (*v1.Node, bool) {
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

func RemoveNodeTaints(client clientset.Interface, nodeLister corelisters.NodeLister, nodeName string, taint v1.Taint, retry *uint64) (*v1.Node, error) {
	for i := uint64(0); i < GetUint64withDefault(retry, defaultRetryTimes); i++ {
		node, err := nodeLister.Get(nodeName)
		if err != nil {
			return nil, err
		}

		updateNode, needUpdate := removeNodeTaints(node, taint)
		if needUpdate {
			klog.V(4).Infof("Removing node taint %v", taint)
			if updateNode, err = client.CoreV1().Nodes().Update(context.Background(), updateNode, metav1.UpdateOptions{}); err != nil {
				if errors.IsConflict(err) {
					continue
				} else {
					return nil, err
				}
			}
		}

		return updateNode, nil
	}

	return nil, fmt.Errorf("update node failed, conflict too more times")
}

func removeNodeTaints(node *v1.Node, taint v1.Taint) (*v1.Node, bool) {

	updatedNode := node.DeepCopy()

	var foundTaint = false
	var taints []v1.Taint

	for _, t := range updatedNode.Spec.Taints {
		if t.Key == taint.Key && t.Effect == taint.Effect {
			foundTaint = true
		} else {
			taints = append(taints, t)
		}
	}

	// found the taint, remove it
	if foundTaint {
		updatedNode.Spec.Taints = taints
		return updatedNode, true
	}

	return updatedNode, false
}
