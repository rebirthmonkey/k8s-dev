package reconcilermgr

import (
	"context"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	"github.com/rebirthmonkey/go/pkg/log"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	crmgr "sigs.k8s.io/controller-runtime/pkg/manager"
)

type ReconcilerManager struct {
	MetricsBindAddress     string
	HealthProbeBindAddress string
	Concurrence            int
	APIServerURL           string
	Kubeconfig             string

	crmgr.Manager
	client.Client
	context.Context

	setupers           []ReconcilerSetuper
	enabledControllers []string
}

type PreparedReconcilerManager struct {
	*ReconcilerManager
}

func (rmgr *ReconcilerManager) With(setupers ...ReconcilerSetuper) {
	for _, setuper := range setupers {
		rmgr.setupers = append(rmgr.setupers, setuper)
		//res := setuper.For()
		//cfg, ok := apis.ResourceMetadatas[res]
		//if ok && cfg.NoPortable && rmgr.config.Portable {
		//	rmgr.logger.Info(fmt.Sprintf("Ignoring controller for non-portable resource %s", res))
		//	continue
		//}
		//if fns.IsEmpty(rmgr.enabledControllers) || fns.Contains(rmgr.enabledControllers, res) {
		//	rmgr.setupers = append(rmgr.setupers, setuper)
		//	rmgr.logger.Info(fmt.Sprintf("Add controller for %s to manager", res))
		//} else {
		//	rmgr.logger.Info(fmt.Sprintf("Controller for %s disabled", res))
		//}
	}
}

func (rmgr *ReconcilerManager) Setup() error {
	mgr := rmgr.Manager
	for _, setuper := range rmgr.setupers {
		if err := setuper.SetupWithManager(mgr); err != nil {
			return err
		}
	}
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		log.Errorf("[ReconcilerManager] unable to set up health check", err)
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		log.Errorf("[ReconcilerManager] unable to set up ready check", err)
		os.Exit(1)
	}
	return nil
}

func (rmgr *ReconcilerManager) GetDefaultConcurrence() int {
	return rmgr.Concurrence
}

func (rmgr *ReconcilerManager) GetClient() client.Client {
	return rmgr.Client
}

func (rmgr *ReconcilerManager) GetContext() context.Context {
	return rmgr.Context
}

func (rmgr *ReconcilerManager) GetScheme() *runtime.Scheme {
	return rmgr.Manager.GetScheme()
}

func (rmgr *ReconcilerManager) PrepareRun(scheme *runtime.Scheme) *PreparedReconcilerManager {
	log.Info("[ReconcilerManager] PrepareRun")

	if rmgr.Kubeconfig == "" {
		rmgr.Kubeconfig = clientcmd.RecommendedHomeFile
	}

	config, err := clientcmd.BuildConfigFromFlags("", rmgr.Kubeconfig)

	mgr, err := ctrl.NewManager(config, ctrl.Options{
		Scheme:           scheme,
		Port:             9443,
		LeaderElection:   false,
		LeaderElectionID: "465bd3f6.wukong.com",
	})
	if err != nil {
		log.Error("[ReconcilerManager] unable to start manager")
		os.Exit(1)
	}

	rmgr.Manager = mgr
	rmgr.Client = mgr.GetClient()
	rmgr.Context = ctrl.SetupSignalHandler()
	return &PreparedReconcilerManager{
		ReconcilerManager: rmgr,
	}
}

func (prmgr *PreparedReconcilerManager) Run() error {
	log.Info("[PreparedReconcilerManager] Run")

	mgr := prmgr.Manager
	go func() {
		log.Info("[PreparedReconcilerManager] Wait until cache synchronized")
		mgr.GetCache().WaitForCacheSync(prmgr.GetContext())
		log.Info("[PreparedReconcilerManager] Cache synchronized, execute AfterCacheSync hook for all controllers")
		for _, setuper := range prmgr.setupers {
			setuper.AfterCacheSync(mgr)
		}
	}()

	if err := prmgr.Manager.Start(prmgr.GetContext()); err != nil {
		log.Error("problem running manager")
		os.Exit(1)
	}

	return nil
}