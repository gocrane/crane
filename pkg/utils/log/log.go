package log

import (
	"fmt"
	"sync"

	"github.com/go-logr/logr"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	once   sync.Once
	logger logr.Logger
)

func Logger() logr.Logger {
	if logger == nil {
		Init("default")
	}
	return logger
}

func Init(name string) {
	once.Do(func() {
		ctrl.SetLogger(klogr.New())
		logger = ctrl.Log.WithName(name)
	})
}

func NewLogger(name string) logr.Logger {
	return ctrl.Log.WithName(name)
}

func GenerateKey(name string, namespace string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}

func GenerateObj(obj klog.KMetadata) string {
	return klog.KObj(obj).String()
}
