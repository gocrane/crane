package main

import (
	"fmt"
	"os"

	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/component-base/logs"

	"github.com/gocrane/crane/cmd/craned/app"
	"github.com/gocrane/crane/pkg/utils/clogs"
)

// craned main.
func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	clogs.InitLogs("craned")

	ctx := genericapiserver.SetupSignalContext()

	if err := app.NewManagerCommand(ctx).Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
