package apiextmgr

import (
	"sync"

	cliflag "github.com/marmotedu/component-base/pkg/cli/flag"
	"github.com/rebirthmonkey/go/pkg/gin"
	"github.com/rebirthmonkey/go/pkg/log"
)

type Options struct {
	*gin.Options `json:"gin"   mapstructure:"gin"`
}

var (
	opt  Options
	once sync.Once
)

// NewOptions creates a new Options object with default parameters.
func NewOptions() *Options {
	once.Do(func() {
		opt = Options{
			Options: gin.NewOptions(),
		}
	})

	return &opt
}

// Validate checks Options and return a slice of found errs.
func (o *Options) Validate() []error {
	var errs []error

	errs = append(errs, o.Options.Validate()...)

	return errs
}

// ApplyTo applies the run options to the method receiver and returns self.
func (o *Options) ApplyTo(c *Config) error {
	if err := o.Options.ApplyTo(c.Config); err != nil {
		log.Panic(err.Error())
	}

	return nil
}

// Flags returns flags for a specific APIServer by section name.
func (o *Options) Flags() (fss cliflag.NamedFlagSets) {
	o.Options.AddFlags()

	return fss
}
