package manager

import (
	cliflag "github.com/marmotedu/component-base/pkg/cli/flag"
	"github.com/rebirthmonkey/go/pkg/log"
	"github.com/rebirthmonkey/k8s-dev/pkg/reconcilermgr"

	"sync"
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

func ParseOptionsFromFlags(allowEmbedAPIExtsServer bool) Options {
	//var configPath string
	//var apiServerURL string
	//var apiextsUrl string
	//var apiextsEnabled bool
	//var apiextsPort int
	//var portable bool
	////flag.StringVar(&configPath, "config", "configs/kubecontroller.yaml", "Teleport config file location.")
	//flag.StringVar(&configPath, "config", "/Users/ruan/workspace/k8s-dev/scaffold/kubecontroller/configs/kubecontroller.yaml", "config file location.")
	//flag.StringVar(&apiServerURL, "apiserver-url", "", "Teleport api server url, assumes running in kubernetes cluster if empty")
	//flag.BoolVar(&portable, "portable", false, "Whether to run the controller manager in portable mode")
	//if allowEmbedAPIExtsServer {
	//	flag.BoolVar(&apiextsEnabled, "apiexts-enabled", true, "Whether to enable embedded APIExts server")
	//	flag.IntVar(&apiextsPort, "apiexts-port", 6084, "Port which APIExts server listens on")
	//}
	//flag.StringVar(&apiextsUrl, "apiexts-url", "", "URL of external APIExts server, this flag is ignored if embedded APIExts server is enabled")
	//
	//zapOptions := zap.Options{
	//	Development: true,
	//}
	//zapOptions.BindFlags(flag.CommandLine)
	//flag.Parse()
	//
	//if apiextsEnabled {
	//	apiextsUrl = fmt.Sprintf("http://127.0.0.1:%d", apiextsPort)
	//}

	//ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zapOptions)))
	//setupLog := ctrl.Log.WithName("setup")

	//err := conf.Init(configPath)
	//if err != nil {
	//	setupLog.Error(err, "Failed to load config file")
	//	os.Exit(1)
	//}

	//if apiextsUrl == "" {
	//	apiextsUrl = conf.Get(conf.APIExtsURL)
	//} else {
	//	conf.Set(conf.APIExtsURL, apiextsUrl)
	//	apiextsUrl = conf.Get(conf.APIExtsURL)
	//}

	return Options{
		//ConfigPath: "/Users/ruan/workspace/k8s-dev/scaffold/kubecontroller/configs/kubecontroller.yaml",
		//APIServerURL:   apiServerURL,
		//APIToken:       conf.Get(conf.BearerToken),
		//APIExtsEnabled: apiextsEnabled,
		//APIExtsPort:    apiextsPort,
		//APIExtsURL:     apiextsUrl,
		//Portable:       portable,
		//Logger:         setupLog,
		//ZapOptions:     zapOptions,
	}
}
