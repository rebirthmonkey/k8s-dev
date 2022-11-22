package registry

import (
	"github.com/rebirthmonkey/k8s-dev/pkg/manager"
)

var (
	ReconcilersBuilder manager.ReconcilersBuilder
)

func Register(funcs ...func(*manager.ReconcilerManager) error) {
	ReconcilersBuilder.Register(funcs...)
}

func AddToManager(rmgr *manager.ReconcilerManager) {
	ReconcilersBuilder.AddToManager(rmgr)
}
