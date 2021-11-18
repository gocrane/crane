package main

import (
	"fmt"
	"os"

	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/component-base/logs"
	"k8s.io/klog/v2/klogr"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/gocrane-io/crane/cmd/crane-manager/app"
)

// crane-Manager main.
func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	ctrl.SetLogger(klogr.New())

	ctx := genericapiserver.SetupSignalContext()

	if err := app.NewManagerCommand(ctx).Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
