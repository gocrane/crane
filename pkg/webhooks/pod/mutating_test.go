package pod

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/yaml"

	"github.com/gocrane/api/ensurance/v1alpha1"

	"github.com/gocrane/crane/pkg/ensurance/config"
)

func TestDefaultingPodQOSInitializer(t *testing.T) {
	configYaml := "apiVersion: ensurance.crane.io/v1alpha1\nkind: QOSConfig\nqosInitializer:\n  enable: true\n  selector: \n    matchLabels:\n      app: nginx\n  initContainerTemplate:\n    name: crane-qos-initializer\n    image: docker.io/gocrane/qos-init:v0.1.1\n    imagePullPolicy: IfNotPresent\n    command:\n      - sh\n      - -x\n      - /qos-checking.sh\n    volumeMounts:\n      - name: podinfo\n        mountPath: /etc/podinfo\n  volumeTemplate:\n    name: podinfo\n    downwardAPI:\n      items:\n      - path: \"annotations\"\n        fieldRef:\n          fieldPath: metadata.annotations"

	config := &config.QOSConfig{}
	err := yaml.Unmarshal([]byte(configYaml), config)
	if err != nil {
		t.Errorf("unmarshal config failed:%v", err)
	}
	m := MutatingAdmission{
		Config:     config,
		listPodQOS: MockListPodQOSFunc,
	}

	type Case struct {
		Pod    *v1.Pod
		Inject bool
	}

	for _, tc := range []Case{
		{Pod: MockPod("offline", "offline", "enable", "app", "nginx"), Inject: true},
		{Pod: MockPod("offline-not-interested", "offline", "enable"), Inject: false},
		{Pod: MockPod("online", "offline", "disable", "app", "nginx"), Inject: false},
		{Pod: MockPod("online-not-interested", "offline", "disable"), Inject: false},
		{Pod: MockPod("default"), Inject: false},
	} {
		assert.NoError(t, m.Default(context.Background(), tc.Pod))
		t.Log(tc.Pod.Name)
		assert.Equal(t, len(tc.Pod.Spec.InitContainers) == 1, tc.Inject)
		assert.Equal(t, len(tc.Pod.Spec.Volumes) == 1, tc.Inject)
	}
}

func TestPrecheck(t *testing.T) {
	configYaml := "apiVersion: ensurance.crane.io/v1alpha1\nkind: QOSConfig\nqosInitializer:\n  enable: true\n  selector: \n    matchLabels:\n      app: nginx\n"

	config := &config.QOSConfig{}
	err := yaml.Unmarshal([]byte(configYaml), config)
	if err != nil {
		t.Errorf("unmarshal config failed:%v", err)
	}
	m := MutatingAdmission{
		Config:     config,
		listPodQOS: MockListPodQOSFunc,
	}
	assert.False(t, m.available())
}

func MockListPodQOSFunc() ([]*v1alpha1.PodQOS, error) {
	return []*v1alpha1.PodQOS{
		{
			TypeMeta:   metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{},
			Spec: v1alpha1.PodQOSSpec{
				LabelSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{"offline": "enable"},
				},
				ResourceQOS: v1alpha1.ResourceQOS{
					CPUQOS: &v1alpha1.CPUQOS{
						CPUPriority: pointer.Int32(7),
					},
				},
			},
		}, {
			TypeMeta:   metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{},
			Spec: v1alpha1.PodQOSSpec{
				LabelSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{"offline": "disable"},
				},
				ResourceQOS: v1alpha1.ResourceQOS{
					CPUQOS: &v1alpha1.CPUQOS{
						CPUPriority: pointer.Int32(0),
					},
				},
			},
		}, {
			TypeMeta:   metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{},
			Spec: v1alpha1.PodQOSSpec{
				ResourceQOS: v1alpha1.ResourceQOS{
					CPUQOS: &v1alpha1.CPUQOS{
						CPUPriority: pointer.Int32(7),
					},
				},
			},
		},
	}, nil
}

func MockPod(name string, labels ...string) *v1.Pod {
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: nil,
		},
	}

	if len(labels) < 2 {
		return pod
	}

	labelmap := map[string]string{}
	for i := 0; i < len(labels)-1; i += 2 {
		labelmap[labels[i]] = labels[i+1]
	}
	pod.Labels = labelmap
	return pod
}
