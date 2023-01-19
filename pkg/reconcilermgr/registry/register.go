package registry

import (
	"github.com/rebirthmonkey/k8s-dev/pkg/reconcilermgr"
)

var (
	ReconcilersBuilder reconcilermgr.ReconcilersBuilder
)

func Register(funcs ...func(*reconcilermgr.ReconcilerManager) error) {
	ReconcilersBuilder.Register(funcs...)
}

func AddToManager(rmgr *reconcilermgr.ReconcilerManager) {
	ReconcilersBuilder.AddToManager(rmgr)
}
