package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/spf13/pflag"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/gocrane/crane/cmd/crane-agent/app"
)

// crane-agent main.
func main() {
	klog.InitFlags(nil)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	rand.Seed(time.Now().UnixNano())

	ctx := signals.SetupSignalHandler()

	root := app.NewAgentCommand(ctx)
	root.AddCommand(app.NewCmdVersion())

	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
