package reconcilermgr

type Config struct {
	MetricsBindAddress     string
	HealthProbeBindAddress string
	Concurrence            int
	APIServerURL           string
}

type CompletedConfig struct {
	*Config
}

func NewConfig() *Config {
	return &Config{}
}

func (c *Config) Complete() *CompletedConfig {
	return &CompletedConfig{c}
}

// New returns a new instance of ReconcilerManager from the given config.
func (c CompletedConfig) New() (*ReconcilerManager, error) {

	rmgr := &ReconcilerManager{
		MetricsBindAddress:     c.MetricsBindAddress,
		HealthProbeBindAddress: c.HealthProbeBindAddress,
		Concurrence:            c.Concurrence,
		APIServerURL:           c.APIServerURL,
	}

	return rmgr, nil
}
