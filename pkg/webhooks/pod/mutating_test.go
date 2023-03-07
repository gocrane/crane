package pod

import (
	"context"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

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
		Config: config,
	}

	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pod1",
			Labels: map[string]string{
				"app": "nginx",
			},
		},
	}
	err = m.Default(context.TODO(), pod)
	if err != nil {
		t.Fatalf("inject pod failed: %v", err)
	}
	if len(pod.Spec.InitContainers) == 0 {
		t.Fatalf("should inject containers")
	}
}
