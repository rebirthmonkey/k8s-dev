package reconcilerapp

import (
	"golang.org/x/sync/errgroup"
	"os"

	"github.com/rebirthmonkey/go/pkg/gin"
	"github.com/rebirthmonkey/go/pkg/log"
	"k8s.io/apimachinery/pkg/runtime"

	amgrregistry "github.com/rebirthmonkey/k8s-dev/pkg/apiextmgr/registry"
	k8sclient "github.com/rebirthmonkey/k8s-dev/pkg/k8s/client"
	"github.com/rebirthmonkey/k8s-dev/pkg/reconcilermgr"
	rmgrregistry "github.com/rebirthmonkey/k8s-dev/pkg/reconcilermgr/registry"
)

var (
	scheme = runtime.NewScheme()
)

type Manager struct {
	*reconcilermgr.ReconcilerManager
	ginServer *gin.Server
}

type PreparedManager struct {
	*reconcilermgr.PreparedReconcilerManager
	preparedGinServer *gin.PreparedServer
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
	preparedGinServer := mgr.ginServer.PrepareRun()

	return &PreparedManager{
		PreparedReconcilerManager: preparedReconcilerMgr,
		preparedGinServer:         preparedGinServer,
	}
}

func (pmgr *PreparedManager) Run() error {
	log.Info("[PreparedManager] Run")

	rmgrregistry.AddToManager(pmgr.ReconcilerManager)
	if err := pmgr.ReconcilerManager.Setup(); err != nil {
		log.Errorf("[PreparedManager] Failed to setup reconcilers", err)
		os.Exit(1)
	}

	var eg errgroup.Group

	eg.Go(func() error {
		if err := pmgr.PreparedReconcilerManager.Run(); err != nil {
			log.Error("[PreparedManager] Error occurred while controller manager is running")
			return err
		}
		return nil
	})

	if pmgr.APIServerEnabled {
		eg.Go(func() error {
			k8scli := k8sclient.NewClientsManager(pmgr.GetScheme(), "", "")
			amgrregistry.AddToManager(pmgr.preparedGinServer, k8scli)
			if err := pmgr.preparedGinServer.Run(); err != nil {
				log.Error("[PreparedManager] Error occurred while Gin server is running")
				return err
			}

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		log.Fatal(err.Error())
		return err
	}

	return nil
}
