package main

import (
	"fmt"
	"k8s.io/component-base/logs"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/gocrane/crane/cmd/craned/app"
)

// craned main.
func main() {
	logs.InitLogs()
	defer logs.FlushLogs()

	ctx := signals.SetupSignalHandler()

	root := app.NewManagerCommand(ctx)
	root.AddCommand(app.NewCmdVersion())

	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
