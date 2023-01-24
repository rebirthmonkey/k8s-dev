package apiextmgr

import (
	"sync"

	"github.com/rebirthmonkey/go/pkg/gin"
	"github.com/rebirthmonkey/go/pkg/log"
)

type Config struct {
	*gin.Config
}

type CompletedConfig struct {
	*gin.CompletedConfig
}

var (
	config     CompletedConfig
	onceConfig sync.Once
)

// NewConfig creates a running configuration instance based
// on a given command line or configuration file option.
func NewConfig() *Config {
	return &Config{
		Config: gin.NewConfig(),
	}
}

// Complete set default Configs.
func (c *Config) Complete() *CompletedConfig {

	onceConfig.Do(func() {
		config = CompletedConfig{
			CompletedConfig: c.Config.Complete(),
		}
	})

	return &config
}

// New creates a new manager based on the configuration
func (c *CompletedConfig) New() (*APIExtManager, error) {

	ginServer, err := c.CompletedConfig.New()
	if err != nil {
		log.Fatalf("Failed to launch APIExtManager: %s", err.Error())
		return nil, err
	}

	mgr := &APIExtManager{
		Server: ginServer,
	}

	return mgr, nil
}
