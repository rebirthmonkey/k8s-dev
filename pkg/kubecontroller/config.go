package kubecontroller

import (
	"sync"

	"github.com/rebirthmonkey/go/pkg/log"

	"github.com/rebirthmonkey/k8s-dev/pkg/apiextmgr"
	"github.com/rebirthmonkey/k8s-dev/pkg/reconcilermgr"
)

type Config struct {
	LogConfig           *log.Config
	ReconcilermgrConfig *reconcilermgr.Config
	APIExtConfig        *apiextmgr.Config
}

type CompletedConfig struct {
	CompletedLogConfig           *log.CompletedConfig
	CompletedReconcilermgrConfig *reconcilermgr.CompletedConfig
	CompletedAPIExtConfig        *apiextmgr.CompletedConfig
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
		APIExtConfig:        apiextmgr.NewConfig(),
	}
}

// Complete set default Configs.
func (c *Config) Complete() *CompletedConfig {

	onceConfig.Do(func() {
		config = CompletedConfig{
			CompletedLogConfig:           c.LogConfig.Complete(),
			CompletedReconcilermgrConfig: c.ReconcilermgrConfig.Complete(),
			CompletedAPIExtConfig:        c.APIExtConfig.Complete(),
		}
	})

	return &config
}

// New creates a new manager based on the configuration
func (c *CompletedConfig) New() (*KubeController, error) {
	err := c.CompletedLogConfig.New()
	if err != nil {
		log.Fatalf("Failed to launch Log: %s", err.Error())
		return nil, err
	}

	rmgr, err := c.CompletedReconcilermgrConfig.New()
	if err != nil {
		log.Fatalf("Failed to launch ReconcierManager: %s", err.Error())
		return nil, err
	}

	apiextmgr, err := c.CompletedAPIExtConfig.New()
	if err != nil {
		log.Fatalf("Failed to launch APIExtManager: %s", err.Error())
		return nil, err
	}

	mgr := &KubeController{
		ReconcilerManager: rmgr,
		APIExtManager:     apiextmgr,
	}

	return mgr, nil
}
