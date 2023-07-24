package runtime

import (
	"context"
	"fmt"
	"sort"
	"strings"

	pb "k8s.io/cri-api/pkg/apis/runtime/v1"
	"k8s.io/klog/v2"
)

type sandboxByCreated []*pb.PodSandbox

func (a sandboxByCreated) Len() int      { return len(a) }
func (a sandboxByCreated) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a sandboxByCreated) Less(i, j int) bool {
	return a[i].CreatedAt > a[j].CreatedAt
}

// ListPodSandboxes sends a ListPodSandboxRequest to the server, and parses
// the returned ListPodSandboxResponse.
// copied from kubernetes-sigs/cri-tools/cmd/crictl/sandbox.go
func ListPodSandboxes(client pb.RuntimeServiceClient, opts ListOptions) ([]*pb.PodSandbox, error) {
	filter := &pb.PodSandboxFilter{}
	if opts.id != "" {
		filter.Id = opts.id
	}
	if opts.state != "" {
		st := &pb.PodSandboxStateValue{}
		st.State = pb.PodSandboxState_SANDBOX_NOTREADY
		switch strings.ToLower(opts.state) {
		case "ready":
			st.State = pb.PodSandboxState_SANDBOX_READY
			filter.State = st
		case "notready":
			st.State = pb.PodSandboxState_SANDBOX_NOTREADY
			filter.State = st
		default:
			klog.Errorf("state should be ready or notready")
			return []*pb.PodSandbox{}, fmt.Errorf("state should be ready or notready")
		}
	}
	if opts.labels != nil {
		filter.LabelSelector = opts.labels
	}
	request := &pb.ListPodSandboxRequest{
		Filter: filter,
	}
	klog.V(6).Info("ListPodSandboxRequest: %v", request)
	r, err := client.ListPodSandbox(context.Background(), request)
	if err != nil {
		return []*pb.PodSandbox{}, err
	}
	klog.V(6).Info("ListPodSandboxResponse: %v", r)

	r.Items = filterSandboxesList(r.GetItems(), opts)
	return r.Items, nil
}

func filterSandboxesList(sandboxesList []*pb.PodSandbox, opts ListOptions) []*pb.PodSandbox {
	filtered := []*pb.PodSandbox{}
	for _, p := range sandboxesList {
		// Filter by pod name/namespace regular expressions.
		if matchRegex(opts.nameRegexp, p.Metadata.Name) &&
			matchRegex(opts.podNamespaceRegexp, p.Metadata.Namespace) {
			filtered = append(filtered, p)
		}
	}

	sort.Sort(sandboxByCreated(filtered))
	n := len(filtered)
	if opts.latest {
		n = 1
	}
	if opts.last > 0 {
		n = opts.last
	}
	n = func(a, b int) int {
		if a < b {
			return a
		}
		return b
	}(n, len(filtered))

	return filtered[:n]
}
