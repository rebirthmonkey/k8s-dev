package reconcilermgr

import (
	"github.com/spf13/pflag"
)

type Options struct {
	MetricsBindAddress     string `json:"metrics-bind-address"       mapstructure:"metrics-bind-address"`
	HealthProbeBindAddress string `json:"health-probe-bind-address"       mapstructure:"health-probe-bind-address"`
	Concurrence            int    `json:"concurrence"       mapstructure:"concurrence"`
	APIServerURL           string `json:"apiserver-url"       mapstructure:"apiserver-url"`
	Kubeconfig             string `json:"kubeconfig"       mapstructure:"kubeconfig"`
}

// NewOptions creates an Options object with default parameters.
func NewOptions() *Options {
	return &Options{
		MetricsBindAddress:     "",
		HealthProbeBindAddress: "",
		Concurrence:            0,
		APIServerURL:           "",
		Kubeconfig:             "",
	}
}

// Validate checks Options and return a slice of found errs.
func (o *Options) Validate() []error {
	var errs []error

	return errs
}

func (o *Options) ApplyTo(c *Config) error {
	c.MetricsBindAddress = o.MetricsBindAddress
	c.HealthProbeBindAddress = o.HealthProbeBindAddress
	c.Concurrence = o.Concurrence
	c.APIServerURL = o.APIServerURL
	c.Kubeconfig = o.Kubeconfig

	return nil
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.APIServerURL, "apiserver-url", "", "APIServer url, assumes running in kubernetes cluster if empty")
	fs.StringVar(&o.Kubeconfig, "kubeconfig", "", "Kubeconfig, assumes running in kubernetes cluster if empty")
}
