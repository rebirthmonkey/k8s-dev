package kubecontroller

import (
	"github.com/rebirthmonkey/go/pkg/log"
	rmgrregistry "github.com/rebirthmonkey/k8s-dev/pkg/reconcilermgr/registry"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/runtime"
	"os"

	"github.com/rebirthmonkey/k8s-dev/pkg/apiextmgr"
	amgrregistry "github.com/rebirthmonkey/k8s-dev/pkg/apiextmgr/registry"
	k8sclient "github.com/rebirthmonkey/k8s-dev/pkg/k8s/client"
	"github.com/rebirthmonkey/k8s-dev/pkg/reconcilermgr"
)

var (
	scheme = runtime.NewScheme()
)

type KubeController struct {
	*reconcilermgr.ReconcilerManager
	*apiextmgr.APIExtManager
}

type PreparedKubeController struct {
	*reconcilermgr.PreparedReconcilerManager
	*apiextmgr.PreparedAPIExtManager
	K8sclients k8sclient.Clients
}

func NewKubeController(opts *Options) (*KubeController, error) {
	log.Info("[KubeController] New")

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
func (mgr *KubeController) PrepareRun() *PreparedKubeController {
	log.Info("[KubeController] PrepareRun")

	preparedReconcilerMgr := mgr.ReconcilerManager.PrepareRun(scheme)
	preparedAPIExtMgr := mgr.APIExtManager.PrepareRun()
	k8sclients := k8sclient.NewClientsManager(scheme, "", "")

	return &PreparedKubeController{
		PreparedReconcilerManager: preparedReconcilerMgr,
		PreparedAPIExtManager:     preparedAPIExtMgr,
		K8sclients:                k8sclients,
	}
}

func (pmgr *PreparedKubeController) Run() error {
	log.Info("[PreparedKubeController] Run")

	var eg errgroup.Group

	eg.Go(func() error {
		rmgrregistry.AddToManager(pmgr.ReconcilerManager)
		if err := pmgr.ReconcilerManager.Setup(); err != nil {
			log.Errorf("[PreparedKubeController] Failed to setup reconcilers: %s", err)
			os.Exit(1)
		}

		if err := pmgr.PreparedReconcilerManager.Run(); err != nil {
			log.Error("[PreparedKubeController] Error occurred while controller manager is running")
			return err
		}
		return nil
	})

	if pmgr.APIServerEnabled {
		eg.Go(func() error {
			amgrregistry.AddToManager(pmgr.PreparedServer, pmgr.K8sclients)
			if err := pmgr.PreparedServer.Run(); err != nil {
				log.Error("[PreparedAPIExtManager] Error occurred while Gin server is running")
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
