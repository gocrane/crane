package app

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/gocrane/crane/cmd/webhook/app/options"
	"github.com/gocrane/crane/pkg/version"
	"github.com/gocrane/crane/pkg/webhooks/podgroupprediction"
	podgrouppredictionv1alpha1 "github.com/gocrane/api/prediction/v1alpha1"
)

// aggregatedScheme aggregates Kubernetes and extended schemes.
var aggregatedScheme = runtime.NewScheme()

func init() {
	var _ = scheme.AddToScheme(aggregatedScheme)                     // add Kubernetes schemes
	var _ = podgrouppredictionv1alpha1.AddToScheme(aggregatedScheme) // add podgroupprediction schemes
}

// NewWebhookCommand creates a *cobra.Command object with default parameters
func NewWebhookCommand(ctx context.Context) *cobra.Command {
	opts := options.NewOptions()

	cmd := &cobra.Command{
		Use:  "crane-webhooks",
		Long: `Start a crane webhooks server`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := Run(ctx, opts); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
		},
	}

	cmd.Flags().AddGoFlagSet(flag.CommandLine)
	opts.AddFlags(cmd.Flags())

	return cmd
}

// Run runs the webhooks server with options. This should never exit.
func Run(ctx context.Context, opts *options.Options) error {
	klog.Infof("crane-webhooks version: %s", version.GetVersionInfo())
	config, err := controllerruntime.GetConfig()
	if err != nil {
		panic(err)
	}
	config.QPS, config.Burst = opts.WebhookClientQPS, opts.WebhookClientBurst

	hookManager, err := controllerruntime.NewManager(config, controllerruntime.Options{
		Scheme:         aggregatedScheme,
		Host:           opts.BindAddress,
		Port:           opts.SecurePort,
		CertDir:        opts.CertDir,
		LeaderElection: false,
	})
	if err != nil {
		klog.Errorf("failed to build webhooks server: %v", err)
		return err
	}

	klog.Info("registering webhooks to the webhooks server")
	hookServer := hookManager.GetWebhookServer()
	hookServer.Register("/validate-podgroupprediction", &webhook.Admission{Handler: &podgroupprediction.ValidatingAdmission{}})
	hookServer.Register("/mutate-podgroupprediction", &webhook.Admission{Handler: &podgroupprediction.MutatingAdmission{}})
	hookServer.WebhookMux.Handle("/readyz/", http.StripPrefix("/readyz/", &healthz.Handler{}))

	// blocks until the context is done.
	if err := hookManager.Start(ctx); err != nil {
		klog.Errorf("webhooks server exits unexpectedly: %v", err)
		return err
	}

	// never reach here
	return nil
}
