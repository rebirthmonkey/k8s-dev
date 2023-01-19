package reconcilerapp

import (
	"os"

	"github.com/rebirthmonkey/go/pkg/log"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/rebirthmonkey/k8s-dev/pkg/reconcilermgr"
	"github.com/rebirthmonkey/k8s-dev/pkg/reconcilermgr/registry"
)

var (
	scheme = runtime.NewScheme()
)

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

	return &PreparedManager{
		PreparedReconcilerManager: preparedReconcilerMgr,
	}
}

func (pmgr *PreparedManager) Run() error {
	log.Info("[PreparedManager] Run")

	registry.AddToManager(pmgr.ReconcilerManager)
	if err := pmgr.ReconcilerManager.Setup(); err != nil {
		log.Errorf("[PreparedManager] Failed to setup reconcilers", err)
		os.Exit(1)
	}

	if err := pmgr.PreparedReconcilerManager.Run(); err != nil {
		log.Error("[PreparedManager] Error occurred while controller manager is running")
	}

	return nil
}
