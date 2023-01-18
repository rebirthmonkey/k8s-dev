package reconcilerapp

import (
	"os"

	"github.com/rebirthmonkey/go/pkg/log"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	demov1 "github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/apis/demo/v1"
	"github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/pkg/reconcilermgr"
	"github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/pkg/registry"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(corev1.AddToScheme(scheme)) // because we will use Pod.
	utilruntime.Must(demov1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

type Manager struct {
	*reconcilermgr.ReconcilerManager
}

type PreparedManager struct {
	*reconcilermgr.PreparedReconcilerManager
}

func NewManager(opts *Options) (*Manager, error) {
	log.Info("[Manager] New")

	config := NewConfig()
	if err := opts.ApplyTo(config); err != nil {
		return nil, err
	}

	mgrInstance, err := config.Complete().New()
	if err != nil {
		return nil, err
	}

	return mgrInstance, nil
}

// PrepareRun creates a running manager instance after complete initialization.
func (mgr *Manager) PrepareRun() *PreparedManager {
	log.Info("[Manager] PrepareRun")

	preparedReconcilerMgr := mgr.ReconcilerManager.PrepareRun(scheme)

	//if err := (&at.Reconciler{
	//	Client: preparedReconcilerMgr.GetClient(),
	//	Scheme: preparedReconcilerMgr.GetScheme(),
	//}).SetupWithManager(preparedReconcilerMgr.Manager); err != nil {
	//	log.Error("unable to create controller AT")
	//	fmt.Println(err)
	//	os.Exit(1)
	//}

	//if err := (&dummy.DummyReconciler{
	//	Client: preparedReconcilerMgr.GetClient(),
	//	Scheme: preparedReconcilerMgr.GetScheme(),
	//}).SetupWithManager(preparedReconcilerMgr.Manager); err != nil {
	//	log.Error("unable to create controller Dummy")
	//	fmt.Println(err)
	//	os.Exit(1)
	//}

	return &PreparedManager{
		PreparedReconcilerManager: preparedReconcilerMgr,
	}
}

func (pmgr *PreparedManager) Run() error {
	log.Info("[PreparedManager] Run")

	registry.AddToManager(pmgr.ReconcilerManager)
	if err := pmgr.ReconcilerManager.Setup(); err != nil {
		log.Errorf("Failed to setup reconcilers", err)
		os.Exit(1)
	}

	if err := pmgr.PreparedReconcilerManager.Run(); err != nil {
		log.Error("Error occurred while controller manager is running")
	}

	return nil
}
