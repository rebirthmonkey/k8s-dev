package manager

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-logr/logr"
	"github.com/rebirthmonkey/k8s-dev/pkg/conf"
	fns "github.com/rebirthmonkey/k8s-dev/pkg/func"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	crmgr "sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/apis"
)

type ReconcilerSetuper interface {
	// SetupWithManager register the controller in manager
	SetupWithManager(mgr ctrl.Manager) error
	// For what resource the controller interests in
	For() string
	// KeyFilter if set, only matched (by key, aka namespace/name ) resource instance will be processed by the controller
	KeyFilter(filter string)
	// AfterCacheSync hook that will be executed after cache sync
	AfterCacheSync(mgr ctrl.Manager) error
}

type Config struct {
	Scheme     *runtime.Scheme
	RestConfig *rest.Config

	Concurrence   int
	NoCacheClient client.Client
	Client        client.Client
	Context       context.Context
	Portable      bool
	ZapOptions    zap.Options
	APIServerUrl  string
	APIExtsURL    string
	APIToken      string
}

type ReconcilerManager struct {
	config             *Config
	setupers           []ReconcilerSetuper
	mgr                crmgr.Manager
	logger             logr.Logger
	enabledControllers []string
}

func (rmgr *ReconcilerManager) With(setupers ...ReconcilerSetuper) {
	for _, setuper := range setupers {
		res := setuper.For()
		cfg, ok := apis.ResourceMetadatas[res]
		if ok && cfg.NoPortable && rmgr.config.Portable {
			rmgr.logger.Info(fmt.Sprintf("Ignoring controller for non-portable resource %s", res))
			continue
		}
		if fns.IsEmpty(rmgr.enabledControllers) || fns.Contains(rmgr.enabledControllers, res) {
			rmgr.setupers = append(rmgr.setupers, setuper)
			rmgr.logger.Info(fmt.Sprintf("Add controller for %s to manager", res))
		} else {
			rmgr.logger.Info(fmt.Sprintf("Controller for %s disabled", res))
		}
	}
}

func (rmgr *ReconcilerManager) Setup() error {
	mgr := rmgr.mgr
	for _, setuper := range rmgr.setupers {
		resourceFilterConfKey := fmt.Sprintf("resource-filter.%s", setuper.For())
		filter := conf.Get(resourceFilterConfKey)
		if filter != "" {
			setuper.KeyFilter(filter)
		}
		if err := setuper.SetupWithManager(mgr); err != nil {
			return err
		}
	}
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		rmgr.logger.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		rmgr.logger.Error(err, "unable to set up ready check")
		os.Exit(1)
	}
	return nil
}

func (rmgr *ReconcilerManager) Start() error {
	rmgr.logger.Info("Starting controller manager")
	mgr := rmgr.mgr
	go func() {
		rmgr.logger.Info("Wait until cache synchronized")
		mgr.GetCache().WaitForCacheSync(rmgr.config.Context)
		rmgr.logger.Info("Cache synchronized, execute AfterCacheSync hook for all controllers")
		for _, setuper := range rmgr.setupers {
			setuper.AfterCacheSync(mgr)
		}
	}()
	return mgr.Start(rmgr.config.Context)
}

func (rmgr *ReconcilerManager) GetNoCacheClient() client.Client {
	return rmgr.config.NoCacheClient
}

func (rmgr *ReconcilerManager) GetScheme() *runtime.Scheme {
	return rmgr.config.Scheme
}

func (rmgr *ReconcilerManager) GetClient() client.Client {
	return rmgr.config.Client
}

func (rmgr *ReconcilerManager) GetDefaultConcurrence() int {
	return rmgr.config.Concurrence
}

func (rmgr *ReconcilerManager) GetLogOptions() zap.Options {
	return rmgr.config.ZapOptions
}

func (rmgr *ReconcilerManager) GetContext() context.Context {
	return rmgr.config.Context
}

func (rmgr *ReconcilerManager) GetAPIExtsURL() string {
	return rmgr.config.APIExtsURL
}

func (rmgr *ReconcilerManager) GetAPIToken() string {
	return rmgr.config.APIToken
}

func (rmgr *ReconcilerManager) GetAPIServerURL() string {
	return rmgr.config.APIServerUrl
}

func NewReconcilerManager(cfg *Config) (*ReconcilerManager, error) {
	mgr, err := ctrl.NewManager(cfg.RestConfig, ctrl.Options{
		Scheme:                 cfg.Scheme,
		MetricsBindAddress:     conf.Get(conf.MetricsBindAddress),
		Port:                   conf.GetInt(conf.Port),
		HealthProbeBindAddress: conf.Get(conf.HealthProbeBindAddress),
		LeaderElection:         conf.GetBool(conf.EnableLeaderElection),
		LeaderElectionID:       conf.Get(conf.LeaderElectionID),
		Namespace:              conf.Get(conf.Namespace),
		SyncPeriod: func() *time.Duration {
			sp := conf.GetDuration(conf.SyncPeriod)
			return &sp
		}(),
	})
	if err != nil {
		return nil, err
	}
	var mapper meta.RESTMapper
	mapper, err = apiutil.NewDiscoveryRESTMapper(mgr.GetConfig())
	var noCacheClient client.Client
	noCacheClient, err = client.New(mgr.GetConfig(), client.Options{
		Scheme: cfg.Scheme,
		Mapper: mapper,
	})
	if err != nil {
		return nil, err
	}
	cfg.NoCacheClient = noCacheClient
	cfg.Client = mgr.GetClient()
	ecs := conf.GetSlice("enabled-controllers")
	return &ReconcilerManager{
		config:             cfg,
		mgr:                mgr,
		enabledControllers: ecs,
		logger:             ctrl.Log.WithName("controller-manager"),
	}, nil
}

type ReconcilersBuilder []func(*ReconcilerManager) error

func (rb *ReconcilersBuilder) Register(funcs ...func(*ReconcilerManager) error) {
	for _, f := range funcs {
		*rb = append(*rb, f)
	}
}

func (rb *ReconcilersBuilder) AddToManager(rmgr *ReconcilerManager) error {
	for _, f := range *rb {
		if err := f(rmgr); err != nil {
			return err
		}
	}
	return nil
}
