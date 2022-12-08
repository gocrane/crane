package utils

import (
	"fmt"
	"strings"
	"testing"
)

func TestGetPodNameReg(t *testing.T) {
	var name = "test"
	var kind = "Deployment"

	fmt.Println("pod reg:", GetPodNameReg(name, kind))
}

func TestGetCustomerExpression(t *testing.T) {
	var name = "test"
	var labels = []string{
		"contianer=\"test\"",
		"namespace=\"default\"",
	}
	fmt.Println("CustomerExpression:", GetCustomerExpression(name, strings.Join(labels, ",")))
}

func TestGetWorkloadCpuUsageExpression(t *testing.T) {
	var namespace = "default"
	var name = "test"
	var kind = "Deployment"
	fmt.Println("WorkloadCpuUsageExpression:", GetWorkloadCpuUsageExpression(namespace, name, kind))
}

func TestGetWorkloadMemUsageExpression(t *testing.T) {
	var namespace = "default"
	var name = "test"
	var kind = "Deployment"
	fmt.Println("WorkloadMemUsageExpression:", GetWorkloadMemUsageExpression(namespace, name, kind))
}

func TestGetContainerCpuUsageExpression(t *testing.T) {
	var namespace = "default"
	var name = "test"
	var kind = "Deployment"
	var nameContainer = "test"
	fmt.Println("ContainerCpuUsageExpression:", GetContainerCpuUsageExpression(namespace, name, kind, nameContainer))
}

func TestGetContainerMemUsageExpression(t *testing.T) {
	var namespace = "default"
	var name = "test"
	var kind = "Deployment"
	var nameContainer = "test"
	fmt.Println("ContainerMemUsageExpression:", GetContainerMemUsageExpression(namespace, name, kind, nameContainer))
}

func TestGetPodCpuUsageExpression(t *testing.T) {
	var namespace = "default"
	var name = "test-pod-001"
	fmt.Println("PodCpuUsageExpression:", GetPodCpuUsageExpression(namespace, name))
}

func TestGetPodMemUsageExpression(t *testing.T) {
	var namespace = "default"
	var name = "test-pod-001"
	fmt.Println("PodMemUsageExpression:", GetPodMemUsageExpression(namespace, name))
}

func TestGetNodeCpuUsageExpression(t *testing.T) {
	var node = "test-node-001"
	fmt.Println("NodeCpuUsageExpression:", GetNodeCpuUsageExpression(node))
}

func TestGetNodeMemUsageExpression(t *testing.T) {
	var node = "test-node-001"
	fmt.Println("NodeMemUsageExpression:", GetNodeMemUsageExpression(node))
}
