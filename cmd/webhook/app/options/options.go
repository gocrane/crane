package options

import (
	"github.com/spf13/pflag"
)

const (
	defaultBindAddress = "0.0.0.0"
	defaultPort        = 8443
	defaultCertDir     = "/etc/craned/tls-certs"
)

// Options contains everything necessary to create and run webhooks server.
type Options struct {
	// BindAddress is the IP address on which to listen for the --secure-port port.
	// Default is "0.0.0.0".
	BindAddress string
	// SecurePort is the port that the webhooks server serves at.
	// Default is 8443.
	SecurePort int
	// CertDir is the directory that contains the server key and certificate.
	// if not set, webhooks server would look up the server key and certificate in {TempDir}/k8s-webhooks-server/serving-certs.
	// The server key and certificate must be named `tls.key` and `tls.crt`, respectively.
	CertDir string
	// WebhookClientQPS is the QPS that webhook server talks to api server.
	WebhookClientQPS float32
	// WebhookClientBurst is the Burst that webhook server talks to api server.
	WebhookClientBurst int
}

// NewOptions builds an empty options.
func NewOptions() *Options {
	return &Options{}
}

// AddFlags adds flags to the specified FlagSet.
func (o *Options) AddFlags(flags *pflag.FlagSet) {
	flags.StringVar(&o.BindAddress, "bind-address", defaultBindAddress,
		"The IP address on which to listen for the --secure-port port.")
	flags.IntVar(&o.SecurePort, "secure-port", defaultPort,
		"The secure port on which to serve HTTPS.")
	flags.StringVar(&o.CertDir, "cert-dir", defaultCertDir,
		"The directory that contains the server key(named tls.key) and certificate(named tls.crt).")
	flags.Float32Var(&o.WebhookClientQPS, "webhook-client-qps", 40.0, "QPS to use while talking with kube-apiserver.")
	flags.IntVar(&o.WebhookClientBurst, "webhook-client-burst", 60, "Burst to use while talking with kube-apiserver.")
}
