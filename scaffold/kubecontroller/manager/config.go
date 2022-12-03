package manager

import (
	"github.com/rebirthmonkey/go/pkg/log"
	"github.com/rebirthmonkey/k8s-dev/pkg/reconcilermgr"
	"sync"
)

type Config struct {
	LogConfig           *log.Config
	ReconcilermgrConfig *reconcilermgr.Config
}

type CompletedConfig struct {
	CompletedLogConfig           *log.CompletedConfig
	CompletedReconcilermgrConfig *reconcilermgr.CompletedConfig
}

var (
	config     CompletedConfig
	onceConfig sync.Once
)

// NewConfig creates a running configuration instance based
// on a given command line or configuration file option.
func NewConfig() *Config {
	return &Config{
		LogConfig:           log.NewConfig(),
		ReconcilermgrConfig: reconcilermgr.NewConfig(),
	}
}

// Complete set default Configs.
func (c *Config) Complete() *CompletedConfig {

	onceConfig.Do(func() {
		config = CompletedConfig{
			CompletedLogConfig:           c.LogConfig.Complete(),
			CompletedReconcilermgrConfig: c.ReconcilermgrConfig.Complete(),
		}
	})

	return &config
}

// New creates a new manager based on the configuration
func (c *CompletedConfig) New() (*Manager, error) {
	err := c.CompletedLogConfig.New()
	if err != nil {
		log.Fatalf("Failed to launch Log: %s", err.Error())
		return nil, err
	}

	rmgr, err := c.CompletedReconcilermgrConfig.New()
	if err != nil {
		log.Fatalf("Failed to launch Log: %s", err.Error())
		return nil, err
	}

	mgr := &Manager{
		ReconcilerManager: rmgr,
	}

	return mgr, nil
}
