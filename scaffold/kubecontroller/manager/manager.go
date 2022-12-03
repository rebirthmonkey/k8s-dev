package manager

import (
	"fmt"
	"os"

	"github.com/rebirthmonkey/go/pkg/log"
	"github.com/rebirthmonkey/k8s-dev/pkg/reconcilermgr"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	demov1 "github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/apis/demo/v1"
	"github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/manager/reconcilers/at"
	"github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/manager/reconcilers/dummy"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
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

	if err := (&at.AtReconciler{
		Client: preparedReconcilerMgr.GetClient(),
		Scheme: preparedReconcilerMgr.GetScheme(),
	}).SetupWithManager(preparedReconcilerMgr.Manager); err != nil {
		log.Error("unable to create controller AT")
		fmt.Println(err)
		os.Exit(1)
	}

	if err := (&dummy.DummyReconciler{
		Client: preparedReconcilerMgr.GetClient(),
		Scheme: preparedReconcilerMgr.GetScheme(),
	}).SetupWithManager(preparedReconcilerMgr.Manager); err != nil {
		log.Error("unable to create controller Dummy")
		fmt.Println(err)
		os.Exit(1)
	}

	return &PreparedManager{
		PreparedReconcilerManager: preparedReconcilerMgr,
	}
}

func (mgr *PreparedManager) Run() error {
	log.Info("[PreparedManager] Run")

	if err := mgr.PreparedReconcilerManager.Run(); err != nil {
		log.Error("Error occurred while controller manager is running")
	}

	return nil
}
