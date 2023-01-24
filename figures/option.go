package figures

import (
	"flag"
	"fmt"
	"github.com/rebirthmonkey/k8s-dev/pkg/reconcilermgr"
	"github.com/spf13/pflag"
	"os"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type Options struct {
	ConfigPath     string `json:"config-path"mapstructure:"config-path"`
	APIServerURL   string `json:"apiserver-url"       mapstructure:"apiserver-url"`
	APIExtsEnabled bool   `json:"apiexts-enabled"       mapstructure:"apiexts-enabled"`
	APIExtsURL     string `json:"apiexts-url"       mapstructure:"apiexts-url"`
	APIExtsPort    int    `json:"apiexts-port"       mapstructure:"apiexts-port"`
	APIToken       string `json:"api-token"       mapstructure:"api-token"`
	Portable       bool   `json:"portable"       mapstructure:"portable"`
}

func NewOptions() *Options {
	return &Options{
		ConfigPath:     "",
		APIServerURL:   "",
		APIToken:       "",
		APIExtsEnabled: false,
		APIExtsPort:    6084,
		APIExtsURL:     "",
		Portable:       false,
	}
}

func (o *Options) Validate() []error {
	var errors []error

	if o.APIServerURL == "" {
		errors = append(errors, fmt.Errorf("invalide config"))
	}

	return errors
}

func (o *Options) ApplyTo(c *reconcilermgr.Config) error {
	c.ConfigPath = o.ConfigPath
	c.APIServerEnabled = o.APIServerURL
	c.APIToken = o.APIToken
	c.APIExtsEnabled = o.APIExtsEnabled
	c.APIExtsPort = o.APIExtsPort
	c.APIExtsURL = o.APIExtsURL
	c.Portable = o.Portable

	return nil
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	flag.StringVar(&o.ConfigPath, "config", "configs/kubecontroller.yaml", "Teleport config file location.")
	flag.StringVar(&o.APIServerURL, "apiserver-url", "", "Teleport api server url, assumes running in kubernetes cluster if empty")
	flag.BoolVar(&o.Portable, "portable", false, "Whether to run the controller manager in portable mode")
	flag.BoolVar(&o.APIExtsEnabled, "apiexts-enabled", true, "Whether to enable embedded APIExts server")
	flag.IntVar(&o.APIExtsPort, "apiexts-port", 6084, "Port which APIExts server listens on")
	flag.StringVar(&o.APIExtsURL, "apiexts-url", "", "URL of external APIExts server, this flag is ignored if embedded APIExts server is enabled")

}

func ParseOptionsFromFlags(allowEmbedAPIExtsServer bool) Options {
	var configPath string
	var apiServerURL string
	var apiextsUrl string
	var apiextsEnabled bool
	var apiextsPort int
	var portable bool
	flag.StringVar(&configPath, "config", "configs/kubecontroller.yaml", "Teleport config file location.")
	flag.StringVar(&apiServerURL, "apiserver-url", "", "Teleport api server url, assumes running in kubernetes cluster if empty")
	flag.BoolVar(&portable, "portable", false, "Whether to run the controller manager in portable mode")
	if allowEmbedAPIExtsServer {
		flag.BoolVar(&apiextsEnabled, "apiexts-enabled", true, "Whether to enable embedded APIExts server")
		flag.IntVar(&apiextsPort, "apiexts-port", 6084, "Port which APIExts server listens on")
	}
	flag.StringVar(&apiextsUrl, "apiexts-url", "", "URL of external APIExts server, this flag is ignored if embedded APIExts server is enabled")

	zapOptions := zap.Options{
		Development: true,
	}
	zapOptions.BindFlags(flag.CommandLine)
	flag.Parse()

	if apiextsEnabled {
		apiextsUrl = fmt.Sprintf("http://127.0.0.1:%d", apiextsPort)
	}

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zapOptions)))
	setupLog := ctrl.Log.WithName("setup")

	err := Init(configPath)
	if err != nil {
		setupLog.Error(err, "Failed to load config file")
		os.Exit(1)
	}

	if apiextsUrl == "" {
		apiextsUrl = Get(APIExtsURL)
	} else {
		Set(APIExtsURL, apiextsUrl)
		apiextsUrl = Get(APIExtsURL)
	}

	return Options{
		ConfigPath:     configPath,
		APIServerURL:   apiServerURL,
		APIToken:       Get(BearerToken),
		APIExtsEnabled: apiextsEnabled,
		APIExtsPort:    apiextsPort,
		APIExtsURL:     apiextsUrl,
		Portable:       portable,
	}
}
