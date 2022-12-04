package manager

import (
	"sync"

	cliflag "github.com/marmotedu/component-base/pkg/cli/flag"
	"github.com/rebirthmonkey/go/pkg/log"
	"github.com/rebirthmonkey/k8s-dev/pkg/reconcilermgr"
)

type Options struct {
	LogOptions           *log.Options           `json:"log"   mapstructure:"log"`
	ReconcilermgrOptions *reconcilermgr.Options `json:"reconcilermgr"   mapstructure:"reconcilermgr"`
}

var (
	opt  Options
	once sync.Once
)

// NewOptions creates a new Options object with default parameters.
func NewOptions() *Options {
	once.Do(func() {
		opt = Options{
			LogOptions:           log.NewOptions(),
			ReconcilermgrOptions: reconcilermgr.NewOptions(),
		}
	})

	return &opt
}

// Validate checks Options and return a slice of found errs.
func (o *Options) Validate() []error {
	var errs []error

	errs = append(errs, o.LogOptions.Validate()...)
	errs = append(errs, o.ReconcilermgrOptions.Validate()...)

	return errs
}

// ApplyTo applies the run options to the method receiver and returns self.
func (o *Options) ApplyTo(c *Config) error {
	if err := o.LogOptions.ApplyTo(c.LogConfig); err != nil {
		log.Panic(err.Error())
	}

	if err := o.ReconcilermgrOptions.ApplyTo(c.ReconcilermgrConfig); err != nil {
		log.Panic(err.Error())
	}

	return nil
}

// Flags returns flags for a specific APIServer by section name.
func (o *Options) Flags() (fss cliflag.NamedFlagSets) {
	o.ReconcilermgrOptions.AddFlags(fss.FlagSet("reconcilermgr"))
	o.LogOptions.AddFlags(fss.FlagSet("log"))

	return fss
}
