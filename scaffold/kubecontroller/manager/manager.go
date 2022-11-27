package manager

import (
	"fmt"
	"github.com/rebirthmonkey/go/pkg/log"
	"github.com/rebirthmonkey/k8s-dev/pkg/reconcilermgr"
	"github.com/rebirthmonkey/k8s-dev/pkg/version"
	demov1 "github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/apis/demo/v1"
	"github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/manager/reconcilers/at"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"os"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(demov1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

type Manager struct {
	ReconcilerMgr *reconcilermgr.ReconcilerManager
}

type PreparedManager struct {
	PreparedReconcilerMgr *reconcilermgr.PreparedReconcilerManager
}

func NewManager(opts *Options) (*Manager, error) {
	log.Info(fmt.Sprintf("Kubecontroller version: %s", version.Info()))

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

	preparedManager := &PreparedManager{
		PreparedReconcilerMgr: mgr.ReconcilerMgr.PrepareRun(scheme),
	}

	if err := (&at.AtReconciler{
		Client: mgr.ReconcilerMgr.Mgr.GetClient(),
		Scheme: mgr.ReconcilerMgr.Mgr.GetScheme(),
	}).SetupWithManager(mgr.ReconcilerMgr.Mgr); err != nil {
		log.Error("unable to create controller AT")
		fmt.Println(err)
		os.Exit(1)
	}

	return preparedManager
}

func (mgr *PreparedManager) Run() error {
	log.Info("[PreparedManager] Run")

	if err := mgr.Run(); err != nil {
		log.Error("Error occurred while controller manager is running")
	}

	return nil
}
