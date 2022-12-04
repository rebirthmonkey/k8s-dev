package reconcilermgr

type Config struct {
	MetricsBindAddress     string
	HealthProbeBindAddress string
	Concurrence            int
	Portable               bool
	APIServerURL           string
	APIExtsEnabled         bool
	APIExtsURL             string
	APIToken               string
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
		Portable:               c.Portable,
		APIServerURL:           c.APIServerURL,
		APIExtsEnabled:         c.APIExtsEnabled,
		APIExtsURL:             c.APIExtsURL,
		APIToken:               c.APIToken,
	}

	return rmgr, nil
}
