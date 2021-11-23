package clogs

import (
	"fmt"
	"sync"

	"k8s.io/klog/v2"

	"github.com/go-logr/logr"
	"k8s.io/klog/v2/klogr"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	craneLogger CLogs
)

type CLogs struct {
	once   sync.Once
	logger logr.Logger
}

func Log() logr.Logger {
	return craneLogger.logger
}

func InitLogs(name string) {
	craneLogger.once.Do(func() {
		ctrl.SetLogger(klogr.New())
		craneLogger.logger = ctrl.Log.WithName(name)
	})
}

func GenerateKey(name string, namespace string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}

func GenerateObj(obj klog.KMetadata) string {
	return klog.KObj(obj).String()
}
