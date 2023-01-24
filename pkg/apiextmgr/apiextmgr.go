package apiextmgr

import (
	"github.com/rebirthmonkey/go/pkg/gin"
	"github.com/rebirthmonkey/go/pkg/log"
)

type APIExtManager struct {
	*gin.Server
}

type PreparedAPIExtManager struct {
	*gin.PreparedServer
}

func NewManager(opts *Options) (*APIExtManager, error) {
	log.Info("[APIExtManager] New")

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
func (mgr *APIExtManager) PrepareRun() *PreparedAPIExtManager {
	log.Info("[APIExtManager] PrepareRun")

	preparedServer := mgr.Server.PrepareRun()

	return &PreparedAPIExtManager{
		PreparedServer: preparedServer,
	}
}

func (pmgr *PreparedAPIExtManager) Run() error {
	log.Info("[APIExtPreparedManager] Run")

	if err := pmgr.PreparedServer.Run(); err != nil {
		log.Error("[PreparedAPIExtManager] Error occurred while server is running")
		return err
	}
	return nil
}
