package utils

import (
	"strings"
	"testing"
)

func TestGetPodNameReg(t *testing.T) {
	test := struct {
		description string
		name        string
		kind        string
		expect      string
	}{
		description: "GetPodNameReg",
		name:        "test",
		kind:        "Deployment",
		expect:      "^test-[a-z0-9]+-[a-z0-9]{5}$",
	}

	requests := GetPodNameReg(test.name, test.kind)
	if requests != test.expect {
		t.Errorf("expect requests %s actual requests %s", test.expect, requests)
	}
}

func TestGetCustomerExpression(t *testing.T) {
	test := struct {
		description string
		name        string
		labels      []string
		expect      string
	}{
		description: "GetCustomerExpression",
		name:        "test",
		labels: []string{
			"container=\"test\"",
			"namespace=\"default\"",
		},
		expect: "sum(test{container=\"test\",namespace=\"default\"})",
	}

	requests := GetCustomerExpression(test.name, strings.Join(test.labels, ","))
	if requests != test.expect {
		t.Errorf("expect requests %s actual requests %s", test.expect, requests)
	}
}

func TestGetWorkloadCpuUsageExpression(t *testing.T) {
	test := struct {
		description string
		namespace   string
		name        string
		kind        string
		expect      string
	}{
		description: "GetWorkloadCpuUsageExpression",
		namespace:   "default",
		name:        "test",
		kind:        "Deployment",
		expect:      "sum(irate(container_cpu_usage_seconds_total{namespace=\"default\",pod=~\"^test-[a-z0-9]+-[a-z0-9]{5}$\",container!=\"\"}[3m]))",
	}

	requests := GetWorkloadCpuUsageExpression(test.namespace, test.name, test.kind)
	if requests != test.expect {
		t.Errorf("expect requests %s actual requests %s", test.expect, requests)
	}
}

func TestGetWorkloadMemUsageExpression(t *testing.T) {
	test := struct {
		description string
		namespace   string
		name        string
		kind        string
		expect      string
	}{
		description: "GetWorkloadMemUsageExpression",
		namespace:   "default",
		name:        "test",
		kind:        "Deployment",
		expect:      "sum(container_memory_working_set_bytes{namespace=\"default\",pod=~\"^test-[a-z0-9]+-[a-z0-9]{5}$\",container!=\"\"})",
	}

	requests := GetWorkloadMemUsageExpression(test.namespace, test.name, test.kind)
	if requests != test.expect {
		t.Errorf("expect requests %s actual requests %s", test.expect, requests)
	}
}

func TestGetContainerCpuUsageExpression(t *testing.T) {
	test := struct {
		description   string
		namespace     string
		name          string
		kind          string
		nameContainer string
		expect        string
	}{
		description:   "GetContainerCpuUsageExpression",
		namespace:     "default",
		name:          "test",
		kind:          "Deployment",
		nameContainer: "test",
		expect:        "irate(container_cpu_usage_seconds_total{container!=\"POD\",namespace=\"default\",pod=~\"^test-[a-z0-9]+-[a-z0-9]{5}$\",container=\"test\"}[3m])",
	}

	requests := GetContainerCpuUsageExpression(test.namespace, test.name, test.kind, test.nameContainer)
	if requests != test.expect {
		t.Errorf("expect requests %s actual requests %s", test.expect, requests)
	}
}

func TestGetContainerMemUsageExpression(t *testing.T) {
	test := struct {
		description   string
		namespace     string
		name          string
		kind          string
		nameContainer string
		expect        string
	}{
		description:   "GetContainerMemUsageExpression",
		namespace:     "default",
		name:          "test",
		kind:          "Deployment",
		nameContainer: "test",
		expect:        "container_memory_working_set_bytes{container!=\"POD\",namespace=\"default\",pod=~\"^test-[a-z0-9]+-[a-z0-9]{5}$\",container=\"test\"}",
	}

	requests := GetContainerMemUsageExpression(test.namespace, test.name, test.kind, test.nameContainer)
	if requests != test.expect {
		t.Errorf("expect requests %s actual requests %s", test.expect, requests)
	}
}

func TestGetPodCpuUsageExpression(t *testing.T) {
	test := struct {
		description string
		namespace   string
		name        string
		expect      string
	}{
		description: "GetPodCpuUsageExpression",
		namespace:   "default",
		name:        "test-pod-001",
		expect:      "sum(irate(container_cpu_usage_seconds_total{container!=\"POD\",namespace=\"default\",pod=\"test-pod-001\"}[3m]))",
	}

	requests := GetPodCpuUsageExpression(test.namespace, test.name)
	if requests != test.expect {
		t.Errorf("expect requests %s actual requests %s", test.expect, requests)
	}
}

func TestGetPodMemUsageExpression(t *testing.T) {
	test := struct {
		description string
		namespace   string
		name        string
		expect      string
	}{
		description: "GetPodMemUsageExpression",
		namespace:   "default",
		name:        "test-pod-001",
		expect:      "sum(container_memory_working_set_bytes{container!=\"POD\",namespace=\"default\",pod=\"test-pod-001\"})",
	}

	requests := GetPodMemUsageExpression(test.namespace, test.name)
	if requests != test.expect {
		t.Errorf("expect requests %s actual requests %s", test.expect, requests)
	}
}

func TestGetNodeCpuUsageExpression(t *testing.T) {
	test := struct {
		description string
		node        string
		expect      string
	}{
		description: "GetNodeCpuUsageExpression",
		node:        "test-node-001",
		expect:      "sum(count(node_cpu_seconds_total{mode=\"idle\",instance=~\"(test-node-001)(:\\\\d+)?\"}) by (mode, cpu)) - sum(irate(node_cpu_seconds_total{mode=\"idle\",instance=~\"(test-node-001)(:\\\\d+)?\"}[3m]))",
	}

	requests := GetNodeCpuUsageExpression(test.node)
	if requests != test.expect {
		t.Errorf("expect requests %s actual requests %s", test.expect, requests)
	}
}

func TestGetNodeMemUsageExpression(t *testing.T) {
	test := struct {
		description string
		node        string
		expect      string
	}{
		description: "GetNodeMemUsageExpression",
		node:        "test-node-001",
		expect:      "sum(node_memory_MemTotal_bytes{instance=~\"(test-node-001)(:\\\\d+)?\"} - node_memory_MemAvailable_bytes{instance=~\"(test-node-001)(:\\\\d+)?\"})",
	}

	requests := GetNodeMemUsageExpression(test.node)
	if requests != test.expect {
		t.Errorf("expect requests %s actual requests %s", test.expect, requests)
	}
}
