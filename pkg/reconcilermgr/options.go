package reconcilermgr

import (
	"github.com/spf13/pflag"
)

type Options struct {
	MetricsBindAddress     string `json:"metrics-bind-address"       mapstructure:"metrics-bind-address"`
	HealthProbeBindAddress string `json:"health-probe-bind-address"       mapstructure:"health-probe-bind-address"`
	Concurrence            int    `json:"concurrence"       mapstructure:"concurrence"`
	APIServerURL           string `json:"apiserver-url"       mapstructure:"apiserver-url"`
	APIExtsEnabled         bool   `json:"apiexts-enabled"       mapstructure:"apiexts-enabled"`
	APIExtsURL             string `json:"apiexts-url"       mapstructure:"apiexts-url"`
	APIExtsPort            int    `json:"apiexts-port"       mapstructure:"apiexts-port"`
	APIToken               string `json:"api-token"       mapstructure:"api-token"`
	Portable               bool   `json:"portable"       mapstructure:"portable"`
}

// NewOptions creates an Options object with default parameters.
func NewOptions() *Options {
	return &Options{
		MetricsBindAddress:     "",
		HealthProbeBindAddress: "",
		Concurrence:            0,
		APIServerURL:           "",
		APIExtsEnabled:         false,
		APIExtsURL:             "",
		APIExtsPort:            0,
		APIToken:               "",
		Portable:               false,
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
	c.APIToken = o.APIToken
	c.APIExtsEnabled = o.APIExtsEnabled
	c.APIExtsURL = o.APIExtsURL
	c.Portable = o.Portable

	return nil
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.APIServerURL, "apiserver-url", "", "Teleport api server url, assumes running in kubernetes cluster if empty")
	fs.BoolVar(&o.Portable, "portable", false, "Whether to run the controller manager in portable mode")
	fs.BoolVar(&o.APIExtsEnabled, "apiexts-enabled", true, "Whether to enable embedded APIExts server")
	fs.StringVar(&o.APIExtsURL, "apiexts-url", "", "URL of external APIExts server, this flag is ignored if embedded APIExts server is enabled")
	fs.IntVar(&o.APIExtsPort, "apiexts-port", 6084, "Port which APIExts server listens on")
}
