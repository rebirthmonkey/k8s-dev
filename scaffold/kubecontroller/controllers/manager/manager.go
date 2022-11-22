package manager

import (
	"context"
	"fmt"
	"os"

	"github.com/rebirthmonkey/k8s-dev/pkg/conf"
	"github.com/rebirthmonkey/k8s-dev/pkg/controller/registry"
	"github.com/rebirthmonkey/k8s-dev/pkg/manager"
	"github.com/rebirthmonkey/k8s-dev/pkg/version"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/apis"
	_ "github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/apis/app/v1"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(apis.AddToScheme(scheme))
}

func Main(opts Options, preStartHook func(context.Context, *manager.ReconcilerManager)) {
	apiServerURL := opts.APIServerURL
	apiToken := opts.APIToken
	apiextsURL := opts.APIExtsURL
	portable := opts.Portable
	zapOptions := opts.ZapOptions
	setupLog := opts.Logger

	ctx := ctrl.SetupSignalHandler()

	setupLog.Info(fmt.Sprintf("Kubecontroller version: %s", version.Info()))
	var err error

	var restConfig *rest.Config
	if apiServerURL == "" {
		restConfig, err = config.GetConfig()
		if err != nil {
			setupLog.Error(err, "Failed to load config, if you are debugging locally, provide --kubeconfig or --apiserver-url flags")
			os.Exit(1)
		}
	} else {
		restConfig, err = clientcmd.BuildConfigFromFlags(apiServerURL, "")
		restConfig.BearerToken = apiToken
	}
	if err != nil {
		setupLog.Error(err, "Failed to build rest.Config")
		os.Exit(1)
	}
	concurrence := conf.GetInt(conf.ReconcileConcurrence)
	reconcilerMgr, err := manager.NewReconcilerManager(&manager.Config{
		RestConfig:   restConfig,
		Scheme:       scheme,
		Concurrence:  concurrence,
		Context:      ctx,
		Portable:     portable,
		ZapOptions:   zapOptions,
		APIServerUrl: apiServerURL,
		APIExtsURL:   apiextsURL,
		APIToken:     apiToken,
	})
	if err != nil {
		os.Exit(1)
	}

	registry.AddToManager(reconcilerMgr)

	if err := reconcilerMgr.Setup(); err != nil {
		setupLog.Error(err, "Failed to setup reconcilers")
		os.Exit(1)
	}

	if preStartHook != nil {
		preStartHook(ctx, reconcilerMgr)
	}
	
	if err := reconcilerMgr.Start(); err != nil {
		setupLog.Error(err, "Error occurred while controller manager is running")
	}
}
