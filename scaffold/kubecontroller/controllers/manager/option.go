package manager

import (
	"flag"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/rebirthmonkey/k8s-dev/pkg/conf"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	_ "github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/apis/app/v1"
)

type Options struct {
	ConfigPath     string
	APIServerURL   string
	APIExtsEnabled bool
	APIExtsURL     string
	APIExtsPort    int
	APIToken       string
	ZapOptions     zap.Options
	Logger         logr.Logger
	Portable       bool
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

	err := conf.Init(configPath)
	if err != nil {
		setupLog.Error(err, "Failed to load config file")
		os.Exit(1)
	}

	if apiextsUrl == "" {
		apiextsUrl = conf.Get(conf.APIExtsURL)
	} else {
		conf.Set(conf.APIExtsURL, apiextsUrl)
		apiextsUrl = conf.Get(conf.APIExtsURL)
	}

	return Options{
		ConfigPath:     configPath,
		APIServerURL:   apiServerURL,
		APIToken:       conf.Get(conf.BearerToken),
		APIExtsEnabled: apiextsEnabled,
		APIExtsPort:    apiextsPort,
		APIExtsURL:     apiextsUrl,
		Portable:       portable,
		Logger:         setupLog,
		ZapOptions:     zapOptions,
	}
}
