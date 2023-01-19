package reconcilermgr

import (
	ctrl "sigs.k8s.io/controller-runtime"
)

type ReconcilerSetuper interface {
	// SetupWithManager register the controller in manager
	SetupWithManager(mgr ctrl.Manager) error
	// For what resource the controller interests in
	For() string
	// KeyFilter if set, only matched (by key, aka namespace/name ) resource instance will be processed by the controller
	KeyFilter(filter string)
	// AfterCacheSync hook that will be executed after cache sync
	AfterCacheSync(mgr ctrl.Manager) error
}
