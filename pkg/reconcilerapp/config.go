package reconcilerapp

import (
	"sync"

	"github.com/rebirthmonkey/go/pkg/gin"
	"github.com/rebirthmonkey/go/pkg/log"

	"github.com/rebirthmonkey/k8s-dev/pkg/reconcilermgr"
)

type Config struct {
	LogConfig           *log.Config
	ReconcilermgrConfig *reconcilermgr.Config
	GinConfig           *gin.Config
}

type CompletedConfig struct {
	CompletedLogConfig           *log.CompletedConfig
	CompletedReconcilermgrConfig *reconcilermgr.CompletedConfig
	CompletedGinConfig           *gin.CompletedConfig
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
		GinConfig:           gin.NewConfig(),
	}
}

// Complete set default Configs.
func (c *Config) Complete() *CompletedConfig {

	onceConfig.Do(func() {
		config = CompletedConfig{
			CompletedLogConfig:           c.LogConfig.Complete(),
			CompletedReconcilermgrConfig: c.ReconcilermgrConfig.Complete(),
			CompletedGinConfig:           c.GinConfig.Complete(),
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
		log.Fatalf("Failed to launch ReconcierManager: %s", err.Error())
		return nil, err
	}

	ginServer, err := c.CompletedGinConfig.New()
	if err != nil {
		log.Fatalf("Failed to launch Gin server: %s", err.Error())
		return nil, err
	}

	mgr := &Manager{
		ReconcilerManager: rmgr,
		ginServer:         ginServer,
	}

	return mgr, nil
}
