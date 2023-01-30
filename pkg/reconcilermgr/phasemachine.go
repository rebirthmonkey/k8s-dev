package reconcilermgr

import (
	"context"
	"github.com/rebirthmonkey/k8s-dev/pkg/apis"
	"github.com/rebirthmonkey/k8s-dev/pkg/pm"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type pmReconcilerShim struct {
	config       *Config
	resource     client.Object
	phaseMachine pm.Interface
	filter       string

	client.Client
}

func (r *pmReconcilerShim) AfterCacheSync(mgr ctrl.Manager) error {
	return nil
}

func (r *pmReconcilerShim) KeyFilter(filter string) {
	r.filter = filter
}

func (r *pmReconcilerShim) For() string {
	resource, _ := apis.GetResourceMetadataByResourceType(r.resource)
	return resource
}

func (r *pmReconcilerShim) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	//return r.phaseMachine.Reconcile(context.WithValue(ctx, reconcilers.ContextKeyClient, r.config.NoCacheClient), req)
	return r.phaseMachine.Reconcile(context.WithValue(ctx, apis.ContextKeyClient, r.Client), req)
}

func (r *pmReconcilerShim) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(r.resource).
		Complete(r)
}

func (r *pmReconcilerShim) predicate(obj client.Object) bool {
	//if !reconcilers.FilterObject(r.filter, obj) {
	//	return false
	//}
	return !r.phaseMachine.IsTerminalPhase(r.phaseMachine.GetPhase(obj)) || obj.GetDeletionTimestamp() != nil
}

func (rmgr *ReconcilerManager) WithPhaseMachine(resource client.Object, phaseMachine pm.Interface, configs ...*Config) {
	//config := rmgr.config
	//if len(configs) > 0 {
	//	config = configs[0]
	//}
	reconciler := &pmReconcilerShim{
		//config:       config,
		Client:       rmgr.GetClient(),
		resource:     resource,
		phaseMachine: phaseMachine,
	}
	rmgr.setupers = append(rmgr.setupers, reconciler)
	//res := reconciler.For()
	//cfg, ok := apis.ResourceMetadatas[res]
	//if ok && cfg.NoPortable && rmgr.config.Portable {
	//	log.Info(fmt.Sprintf("Ignoring phasemachine controller for non-portable resource %s", res))
	//	return
	//}
	//if fns.IsEmpty(rmgr.enabledControllers) || fns.Contains(rmgr.enabledControllers, res) {
	//	rmgr.setupers = append(rmgr.setupers, reconciler)
	//	rmgr.logger.Info(fmt.Sprintf("Add phasemachine controller for %s to manager", res))
	//} else {
	//	rmgr.logger.Info(fmt.Sprintf("Phasemachine controller for %s disabled", res))
	//}
}
