package reconcilerapp

import (
	"os"

	"github.com/rebirthmonkey/go/pkg/gin"
	"github.com/rebirthmonkey/go/pkg/log"
	"golang.org/x/sync/errgroup"
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
	*gin.PreparedServer
	K8sclients k8sclient.Clients
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
	k8sclients := k8sclient.NewClientsManager(scheme, "", "")

	return &PreparedManager{
		PreparedReconcilerManager: preparedReconcilerMgr,
		PreparedServer:            preparedGinServer,
		K8sclients:                k8sclients,
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
			amgrregistry.AddToManager(pmgr.PreparedServer, pmgr.K8sclients)
			if err := pmgr.PreparedServer.Run(); err != nil {
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
